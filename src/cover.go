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
	"bufio"
	"gospel/logger"
)

///////////////////////////////////////////////////////////////////////
// Public types

/*
 * Tag represents all HTML tags from a cover server response (content)
 * that refer to an external ressource and therefore must be conserved
 * and translated to match the profile of a "normal" usage of the cover
 * site. (Resources are replaces by "innocent" and "unharnful" content
 * on the fly during the response handling for non-HTML ressources)
 */
type Tag struct {
	name	string
	attrs	map[string]string
}

//---------------------------------------------------------------------
/*
 * Instantiate a new Tag object with given parameters.
 * @param n string - name of tag
 * @param a map[string]string - list of attributes
 * @return *Tag - pointer to new instance
 */
func NewTag (n string, a map[string]string) *Tag {
	return &Tag {
		name:	n,
		attrs:	a,
	}
}

//---------------------------------------------------------------------
/*
 * Stringify tag
 * @return string - string representation of tag
 */
func (t *Tag) String() string {
	res := "<" + t.name
	for key,val := range t.attrs {
		res += " " + key + "=" + val
	}
	return res + "/>"
}

//=====================================================================
/*
 * State information for cover server connections.
 */
type State struct {
	reqBalance		int			// size balance for request translation
	reqRessource	string		// ressource requested
	respBalance		int			// size balance for response translation
	respCont		bool		// response continuation?
	respSize		int			// expected response size (total length)
	respType		string		// format identifier for response content (mime type)
	respBinary		bool		// pending response is binary data?
	respTags		[]*Tag		// list of tags to be included in response
	respHtmlDone	bool		// HTML closed?
}

//=====================================================================
/*
 * Cover server instance (stateful)
 */
type Cover struct {
	server		string						// "host:port" of cover server
	states		map[net.Conn]*State			// state of active connections
	htmls		map[string]string			// HTML page replacements
	hdlr		UploadHandler				// handler of cover uploads
}

///////////////////////////////////////////////////////////////////////
// Public types

/*
 * Create a new cover server instance
 * @return *Cover - pointer to cover server instance
 */
func NewCover() *Cover {
	// currently we only have one cover server implementation
	return NewCvrImgon()
}

///////////////////////////////////////////////////////////////////////
// Public methods

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
		reqBalance:		0,
		reqRessource:	"",
		respBalance:	0,
		respCont:		false,
		respSize:		0,
		respType:		"text/html",
		respBinary:		false,
		respTags:		make ([]*Tag, 0),
		respHtmlDone:	false,
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
 * @return []byte - transformed request (sent to cover server)
 */
