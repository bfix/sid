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
 * State information for cover server connection.
 */
type state struct {
	reqBalance	int			// size balance for request translation
	respBalance	int			// size balance for response translation
	binResp		bool		// pending response is binary data?
}

//---------------------------------------------------------------------
/*
 * Cover server instance (stateful)
 */
type Cover struct {
	server		string					// "host:port" of cover server
	states		map[net.Conn]*state		// state of active connections
}

///////////////////////////////////////////////////////////////////////
// Public types

/*
 * Create a new cover server instance
 * @return *Cover - pointer to cover server instance
 */
func NewCover() *Cover {
	return &Cover {
		server:		"www.picpost.com:80",
		states:		make (map[net.Conn]*state),
	}
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
	
	// allocate state information and add to
	// state list
	c.states[conn] = &state {
		reqBalance:		0,
		respBalance:	0,
		binResp:		false,
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
func (c *Cover) GetState (conn net.Conn) *state {
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
func (c *Cover) xformReq (s *state, data []byte, num int) []byte {

	inStr := string(data)
	logger.Printf (logger.INFO, "[http] %d bytes received from cover server.\n", len(data))
	logger.Println (logger.DBG_ALL, "[http] Incoming response:\n" + inStr + "\n")

	// assemble transformed request
	rdr := bufio.NewReader (strings.NewReader (inStr))
	req := ""
	complete := false
	for {
		// get next line (terminated by line break); if the
		// line is continued on the next block
		b,broken,_ := rdr.ReadLine()
		if b == nil || len(b) == 0 {
			complete = !broken
			break
		}
		line := string(b)
		//log.Printf ("[cover] +%s\n", line)
		
		// transform request data
		switch {
			//---------------------------------------------------------
			// GET command: request resource
			// If the requested resource identifier is a translated
			// entry, we need to translate that back into its original
			// form. Translated entries start with "/&&".
			// It is assumed, that a "GET" line is one of the first
			// lines in a request and therefore never fragmented.
			//---------------------------------------------------------
			case strings.HasPrefix (line, "GET "):
				// split line into parts
				parts := strings.Split (line, " ")
				logger.Printf (logger.DBG_HIGH, "[cover] resource='%s'\n", parts[1])
				
				// check for back-translation
				uri := parts[1]
				if strings.HasPrefix (uri, "/&&") {
				}
				// assemble new ressource request
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
func (c *Cover) xformResp (s *state, data []byte, num int) []byte {
	logger.Printf (logger.INFO, "[cover] %d bytes received from client.\n", len(data))
	logger.Println (logger.DBG_ALL, "[cover] Incoming request:\n" + string(data) + "\n")
	return data
}
