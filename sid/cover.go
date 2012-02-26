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

package main

///////////////////////////////////////////////////////////////////////
// Import external declarations.

import (
	"net"
	"strings"
	"strconv"
	"bytes"
	"bufio"
	"gospel/logger"
)

///////////////////////////////////////////////////////////////////////
// Constants

const (
	//-----------------------------------------------------------------
	// Request modes
	//-----------------------------------------------------------------
	REQ_UNKNOWN = iota	// unknown request type
	REQ_GET				// "get resource" request
	REQ_POST			// "post data" request
	
	//-----------------------------------------------------------------
	// Request parser states
	//-----------------------------------------------------------------
	RS_HDR = iota		// parsing header (initial state)
	RS_HDR_COMPLETE		// parsing header completed
	RS_CONTENT			// parsing content (POST request)
	RS_DONE				// parsing complete
)

///////////////////////////////////////////////////////////////////////
/*
 * State information for cover server connections.
 */
type State struct {
	//-----------------------------------------------------------------
	// Request state
	//-----------------------------------------------------------------
	reqMode			int			// request type (GET, POST)
	reqState		int			// request processing (HDR,APPEND)
	reqResource		string		// resource requested by client
	reqBoundaryIn	string		// POST boundary separator (incoming,client)
	reqBoundaryOut	string		// POST boundary separator (outgoing,cover)
	reqCoverPost	[]byte		// cover POST content
	reqCoverPostPos	int			// index into POST content
	reqUpload		bool		// parsing client document upload?
	reqUploadData	string		// client document data
	
	//-----------------------------------------------------------------
	// Response state
	//-----------------------------------------------------------------
	respPending		string		// pending (HTML) response
	respEnc			string		// response encoding
	respMode		int			// response mode (0=init,1=hdr,2=body)
	respSize		int			// expected response size (total length)
	respType		string		// format identifier for response content (mime type)
	respHdr			*TagList	// list of tags for header
	respTags		*TagList	// list of tags to be included in response body
}

///////////////////////////////////////////////////////////////////////
/*
 * Cover server instance (stateful)
 */
type Cover struct {
	server		string						// "host:port" of cover server
	states		map[net.Conn]*State			// state of active connections
	htmls		map[string]string			// HTML body replacements
	hdlr		UploadHandler				// handler of cover uploads
}

//---------------------------------------------------------------------
/*
 * Create a new cover server instance
 * @return *Cover - pointer to cover server instance
 */
func NewCover() *Cover {
	// currently we only have one cover server implementation
	return NewCvrImgon()
}

///////////////////////////////////////////////////////////////////////
// Public methods for Cover instance

/*
 * Connect to cover server
 * @return net.Conn - connection to cover server (or nil)
 */