func (c *Cover) xformReq (s *State, data []byte, num int) []byte {

	inStr := string(data[0:num])
	logger.Printf (logger.DBG_HIGH, "[http] %d bytes received from cover server.\n", num)
	logger.Println (logger.DBG_ALL, "[http] Incoming response:\n" + inStr + "\n")

	// assemble transformed request
	rdr := bufio.NewReader (strings.NewReader (inStr))
	req := ""
	complete := false
	for {
		// get next line (terminated by line break)
		b,broken,_ := rdr.ReadLine()
		if b == nil || len(b) == 0 {
			complete = !broken
			break
		}
		line := string(b)
		
		// transform request data
		switch {
			//---------------------------------------------------------
			// GET command: request resource
			// If the requested resource identifier is a translated
			// entry, we need to translate that back into its original
			// form. Translated entries start with "/&".
			// It is assumed, that a "GET" line is one of the first
			// lines in a request and therefore never fragmented.
			//---------------------------------------------------------
			case strings.HasPrefix (line, "GET "):
				// split line into parts
				parts := strings.Split (line, " ")
				logger.Printf (logger.DBG_HIGH, "[cover] resource='%s'\n", parts[1])
				
				// check for back-translation
				uri := parts[1]
				if strings.HasPrefix (uri, "/&") {
					// split into scheme and remaining URI
					pos := strings.Index (string(uri[2:]), "/")
					scheme := string(uri[2:pos])
					res := string(uri[pos:])
					uri = scheme + "://" + res 
				}
				// assemble new ressource request
				s.reqRessource = uri
				req += "GET " + uri + " " + parts[2] + "\n"
				// keep balance
				s.reqBalance += (len(parts[1]) - len(uri))
			
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
				logger.Printf (logger.DBG_HIGH, "[cover] Host replaced with '%s'\n", c.server)
				req += "Host: " + c.server + "\n"
				// keep track of balance
				s.reqBalance += (len(parts[1]) - len(c.server))
				
			//---------------------------------------------------------
			// try to get balance straight on language header line:
			// "Accept-Language: de-de,de;q=0.8,en-us;q=0.5,en;q=0.3"
			//---------------------------------------------------------
			//case s.reqBalance != 0 && strings.HasPrefix (line, "Accept-Language: "):
			// @@@TODO: Is this the right place to balance the translation? 

			//---------------------------------------------------------
			// add unchanged request lines. 
			//---------------------------------------------------------
			default:
				req += line
				if !broken {
					req += "\n"
				}
		}
	}
	// check if the request processing has completed
	if complete {
		// add delimiting empty line
		req += "\n"
		if s.reqBalance != 0 {
			logger.Printf (logger.WARN, "[cover] Unbalanced request: %d bytes diff\n", s.reqBalance)
		}
		logger.Printf (logger.DBG_ALL, "[cover] Transformed request:\n" + req + "\n")
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
 * @return []data - transformed response (sent to client)
 */
func (c *Cover) xformResp (s *State, data []byte, num int) []byte {

	inStr := string(data[0:num])
	logger.Printf (logger.DBG_HIGH, "[cover] %d bytes received from cover server.\n", num)
	logger.Println (logger.DBG_ALL, "[cover] Incoming response:\n" + inStr + "\n")

	rdr := bufio.NewReader (strings.NewReader (inStr))
	resp := ""
	if !s.respCont {
		// start of new response encountered: parse header fields
		for {
			// get next line (terminated by line break); if the
			// line is continued on the next block
			b,broken,_ := rdr.ReadLine()
			if b == nil || len(b) == 0 {
				if broken {
					// header is not complete: wait for next response fragment
					return data
				}
				// we have parsed the header; continue with body
				break
			}
			line := string(b)
			// assemble response
			resp += line + "\n"
			
			// parse response header
			switch {
				//-----------------------------------------------------
				// Content-Type:
				//-----------------------------------------------------
				case strings.HasPrefix (line, "Content-Type: "):
					// split line into parts
					parts := strings.Split (line, " ")
					s.respType = strings.TrimRight (parts[1], ";")
					logger.Println (logger.DBG_HIGH, "[cover] response type: " + s.respType)
					
					// set response representation
					s.respBinary = false
					switch {
						case strings.HasPrefix (s.respType, "img"):
							s.respBinary = true
					}
					logger.Printf (logger.DBG_HIGH, "[cover] response is binary? %v\n",s.respBinary)
			}
		}
		// add the delimiter (empty) line
		resp += "\n"
	}

	// we have parsed the response header; now process the response body
	if strings.HasPrefix (s.respType, "text/") {
		// do content translation/assembly
		parseHTML (rdr, s.respTags)
		resp += c.assembleHTML (s, num)
		// we are now in continuation mode.
		s.respCont = true
		// return response data
		return []byte(resp)
	}
	
	//return untranslated response
	logger.Println (logger.ERROR, "[cover] Unhandled response!")
	return data		
}

//=====================================================================
/*
 * Assemble a response from the current state (like response header),
 * the resource list and a replacement body (addressed by the requested
 * ressource path from state).
 * @param s *state - current state info
 * @param size int - target size of response
 * @return []byte - assembled response
 */
func (c *Cover) assembleHTML (s *State, size int) string {
	resp := ""
	if !s.respCont {
		// start of a new HTML response. Use pre-defined HTM page
		// for content and add as many pending tags as possible.
		page := c.getReplacementPage (s.reqRessource)
		// compute remaining size
		size -= len(page)
		// we have a pre-defined page.
		resp = page
	}
	
	// add ressources (if any are pending)
	if len(s.respTags) > 0 {
		next := 0
		for pos,tag := range s.respTags {
			inl := c.translateTag (tag)
			if len(inl) < size {
				resp += inl
				size -= len(inl)
				next = pos+1
			} else {
				break
			}
		}
		// removed written tags
		s.respTags = s.respTags[next:]
	}
				
	if !s.respHtmlDone {
		// close HTML if space allows
		htmlOff := "</body></html>"
		if len(htmlOff) < size {
			resp += htmlOff
			size -= len(htmlOff)
			resp += padding (size)
			s.respHtmlDone = true
		} else {
			resp += padding (size)
			s.respHtmlDone = false
		}
	} else {
		// we are done, but have still response data to transfer. Fill up
		// with padding sequence. 
		resp += padding (size)
	}
	return resp
}

//---------------------------------------------------------------------
/*
 * Get HTML replacement page: Return defined replacement page. If no
 * replacement is defined, return an error page. If the replacement
 * is tagged "[Upload]", generate a upload form
 * @param res string - name of the HTML ressource
 * @return string - page content
 */
func (c *Cover) getReplacementPage (res string) string {

	// lookup pre-defined replacement page
	page,ok := c.htmls[res]
	// return error page if no replacement is defined.
	if !ok {
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

	// translate "src" attribute of tag
	if src,ok := tag.attrs["src"]; ok {
		if pos := strings.Index (src, "://"); pos != -1 {
			// we have an absolute URI that needs translation
			logger.Printf (logger.INFO, "[cover] URI translation of '%s'\n", src)
			scheme := string(src[0:pos])
			res := string(src[pos+2:])
			tag.attrs["src"] = "/&" + scheme + res
		}
	} else {
		// failed to access "src" attribute?!
		s := tag.String()
		logger.Println (logger.ERROR, "[cover] Tag translation failed: " + s)
		return s
	}
	// return tag representation
	return tag.String()
}
