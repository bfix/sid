/*
 * Cover server communication to disguise client communication with SID.
 *
 * (c) 2012 Bernd Fix   >Y<
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or (at
 * your option) any later version.
 *
 * This program is distributed in the hope that it will be useful, but
 * WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the GNU
 * General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package sid

///////////////////////////////////////////////////////////////////////
// Import external declarations.

import (
	"bufio"
	"bytes"
	"gospel/logger"
	"net"
	"strconv"
	"strings"
)

///////////////////////////////////////////////////////////////////////
// Constants

const (
	//-----------------------------------------------------------------
	// Request modes
	//-----------------------------------------------------------------
	REQ_UNKNOWN = iota // unknown request type
	REQ_GET            // "get resource" request
	REQ_POST           // "post data" request

	//-----------------------------------------------------------------
	// Request parser states
	//-----------------------------------------------------------------
	RS_HDR          = iota // parsing header (initial state)
	RS_HDR_COMPLETE        // parsing header completed
	RS_CONTENT             // parsing content (POST request)
	RS_DONE                // parsing complete
)

///////////////////////////////////////////////////////////////////////
/*
 * State information for cover server connections.
 */
type State struct {
	//-----------------------------------------------------------------
	// Request state
	//-----------------------------------------------------------------
	ReqMode          int    // request type (GET, POST)
	ReqState         int    // request processing (HDR,APPEND)
	ReqResource      string // resource requested by client
	ReqBoundaryIn    string // POST boundary separator (incoming,client)
	ReqBoundaryOut   string // POST boundary separator (outgoing,cover)
	ReqCoverPost     []byte // cover POST content
	ReqCoverPostPos  int    // index into POST content
	ReqUpload        bool   // parsing client document upload?
	ReqUploadData    string // client document data
	ReqContentLength int    // content length of request

	//-----------------------------------------------------------------
	// Response state
	//-----------------------------------------------------------------
	RespPending string   // pending (HTML) response
	RespEnc     string   // response encoding
	RespMode    int      // response mode (0=init,1=hdr,2=body)
	RespSize    int      // expected response size (total length)
	RespType    string   // format identifier for response content (mime type)
	RespHdr     *TagList // list of tags for header
	RespTags    *TagList // list of tags to be included in response body
	RespXtra    *TagList // list of tags with extra information (e.g. hidden input fields)

	//-----------------------------------------------------------------
	// Shared additional data
	//-----------------------------------------------------------------
	Data map[string]string // additional data
}

///////////////////////////////////////////////////////////////////////
/*
 * Cover server instance (stateful)
 */
type Cover struct {
	Name     string              // hostname of cover server
	Port     int                 // target port of cover server
	Protocol string              // HTTP/HTTPS protocol spec
	States   map[net.Conn]*State // state of active connections
	Posts    map[string]([]byte) // list of cover POST replacements

	HandleRequest func(*Cover, *State) (string, string) // Handle HTML request (w/ special cases)
	SyncCover     func(*Cover, *State)                  // synchronize cover content with response HTML
	FinalizeCover func(*Cover, *State) []byte           // Finalize cover content
}

///////////////////////////////////////////////////////////////////////
// Public methods for Cover instance

/*
 * Connect to cover server
 * @return net.Conn - connection to cover server (or nil)
 */
func (c *Cover) connect() net.Conn {
	// establish connection
	addr := c.Name + ":" + strconv.Itoa(c.Port)
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		// can't connect
		logger.Printf(logger.ERROR, "[sid.cover] failed to connect to cover server: %s\n", err.Error())
		return nil
	}
	logger.Println(logger.INFO, "[sid.cover] connected to cover server...")

	// allocate state information and add to state list
	// initialize struct with default data
	c.States[conn] = &State{
		//-------------------------------------------------------------
		// Request state
		//-------------------------------------------------------------
		ReqMode:         REQ_UNKNOWN,
		ReqState:        RS_HDR,
		ReqResource:     "",
		ReqBoundaryIn:   "",
		ReqBoundaryOut:  "",
		ReqCoverPost:    nil,
		ReqCoverPostPos: 0,
		ReqUpload:       false,
		ReqUploadData:   "",

		//-------------------------------------------------------------
		// Response state
		//-------------------------------------------------------------
		RespPending: "",
		RespEnc:     "",
		RespMode:    0,
		RespSize:    0,
		RespType:    "text/html",
		RespHdr:     NewTagList(),
		RespTags:    NewTagList(),
		RespXtra:    NewTagList(),

		//-------------------------------------------------------------
		// Additional data
		//-------------------------------------------------------------
		Data: make(map[string]string),
	}
	return conn
}