func (c *Cover) connect () net.Conn {
	// establish connection
	conn,err := net.Dial ("tcp", c.server)
	if err != nil {
		// can't connect
		logger.Printf (logger.ERROR, "[cover] failed to connect to cover server: %s\n", err.String())
		return nil
	}
	logger.Println (logger.INFO, "[cover] connected to cover server...")
	
	// allocate state information and add to state list
	// initialize struct with default data
	c.states[conn] = &State {
		//-------------------------------------------------------------
		// Request state
		//-------------------------------------------------------------
		reqMode:			REQ_UNKNOWN,
		reqState:			RS_HDR,
		reqResource:		"",
		reqBoundaryIn:		"",
		reqBoundaryOut:		"",
		reqCoverPost:		nil,
		reqCoverPostPos:	0,
		reqUpload:			false,
		reqUploadData:		"",
		
		//-------------------------------------------------------------
		// Response state
		//-------------------------------------------------------------
		respPending:		"",
		respEnc:			"",
		respMode:			0,
		respSize:			0,
		respType:			"text/html",
		respHdr:			NewTagList(),
		respTags:			NewTagList(),
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
func (c *Cover) disconnect (conn net.Conn) {
	c.states[conn] = nil,false
	conn.Close()
}

//---------------------------------------------------------------------
/*
 * Get state associated with given connection.
 * @param conn net.Conn - client connection
 * @return *state - reference to state instance
 */
func (c *Cover) GetState (conn net.Conn) *State {
	if s,ok := c.states[conn]; ok {
		return s
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
func (c *Cover) xformReq (s *State, data []byte, num int) []byte {

	inStr := string(data[0:num])
	logger.Printf (logger.DBG_HIGH, "[cover] %d bytes received from client.\n", num)
	logger.Println (logger.DBG_ALL, "[cover] Incoming request:\n" + inStr + "\n")

	// assemble transformed request
	rdr := bufio.NewReader (strings.NewReader (inStr))
	req := ""
	hasContentEncoding := false		// expected content encoding defined?
	//hasTransferEncoding := false	// expected transfer encoding defined?
	mime := "text/html"				// expected content type
	targetHost := c.server			// request resource from this host (default)
	balance := 0					// balance between incoming and outgoing information
	
	// use identical line break sequence	
	lb := "\r\n"
	if strings.Index (inStr, lb) == -1 {
		lb = "\n"
	}
	for s.reqState == RS_HDR {
		// get next line (terminated by line break)
		b,broken,_ := rdr.ReadLine()
		if b == nil || len(b) == 0 {
			if !broken {
				s.reqState = RS_HDR_COMPLETE
			}
			break
		}
		line := strings.TrimRight (string(b), "\r\n")
		
		// transform request data
		switch {
			//---------------------------------------------------------
			// POST command: upload document
			// This command triggers the upload of a document to SID
			// that is covered by an upload to the cover site of the
			// same length.
			//---------------------------------------------------------
			case strings.HasPrefix (line, "POST "):
				// split line into parts
				parts := strings.Split (line, " ")
				logger.Printf (logger.DBG_HIGH, "[cover] POST '%s'\n", parts[1])
				
				// POST uri encodes the key to the cover POST content
				pos := strings.LastIndex (parts[1], "/")
				s.reqBoundaryOut = parts[1][pos+1:]
				uri := parts[1][0:pos]
				s.reqCoverPost = c.hdlr.getPostContent (s.reqBoundaryOut)
				s.reqCoverPostPos = 0
				
				// perform translation (if required)
				uri = translateURI (uri)
				logger.Printf (logger.INFO, "[cover] URI translation: '%s' => '%s'\n", parts[1], uri)
				
				// if URI refers to an external host, split into
				// host reference and resource specification
				if pos := strings.Index (uri, "://"); pos != -1 {
					pos = strings.Index (string(uri[pos+3:]), "/")
					if pos != -1 {
						targetHost = uri[0:pos]
						uri = uri[pos:]
						logger.Printf (logger.INFO, "[cover] URI split: '%s', '%s'\n", targetHost, uri)
					} else { 
						logger.Printf (logger.WARN, "[cover] URI split failed on '%s'\n", uri)
					}
				} else {
					targetHost = c.server
				}  

				// assemble new POST request
				s.reqResource = uri
				req += "POST " + uri + " HTTP/1.0" + lb
				s.reqMode = REQ_POST
				
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
			case strings.HasPrefix (line, "GET "):
				// split line into parts
				parts := strings.Split (line, " ")
				logger.Printf (logger.DBG_HIGH, "[cover] resource='%s'\n", parts[1])
				
				// perform translation (if required)
				uri := translateURI (parts[1])
				logger.Printf (logger.INFO, "[cover] URI translation: '%s' => '%s'\n", parts[1], uri)
				
				// if URI refers to an external host, split into
				// host reference and resource specification
				if pos := strings.Index (uri, "://"); pos != -1 {
					pos = strings.Index (string(uri[pos+3:]), "/")
					if pos != -1 {
						targetHost = uri[0:pos]
						uri = uri[pos:]
						logger.Printf (logger.INFO, "[cover] URI split: '%s', '%s'\n", targetHost, uri)
					} else { 
						logger.Printf (logger.WARN, "[cover] URI split failed on '%s'\n", uri)
					}
				} else {
					targetHost = c.server
				}  

				// assemble new resource request
				s.reqResource = uri
				req += "GET " + uri + " HTTP/1.0" + lb
				s.reqMode = REQ_GET

				// keep balance
				balance += (len(parts[1]) - len(uri))
			
			//---------------------------------------------------------
			// Host reference: change to hostname of cover server
			// This translation may leed to unbalanced request sizes;
			// the balance will be equalled in a later line
			// It is assumed, that a "Host:" line is one of the first
			// lines in a request and therefore never fragmented.
			//---------------------------------------------------------
			case strings.HasPrefix (line, "Host: "):
				// split line into parts
				parts := strings.Split (line, " ")
				// replace hostname reference 
				logger.Printf (logger.DBG_HIGH, "[cover] Host replaced with '%s'\n", targetHost)
				req += "Host: " + targetHost + lb
				// keep track of balance
				balance += (len(parts[1]) - len(targetHost))
				
			//---------------------------------------------------------
			// try to get balance straight on language header line:
			// "Accept-Language: de-de,de;q=0.8,en-us;q=0.5,en;q=0.3"
			//---------------------------------------------------------
			//case s.reqBalance != 0 && strings.HasPrefix (line, "Accept-Language: "):
			// @@@TODO: Is this the right place to balance the translation? 

			//---------------------------------------------------------
			// Acceptable content encoding: we only want plain HTML
			//---------------------------------------------------------
			case strings.HasPrefix (line, "Accept-Encoding: "):
				// split line into parts
				parts := strings.Split (line, " ")
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
			case strings.HasPrefix (line, "Content-Type: "):
				// split line into parts
				parts := strings.Split (line, " ")
				mime = parts[1]
				// remember boundary definition
				if s.reqMode == REQ_POST {
					// strip "boundary="
					s.reqBoundaryIn = string(parts[2][9:]) 														
					logger.Println (logger.DBG_HIGH, "[cover] Boundary=" + s.reqBoundaryIn)
					repl := parts[0] + " " + mime +
						  " boundary=-----------------------------" + s.reqBoundaryOut
					balance += len(repl) - len(line)
					req += repl + lb
				} else {
					req += line + lb
				}

			//---------------------------------------------------------
			// Referer
			//---------------------------------------------------------
			case strings.HasPrefix (line, "Referer: "):
				// don't add spec

			//---------------------------------------------------------
			// Connection
			//---------------------------------------------------------
			case strings.HasPrefix (line, "Connection: "):
				// split line into parts
				parts := strings.Split (line, " ")
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
			case strings.HasPrefix (line, "Keep-Alive: "):
				// don't add spec

			//---------------------------------------------------------
			// Content-Length
			//---------------------------------------------------------
			case strings.HasPrefix (line, "Content-Length: "):
				repl := "Content-Length: " + strconv.Itoa (len(s.reqCoverPost))
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
	if s.reqState == RS_HDR_COMPLETE {
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

		if s.reqMode == REQ_POST {
			// switch state			
			s.reqState = RS_CONTENT
		} else {
			// we are done
			s.reqState = RS_DONE
		}
	}
	
	// handle processing of request contents for POST requests
	if s.reqState == RS_CONTENT {
	
		// parse data until end of request
		for {
			// get next line (terminated by line break)
			// and adjust number of bytes read
			b,_,err := rdr.ReadLine()
			if err != nil {
				break
			}
			line := strings.TrimRight (string(b), "\r\n")

			//logger.Println (logger.DBG_ALL, "[cover] POST content: " + line + "\n")
			
			if !s.reqUpload {
				// check for start of document
				if strings.Index (line, "name=\"file\";") != -1 {
					s.reqUpload = true
					s.reqUploadData = ""
				}
			} else {
				if strings.Index (line, s.reqBoundaryIn) != -1 {
					s.reqUpload = false
					PostprocessUploadData (s.reqUploadData)
				}
				// we are uploading client data
				s.reqUploadData += line + lb
			}
		}
		
		// build new request data
		binReq := []byte(req)
		copy (data, binReq)
		pos := len(binReq)
		count := num - pos
	
		// we have "count" bytes of response data to sent out
		start := s.reqCoverPostPos
		total := len(s.reqCoverPost)
		if start <  total {
			end := start + count
			s.reqCoverPostPos = end
			if end > total {
				end = total
			}
			copy (data[pos:], s.reqCoverPost [start:end])
			pos += (end - start)
		}
		
		// fill up with line breaks
		if pos < num {
			fill := ""
			for count = num - pos; count > 0; count-- {
				fill += "\n"
			}
			data = append (data[0:pos], []byte(fill)...)
			pos = num
		}
		
		outStr := string(data[0:pos])
		logger.Printf (logger.DBG_HIGH, "[cover] %d bytes send to cover server.\n", pos)
		logger.Println (logger.DBG_ALL, "[cover] Outgoing request:\n" + outStr + "\n")
		return data[0:pos]
	}
	
	// check for completed request processing
	if s.reqState == RS_DONE {
		if balance != 0 {
			logger.Printf (logger.WARN, "[cover] Unbalanced request: %d bytes diff\n", balance)
		}
	}
	
	// padding of request with line breaks (if assembled request is smaller; GET only)
	for num > len(req) && s.reqMode == REQ_GET {
		req += "\n"
	}
	// return transformed request
	if num != len(req) {
		logger.Printf (logger.WARN, "[cover] DIFF(request) = %d\n", len(req)-num)
	}
	logger.Printf (logger.DBG_ALL, "[cover] Transformed request:\n" + req + "\n")
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
func (c *Cover) xformResp (s *State, data []byte, num int) []byte {

	// log incoming packet
	inStr := string(data[0:num])
	logger.Printf (logger.DBG_HIGH, "[cover] %d bytes received from cover server.\n", num)
	logger.Println (logger.DBG_ALL, "[cover] Incoming response:\n" + inStr + "\n")

	// setup reader and response
	size := num
	rdr := bytes.NewBuffer (data[0:num])
	resp := ""
	
	// initial response package
	if s.respMode == 0 {
	
		// use identical line break sequence	
		lb := "\r\n"
		if strings.Index (inStr, lb) == -1 {
			lb = "\n"
		}
		// start of new response encountered: parse header fields
		hdr: for {
			// get next line (terminated by line break)
			line,err := rdr.ReadString('\n')
			line = strings.TrimRight (line, "\n\r")
			if err != nil {
				// header is not complete: wait for next response fragment
				logger.Println (logger.WARN, "[cover] Response header fragmented!")
				logger.Println (logger.DBG, "[cover] Assembled response:\n" + resp)
				if size != len(resp) {
					logger.Printf (logger.WARN, "[cover] DIFF(response:1) = %d\n", len(resp)-size)
				}
				return []byte(resp)
			}
			// check if header is available at all..
			if strings.HasPrefix (line, "<!") {
				logger.Println (logger.INFO, "[cover] No response header found: " + line)
				break hdr
			}
			
			// parse response header
			switch {
				//-----------------------------------------------------
				// Header parsing complete
				//-----------------------------------------------------
				case len(line) == 0:
					// we have parsed the header; continue with body
					logger.Println (logger.DBG_ALL, "[cover] Incoming response header:\n" + resp)
					// drop length encoding on gzip content
					break hdr

				//-----------------------------------------------------
				// Status line
				//-----------------------------------------------------
				case strings.HasPrefix (line, "HTTP/"):
					// split line into parts
					parts := strings.Split (line, " ")
					status,_ := strconv.Atoi (parts[1])
					logger.Printf (logger.DBG, "[cover] response status: %d\n", status)
					if status != 200 {
						// pass back anything that is not OK
						return data[0:size]
					}
			
				//-----------------------------------------------------
				// Content-Type:
				//-----------------------------------------------------
				case strings.HasPrefix (line, "Content-Type: "):
					// split line into parts
					parts := strings.Split (line, " ")
					s.respType = strings.TrimRight (parts[1], ";")
					logger.Println (logger.DBG_HIGH, "[cover] response type: " + s.respType)

				//-----------------------------------------------------
				// Content-Encoding:
				//-----------------------------------------------------
				case strings.HasPrefix (line, "Content-Encoding: "):
					// split line into parts
					parts := strings.Split (line, " ")
					s.respEnc = parts[1]
					logger.Println (logger.DBG_HIGH, "[cover] response encoding: " + s.respEnc)
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
	if s.respMode == 0 {
		//-------------------------------------------------------------
		// (initial) HTML response		
		//-------------------------------------------------------------		
		if strings.HasPrefix (s.respType, "text/html") {
			// start of a new HTML response. Use pre-defined HTM page
			// to initialize response.
			s.respPending = c.getReplacementBody (s.reqResource)
			// emit HTML introduction sequence
			resp += htmlIntro
			num -= len(htmlIntro)
		}
		// switch to next mode
		s.respMode = 1
	}

	switch {
		//-------------------------------------------------------------
		// assmble HTML response		
		//-------------------------------------------------------------		
		case strings.HasPrefix (s.respType, "text/html"):
			// do content translation (collect resource tags)
			done := parseHTML (rdr, s.respHdr, s.respTags)
			// assemble header if required
			if s.respMode == 1 && s.respHdr.Count() > 0 {
				hdr := c.assembleHeader (s.respHdr, num)
				resp += hdr
				num -= len(hdr)
				// handle HTML body
				s.respMode = 2
			}
			// assemble HTML body
			resp += c.assembleBody (s, num, done)
			logger.Println (logger.DBG_ALL, "[cover] Translated response:\n" + resp)
			// return response data
			if size != len(resp) {
				logger.Printf (logger.WARN, "[cover] DIFF(response:2) = %d\n", len(resp)-size)
			}
			return []byte(resp)
			
		//-------------------------------------------------------------
		// Images: Images are considered harmless, so we simply
		// pass them back to the client.
		//-------------------------------------------------------------		
		case strings.HasPrefix (s.respType, "image/"):
			logger.Println (logger.DBG, "[cover] Image data passed to client")
			return data[0:size]
			
		//-------------------------------------------------------------
		// JavaScript: Simply replace any JavaScript content with
		// spaces (looks like the client browser has disabled
		// JavaScript).
		//-------------------------------------------------------------		
		case strings.HasPrefix (s.respType, "application/x-javascript"):
			// padding to requested size
			for n := 0; n < num; n++ {
				resp += " " 
			}
			// return response data
			logger.Println (logger.DBG, "[cover] JavaScript scrubbed")
			if size != len(resp) {
				logger.Printf (logger.WARN, "[cover] DIFF(response:3) = %d\n", len(resp)-size)
			}
			return []byte(resp)
			
		//-------------------------------------------------------------
		// CSS: Simply replace any style sheets with spaces. No image
		// references in CSS are parsed (looks like those are cached
		// resources to an eavesdropper)
		//-------------------------------------------------------------		
		case strings.HasPrefix (s.respType, "text/css"):
			// padding to requested size
			for n := 0; n < num; n++ {
				resp += " " 
			}
			// return response data
			logger.Println (logger.DBG, "[cover] CSS scrubbed")
			if size != len(resp) {
				logger.Printf (logger.WARN, "[cover] DIFF(response:4) = %d\n", len(resp)-size)
			}
			return []byte(resp)
	}
	
	//return untranslated response
	logger.Println (logger.ERROR, "[cover] Unhandled response!")
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
func (c *Cover) assembleBody (s *State, size int, done bool) string {

	// emit pending reponse data first
	resp := ""
	pending := len(s.respPending)
	switch {
		case pending > size:
			resp = string(s.respPending[0:size])
			s.respPending = string(s.respPending[size:])
			return resp
		case pending > 0:
			resp = s.respPending
			size -= pending
			s.respPending = ""
	}
	
	// add resources (if any are pending)
	for s.respTags.Count() > 0 {
		// get next tag
		tag := s.respTags.Get()
		if tag == nil {
			break
		}
		// translate tag for client
		inl := c.translateTag (tag)
		// check if we can add the tag?
		if len(inl) < size {
			// yes: add it to response
			resp += inl
			size -= len(inl)
		} else {
			// no: put it back
			s.respTags.Put (tag)
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
	resp += padding (size)

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
func (c *Cover) assembleHeader (tags *TagList, size int) string {

	// add header resources
	hdr := "<head>\n"
	for tags.Count() > 0 {
		// get next tag
		tag := tags.Get()
		if tag == nil {
			break
		}
		// translate tag for client
		inl := c.translateTag (tag) + "\n"
		// check if we can add the tag?
		if len(inl) < size {
			// yes: add it to response
			hdr += inl
			size -= len(inl)
		} else {
			// no: put it back
			logger.Printf (logger.WARN, "[cover] can't add all header tags: %d are skipped\n", tags.Count()+1) 
			break
		}
	}
	
	// close header
	hdr += "</head>\n"
	return hdr
}

//---------------------------------------------------------------------
/*
 * Get HTML replacement page: Return defined replacement page. If no
 * replacement is defined, return an error page. If the replacement
 * is tagged "[Upload]", generate a upload form
 * @param res string - name of the HTML resource
 * @return string - HTML body content (upload form)
 */
func (c *Cover) getReplacementBody (res string) string {

	// lookup pre-defined replacement page
	page,ok := c.htmls[res]
	// return error page if no replacement is defined.
	if !ok {
		logger.Println (logger.WARN, "[cover] Unknown HTML resource requested: " + res)
		return "<h1>Unsupported page. Please return to previous page!</h1>"
	}
	// return normal pages
	if !strings.HasPrefix (page, "[UPLOAD]") {
		return page
	}
	// generate upload form page
	return c.hdlr.getForm()
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
func (c *Cover) translateTag (tag *Tag) string {

	if src,ok := tag.attrs["src"]; ok {
		// translate "src" attribute of tag
		trgt := translateURI (src)
		logger.Printf (logger.INFO, "[cover] URI translation of '%s' => '%s'\n", src, trgt)
		tag.attrs["src"] = trgt
	} else if src,ok := tag.attrs["href"]; ok {
		// translate "href" attribute of tag
		trgt := translateURI (src)
		logger.Printf (logger.INFO, "[cover] URI translation of '%s' => '%s'\n", src, trgt)
		tag.attrs["href"] = trgt
	} else {
		// failed to access reference attribute?!
		s := tag.String()
		logger.Println (logger.ERROR, "[cover] Tag translation failed: " + s)
		return s
	}
	// return tag representation
	return tag.String()
}
