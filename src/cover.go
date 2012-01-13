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
	"log"
	"strings"
	"bufio"
)

///////////////////////////////////////////////////////////////////////
// Public types

/*
 * Cover server instance (stateful)
 */
type Cover struct {
	server	string			// "host:port" spec. for cover server
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
		log.Printf ("[cover] failed to connect to cover server: %s\n", err.String())
		return nil
	}
	log.Println ("[cover] connected to cover server...")
	return conn
}

//---------------------------------------------------------------------
/*
 * Transform client request: this is supposed to work on fragmented
 * requests if necessary (currently not really supported)
 * @param data []byte - request data from client
 * @param num int - length of request in bytes
 * @return []byte - transformed request (sent to cover server)
 */
func (c *Cover) xformReq (data []byte, num int) []byte {

	// parse HTTP request
	rdr := bufio.NewReader (strings.NewReader (string (data)))
	
	// assemble transformed request
	req := ""
	for {
		b,_,_ := rdr.ReadLine()
		if b == nil || len(b) == 0 {
			break
		}
		
		line := string(b)
		//log.Printf ("[cover] +%s\n", line)
		switch {
			// GET command
			case strings.HasPrefix (line, "GET "):
				parts := strings.Split (line, " ")
				log.Printf ("[cover] URI='%s'\n", parts[1])
				req += "GET " + parts[1] + " " + parts[2] + "\n"
			
			// Hostname
			case strings.HasPrefix (line, "Host: "):
				log.Printf ("[cover] Host replaced with '%s'\n", c.server)
				req += "Host: " + c.server + "\n"
				
			default:
				req += line + "\n"
		}
	}
	// add delimiting empty line
	req += "\n"
	if verbose {
		log.Println ("[cover] Transformed request:\n" + req + "\n")
	}
	return []byte(req)
}

//---------------------------------------------------------------------
/*
 * Transform cover server response: Substitute absolute URLs in the
 * response to local links to be handled by the request translations.
 * @param data []byte - response data from cover server
 * @param num int - length of response data
 * @return []data - transformed response (sent to client)
 */
func (c *Cover) xformResp (data []byte, num int) []byte {
	return data
}