//---------------------------------------------------------------------
/*
 * Disconnect from cover server: Since this instance may be shared
 * across multiple HTTP sessions, this is the place for a cover server
 * clean-up to avoid cluttering of cover instance data.
 * @param conn net.Conn - client connection
 */
func (c *Cover) disconnect(conn net.Conn) {
	delete(c.States, conn)
	conn.Close()
}

//---------------------------------------------------------------------
/*
 * Get state associated with given connection.
 * @param conn net.Conn - client connection
 * @return *state - reference to state instance
 */
func (c *Cover) GetState(conn net.Conn) *State {
	if s, ok := c.States[conn]; ok {
		return s
	}
	return nil
}

//---------------------------------------------------------------------
/*
 * get cover site POST content for given boundary id.
 * @param id string - boundary id (key used to store POST content)
 * @return []byte - POST content
 */
func (c *Cover) GetPostContent(id string) []byte {
	if post, ok := c.Posts[id]; ok {
		// delete POST from list
		delete(c.Posts, id)
		return post
	}
	return nil
}

//---------------------------------------------------------------------
/*
 * Transform client request: this is supposed to work on fragmented
 * requests if necessary (currently not really supported)
 * @param s *state - reference to state information
 * @param data []byte - request data from client
 * @param num int - length of request in bytes
 * @return []byte - transformed request (send to cover server)
 */
func (c *Cover) xformReq(s *State, data []byte, num int) []byte {

	inStr := string(data[0:num])
	logger.Printf(logger.DBG_HIGH, "[sid.cover] %d bytes received from client.\n", num)
	logger.Println(logger.DBG_ALL, "[sid.cover] Incoming request:\n"+inStr+"\n")

	// assemble transformed request
	rdr := bufio.NewReader(strings.NewReader(inStr))
	req := ""
	hasContentEncoding := false // expected content encoding defined?
	//hasTransferEncoding := false		// expected transfer encoding defined?
	mime := "text/html"  // expected content type
	targetHost := c.Name // request resource from this host (default)
	balance := 0         // balance between incoming and outgoing information

	// use identical line break sequence	
	lb := "\r\n"
	if strings.Index(inStr, lb) == -1 {
		lb = "\n"
	}
	for s.ReqState == RS_HDR {
		// get next line (terminated by line break)
		b, broken, _ := rdr.ReadLine()
		if b == nil || len(b) == 0 {
			if !broken {
				s.ReqState = RS_HDR_COMPLETE
			}
			break
		}
		line := strings.TrimRight(string(b), "\r\n")

		// transform request data
		switch {
		//---------------------------------------------------------
		// POST command: upload document
		// This command triggers the upload of a document to SID
		// that is covered by an upload to the cover site of the
		// same length.
		//---------------------------------------------------------
		case strings.HasPrefix(line, "POST "):
			// split line into parts
			parts := strings.Split(line, " ")
			logger.Printf(logger.DBG_HIGH, "[sid.cover] POST '%s'\n", parts[1])

			// POST uri encodes the key to the cover POST content and the
			// target POST URL
			elem := strings.Split(parts[1], "/")
			s.ReqBoundaryOut = elem[1]
			uri := ""
			for i := 2; i < len(elem); i++ {
				uri += "/" + elem[i]
			}

			// try to get pre-defined cover content. if no cover content
			// has been constructed yet, the 'reqCoverPost' will contain
			// nil and the content is constructed later when the content
			// length of the incoming request is known.
			s.ReqCoverPost = c.GetPostContent(s.ReqBoundaryOut)
			s.ReqCoverPostPos = 0

			// if URI refers to an external host, split into
			// host reference and resource specification
			if pos := strings.Index(uri, "://"); pos != -1 {
				rem := string(uri[pos+3:])
				pos = strings.Index(rem, "/")
				if pos != -1 {
					targetHost = rem[0:pos]
					uri = rem[pos:]
					logger.Printf(logger.INFO, "[sid.cover] URI split: '%s', '%s'\n", targetHost, uri)
				} else {
					logger.Printf(logger.WARN, "[sid.cover] URI split failed on '%s'\n", uri)
				}
			} else {
				targetHost = c.Name
			}

			// assemble new POST request
			s.ReqResource = uri
			req += "POST " + uri + " HTTP/1.0" + lb
			s.ReqMode = REQ_POST

			// keep balance
			balance += (len(parts[1]) - len(uri))

		//---------------------------------------------------------
		// GET command: request resource
		// If the requested resource identifier is a translated
		// entry, we need to translate that back into its original
		// form. Translated entries start with "/&".
		// It is assumed, that a "GET" line is one of the first
		// lines in a request and therefore never fragmented.
		// N.B.: We also force HTTP/1.0 to ensure that no
		// chunking is used by the server (easier parsing).
		//---------------------------------------------------------
		case strings.HasPrefix(line, "GET "):
			// split line into parts
			parts := strings.Split(line, " ")
			logger.Printf(logger.DBG_HIGH, "[sid.cover] resource='%s'\n", parts[1])

			// perform translation (if required)
			uri := translateURI(parts[1])
			logger.Printf(logger.INFO, "[sid.cover] URI translation: '%s' => '%s'\n", parts[1], uri)

			// if URI refers to an external host, split into
			// host reference and resource specification
			if pos := strings.Index(uri, "://"); pos != -1 {
				rem := string(uri[pos+3:])
				pos = strings.Index(rem, "/")
				if pos != -1 {
					targetHost = rem[0:pos]
					uri = rem[pos:]
					logger.Printf(logger.INFO, "[sid.cover] URI split: '%s', '%s'\n", targetHost, uri)
				} else {
					logger.Printf(logger.WARN, "[sid.cover] URI split failed on '%s'\n", uri)
				}
			} else {
				targetHost = c.Name
			}

			// assemble new resource request
			s.ReqResource = uri
			req += "GET " + uri + " HTTP/1.0" + lb
			s.ReqMode = REQ_GET

			// keep balance
			balance += (len(parts[1]) - len(uri))

		//---------------------------------------------------------
		// Host reference: change to hostname of cover server
		// This translation may leed to unbalanced request sizes;
		// the balance will be equalled in a later line
		// It is assumed, that a "Host:" line is one of the first
		// lines in a request and therefore never fragmented.
		//---------------------------------------------------------
		case strings.HasPrefix(line, "Host: "):
			// split line into parts
			parts := strings.Split(line, " ")
			// replace hostname reference 
			logger.Printf(logger.DBG_HIGH, "[sid.cover] Host replaced with '%s'\n", targetHost)
			req += "Host: " + targetHost + lb
			// keep track of balance
			balance += (len(parts[1]) - len(targetHost))

		//---------------------------------------------------------
		// try to get balance straight on language header line:
		// "Accept-Language: de-de,de;q=0.8,en-us;q=0.5,en;q=0.3"
		//---------------------------------------------------------
		//case s.ReqBalance != 0 && strings.HasPrefix (line, "Accept-Language: "):
		// @@@TODO: Is this the right place to balance the translation? 

		//---------------------------------------------------------
		// Acceptable content encoding: we only want plain HTML
		//---------------------------------------------------------
		case strings.HasPrefix(line, "Accept-Encoding: "):
			// split line into parts
			parts := strings.Split(line, " ")
			hasContentEncoding = true
			if mime == "text/html" && parts[1] != "identity" {
				// change to identity encoding for HTML pages
				repl := "Accept-Encoding: identity"
				balance += len(repl) - len(line)
				req += repl + lb
			} else {
				req += line + lb
			}
			/*
				//---------------------------------------------------------
				// Acceptable transfer encoding: we only want no chunking
				//---------------------------------------------------------
				case strings.HasPrefix (line, "Transfer-Encoding: "):
					// split line into parts
					parts := strings.Split (line, " ")
					hasTransferEncoding = true
					if mime == "text/html" && parts[1] != "identity" {
						// change to identity transfer for HTML pages
						repl := "Transfer-Encoding: identity"
						balance += len(repl) - len(line)
						req += repl + lb
					} else {
						req += line + lb
					}
			*/
		//---------------------------------------------------------
		// Expected content type
		//---------------------------------------------------------
		case strings.HasPrefix(line, "Content-Type: "):
			// split line into parts
			parts := strings.Split(line, " ")
			mime = parts[1]
			// remember boundary definition
			if s.ReqMode == REQ_POST {
				// strip "boundary="
				s.ReqBoundaryIn = string(parts[2][9:])
				logger.Println(logger.DBG_HIGH, "[sid.cover] Boundary="+s.ReqBoundaryIn)
				repl := parts[0] + " " + mime +
					" boundary=---------------------------" + s.ReqBoundaryOut
				balance += len(repl) - len(line)
				req += repl + lb
			} else {
				req += line + lb
			}

		//---------------------------------------------------------
		// Referer
		//---------------------------------------------------------
		case strings.HasPrefix(line, "Referer: "):
			repl := "Referer: " + c.Protocol + "://" + targetHost + "/"
			balance += len(repl) - len(line)
			req += repl + lb

		//---------------------------------------------------------
		// Connection
		//---------------------------------------------------------
		case strings.HasPrefix(line, "Connection: "):
			// split line into parts
			parts := strings.Split(line, " ")
			if parts[1] != "close" {
				repl := "Connection: close"
				balance += len(repl) - len(line)
				req += repl + lb
			} else {
				req += line + lb
			}

		//---------------------------------------------------------
		// Keep-Alive:
		//---------------------------------------------------------
		case strings.HasPrefix(line, "Keep-Alive: "):
			// don't add spec
			balance -= len(line)

		//---------------------------------------------------------
		// Content-Length
		//---------------------------------------------------------
		case strings.HasPrefix(line, "Content-Length: "):
			// do we have a pre-defined cover content?
			if s.ReqCoverPost == nil || s.ReqCoverPost[0] == '!' {
				// split line into parts
				parts := strings.Split(line, " ")
				// get incoming content length
				s.ReqContentLength, _ = strconv.Atoi(parts[1])
				// construct/expand cover content for given size
				s.ReqCoverPost = c.FinalizeCover(c, s)
			}
			// use cover content to construct a content length
			repl := "Content-Length: " + strconv.Itoa(len(s.ReqCoverPost))
			balance += len(repl) - len(line)
			req += repl + lb

		//---------------------------------------------------------
		// add unchanged request lines. 
		//---------------------------------------------------------
		default:
			req += line
			if !broken {
				req += lb
			}
		}
	}

	// check for completed header in this pass
	if s.ReqState == RS_HDR_COMPLETE {
		// add delimiting empty line
		req += lb

		// post-process header
		if mime == "text/html" {
			if !hasContentEncoding {
				// enforce identity encoding for HTML pages
				repl := "Accept-Encoding: identity"
				balance += len(repl)
				req += repl + lb
			}
			/*
				if !hasTransferEncoding {
					// enforce identity transfer for HTML pages
					repl := "Transfer-Encoding: identity"
					balance += len(repl)
					req += repl + lb
				}
			*/
		}

		if s.ReqMode == REQ_POST {
			// switch state			
			s.ReqState = RS_CONTENT
		} else {
			// we are done
			s.ReqState = RS_DONE
		}
	}

	// handle processing of request contents for POST requests
	if s.ReqState == RS_CONTENT {

		// parse data until end of request
		for {
			// get next line (terminated by line break)
			// and adjust number of bytes read
			b, _, err := rdr.ReadLine()
			if err != nil {
				break
			}
			line := strings.TrimRight(string(b), "\r\n")

			//logger.Println (logger.DBG_ALL, "[sid.cover] POST content: " + line + "\n")

			if !s.ReqUpload {
				// check for start of document
				if strings.Index(line, "name=\"file\";") != -1 {
					s.ReqUpload = true
					s.ReqUploadData = ""
				}
			} else {
				if strings.Index(line, s.ReqBoundaryIn) != -1 {
					s.ReqUpload = false
					PostprocessUploadData([]byte(s.ReqUploadData))
				}
				// we are uploading client data
				s.ReqUploadData += line + lb
			}
		}

		// build new request data
		binReq := []byte(req)
		copy(data, binReq)
		pos := len(binReq)
		count := num - pos

		// we have "count" bytes of response data to sent out
		start := s.ReqCoverPostPos
		total := len(s.ReqCoverPost)
		if start < total {
			end := start + count
			if end < total {
				end = total
			}
			s.ReqCoverPostPos = end
			if end > total {
				end = total
			}
			copy(data[pos:], s.ReqCoverPost[start:end])
			pos += (end - start)
		}

		// fill up with line breaks
		if pos < num {
			fill := ""
			for count = num - pos; count > 0; count-- {
				fill += "\n"
			}
			data = append(data[0:pos], []byte(fill)...)
			pos = num
		}

		outStr := string(data[0:pos])
		logger.Printf(logger.DBG_HIGH, "[sid.cover] %d bytes send to cover server.\n", pos)
		logger.Println(logger.DBG_ALL, "[sid.cover] Outgoing request:\n"+outStr+"\n")
		return data[0:pos]
	}

	// check for completed request processing
	if s.ReqState == RS_DONE {
		if balance != 0 {
			logger.Printf(logger.WARN, "[sid.cover] Unbalanced request: %d bytes diff\n", balance)
		}
	} else {
		// padding of request with line breaks (if assembled request is smaller; GET only)
		for num > len(req) && s.ReqMode == REQ_GET {
			req += "\n"
		}
		// return transformed request
		if num != len(req) {
			logger.Printf(logger.WARN, "[sid.cover] DIFF(request) = %d\n", len(req)-num)
		}
		logger.Printf(logger.DBG_ALL, "[sid.cover] Transformed request:\n"+req+"\n")
	}
	return []byte(req)
}

//---------------------------------------------------------------------
/*
 * Transform cover server response: Substitute absolute URLs in the
 * response to local links to be handled by the request translations.
 * @param s *state - reference to state information
 * @param data []byte - response data from cover server
 * @param num int - length of response data
 * @return []data - transformed response (send to client)
 */
func (c *Cover) xformResp(s *State, data []byte, num int) []byte {

	// log incoming packet
	inStr := string(data[0:num])
	logger.Printf(logger.DBG_HIGH, "[sid.cover] %d bytes received from cover server.\n", num)
	logger.Println(logger.DBG_ALL, "[sid.cover] Incoming response:\n"+inStr+"\n")

	// setup reader and response
	size := num
	rdr := bytes.NewBuffer(data[0:num])
	resp := ""

	// initial response package
	if s.RespMode == 0 {

		// use identical line break sequence	
		lb := "\r\n"
		if strings.Index(inStr, lb) == -1 {
			lb = "\n"
		}
		// start of new response encountered: parse header fields
	hdr:
		for {
			// get next line (terminated by line break)
			line, err := rdr.ReadString('\n')
			line = strings.TrimRight(line, "\n\r")
			if err != nil {
				// header is not complete: wait for next response fragment
				logger.Println(logger.WARN, "[sid.cover] Response header fragmented!")
				logger.Println(logger.DBG, "[sid.cover] Assembled response:\n"+resp)
				if size != len(resp) {
					logger.Printf(logger.WARN, "[sid.cover] DIFF(response:1) = %d\n", len(resp)-size)
				}
				return []byte(resp)
			}
			// check if header is available at all..
			if strings.HasPrefix(line, "<!") {
				logger.Println(logger.INFO, "[sid.cover] No response header found: "+line)
				break hdr
			}

			// parse response header
			switch {
			//-----------------------------------------------------
			// Header parsing complete
			//-----------------------------------------------------
			case len(line) == 0:
				// we have parsed the header; continue with body
				logger.Println(logger.DBG_ALL, "[sid.cover] Incoming response header:\n"+resp)
				// drop length encoding on gzip content
				break hdr

			//-----------------------------------------------------
			// Status line
			//-----------------------------------------------------
			case strings.HasPrefix(line, "HTTP/"):
				// split line into parts
				parts := strings.Split(line, " ")
				status, _ := strconv.Atoi(parts[1])
				logger.Printf(logger.DBG, "[sid.cover] response status: %d\n", status)
				if status != 200 {
					return data[:size]
				}

			//-----------------------------------------------------
			// Content-Type:
			//-----------------------------------------------------
			case strings.HasPrefix(line, "Content-Type: "):
				// split line into parts
				parts := strings.Split(line, " ")
				s.RespType = strings.TrimRight(parts[1], ";")
				logger.Println(logger.DBG_HIGH, "[sid.cover] response type: "+s.RespType)

			//-----------------------------------------------------
			// Content-Encoding:
			//-----------------------------------------------------
			case strings.HasPrefix(line, "Content-Encoding: "):
				// split line into parts
				parts := strings.Split(line, " ")
				s.RespEnc = parts[1]
				logger.Println(logger.DBG_HIGH, "[sid.cover] response encoding: "+s.RespEnc)

			//-----------------------------------------------------
			// location:
			//-----------------------------------------------------
			case strings.HasPrefix(line, "location: "):
				// split line into parts
				parts := strings.Split(line, " ")
				line = "location: " + translateURI(parts[1])
				logger.Println(logger.DBG_HIGH, "[sid.cover] changing location => "+line)
			}
			// assemble response
			resp += line + lb
		}
		// add delimiter line
		resp += lb
		// adjust remaining content size
		num -= len(resp)
	}

	// are we still in the initial response packet?	
	if s.RespMode == 0 {
		//-------------------------------------------------------------
		// (initial) HTML response		
		//-------------------------------------------------------------		
		if strings.HasPrefix(s.RespType, "text/html") {
			// start of a new HTML response. Use pre-defined HTM page
			// to initialize response.
			var coverId string = ""
			s.RespPending, coverId = c.HandleRequest(c, s)
			s.Data["CoverId"] = coverId
		}
		// switch to next mode
		s.RespMode = 1
	}

	switch {
	//-------------------------------------------------------------
	// assemble HTML response		
	//-------------------------------------------------------------		
	case strings.HasPrefix(s.RespType, "text/html"):
		// do content translation (collect resource tags)
		done := parseHTML(rdr, s.RespHdr, s.RespTags, s.RespXtra)
		// sync replacement body (cover content) if response has
		// been completely processed.
		if done {
			c.SyncCover(c, s)
		}
		// assemble header if required
		if s.RespMode == 1 && s.RespHdr.Count() > 0 {
			hdr := c.assembleHeader(s.RespHdr, num)
			resp += hdr
			num -= len(hdr)
			// handle HTML body
			s.RespMode = 2
		}
		// assemble HTML body
		resp += c.assembleBody(s, num, done)
		logger.Println(logger.DBG_ALL, "[sid.cover] Translated response:\n"+resp)
		// return response data
		if size != len(resp) {
			logger.Printf(logger.WARN, "[sid.cover] DIFF(response:2) = %d\n", len(resp)-size)
		}
		return []byte(resp)

	//-------------------------------------------------------------
	// Images: Images are considered harmless, so we simply
	// pass them back to the client.
	//-------------------------------------------------------------		
	case strings.HasPrefix(s.RespType, "image/"):
		logger.Println(logger.DBG, "[sid.cover] Image data passed to client")
		return data[0:size]

	//-------------------------------------------------------------
	// JavaScript: Simply replace any JavaScript content with
	// spaces (looks like the client browser has disabled
	// JavaScript).
	//-------------------------------------------------------------		
	case strings.HasPrefix(s.RespType, "application/x-javascript"):
		// padding to requested size
		for n := 0; n < num; n++ {
			resp += " "
		}
		// return response data
		logger.Println(logger.DBG, "[sid.cover] JavaScript scrubbed")
		if size != len(resp) {
			logger.Printf(logger.WARN, "[sid.cover] DIFF(response:3) = %d\n", len(resp)-size)
		}
		return []byte(resp)

	//-------------------------------------------------------------
	// CSS: Simply replace any style sheets with spaces. No image
	// references in CSS are parsed (looks like those are cached
	// resources to an eavesdropper)
	//-------------------------------------------------------------		
	case strings.HasPrefix(s.RespType, "text/css"):
		// padding to requested size
		for n := 0; n < num; n++ {
			resp += " "
		}
		// return response data
		logger.Println(logger.DBG, "[sid.cover] CSS scrubbed")
		if size != len(resp) {
			logger.Printf(logger.WARN, "[sid.cover] DIFF(response:4) = %d\n", len(resp)-size)
		}
		return []byte(resp)
	}

	//return untranslated response
	logger.Println(logger.ERROR, "[sid.cover] Unhandled response!")
	return data[0:size]
}

//=====================================================================
/*
 * Assemble a HTML body from the current state (like response header),
 * the resource list and a replacement body (addressed by the requested
 * resource path from state).
 * @param s *state - current state info
 * @param size int - target size of response
 * @param done bool - can we close the HTML
 * @return string - assembled HTML body
 */
func (c *Cover) assembleBody(s *State, size int, done bool) string {

	// check if requested size can hold HTML wrapper at all.
	if size < len(htmlIntro)+len(htmlOutro)+10 {
		return ""
	}
	// create HTML intro
	resp := htmlIntro
	size -= len(htmlIntro)

	// emit pending reponse data first
	pending := len(s.RespPending)
	logger.Printf(logger.DBG_ALL, "[sid.cover] assembleBody (%d) -- %d\n", size, pending)
	switch {
	case pending > size:
		resp = string(s.RespPending[0:size])
		s.RespPending = string(s.RespPending[size:])
		return resp
	case pending > 0:
		resp = s.RespPending
		size -= pending
		s.RespPending = ""
	}

	// add resources (if any are pending)
	for s.RespTags.Count() > 0 {
		// get next tag
		tag := s.RespTags.Get()
		if tag == nil {
			break
		}
		// translate tag for client
		inl := c.translateTag(tag)
		// check if we can add the tag?
		if len(inl) < size {
			// yes: add it to response
			resp += inl
			size -= len(inl)
		} else {
			// no: put it back
			s.RespTags.Put(tag)
			break
		}
	}

	// close HTML if possible
	if done {
		resp += htmlOutro
		size -= len(htmlOutro)
	}
	// we are done, but have still response data to transfer. Fill up
	// with padding sequence. 
	resp += padding(size)

	return resp
}

//=====================================================================
/*
 * Assemble a HTML header from the current state if there are header
 * links we need to reproduce.
 * @param tags *TagList - header tags
 * @param size int - max size of response
 * @return string - assembled header
 */
func (c *Cover) assembleHeader(tags *TagList, size int) string {

	// add header resources
	hdr := "<head>\n"
	for tags.Count() > 0 {
		// get next tag
		tag := tags.Get()
		if tag == nil {
			break
		}
		// translate tag for client
		inl := c.translateTag(tag) + "\n"
		// check if we can add the tag?
		if len(inl) < size {
			// yes: add it to response
			hdr += inl
			size -= len(inl)
		} else {
			// no: put it back
			logger.Printf(logger.WARN, "[sid.cover] can't add all header tags: %d are skipped\n", tags.Count()+1)
			break
		}
	}

	// close header
	hdr += "</head>\n"
	return hdr
}

//---------------------------------------------------------------------
/*
 * Translate tag source attribute: if the source specification is an
 * URI of the form "<scheme>://<server>/<path>/<to>/<resource...>" it
 * is transformed to an absolute path on on the sending server (that is
 * the SID instance) that can later be translated back to its original
 * form; it looks like "/&<scheme>/<server>/<path>/<to>/<resource...>"
 * @param tag *Tag - tag to be translated
 * @return string - translated tag
 */
func (c *Cover) translateTag(tag *Tag) string {

	if src, ok := tag.attrs["src"]; ok {
		// translate "src" attribute of tag
		trgt := translateURI(src)
		logger.Printf(logger.INFO, "[sid.cover] URI translation of '%s' => '%s'\n", src, trgt)
		tag.attrs["src"] = trgt
	} else if src, ok := tag.attrs["href"]; ok {
		// translate "href" attribute of tag
		trgt := translateURI(src)
		logger.Printf(logger.INFO, "[sid.cover] URI translation of '%s' => '%s'\n", src, trgt)
		tag.attrs["href"] = trgt
	} else {
		// failed to access reference attribute?!
		s := tag.String()
		logger.Println(logger.ERROR, "[sid.cover] Tag translation failed: "+s)
		return s
	}
	// return tag representation
	return tag.String()
}
