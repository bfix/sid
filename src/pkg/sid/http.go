/*
 * Handle HTTP session: Every session is associated with two queues
 * for incoming and outgoing stream data and a handler in between to
 * read/write data from these queues.
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
	"net"
	"log"
	"bufio"
	"strings"
)

///////////////////////////////////////////////////////////////////////
// Constants

const (
	verbose = true
)

///////////////////////////////////////////////////////////////////////
// Public types

/*
 * HTTP service instance
 */
type HttpSrv struct  {
}

///////////////////////////////////////////////////////////////////////
/*
 * Create and initialize new HTTP service instance
 */
func NewHttpSrv() *HttpSrv {
	return &HttpSrv{}
}

///////////////////////////////////////////////////////////////////////
// HTTP service methods (implements Service interface)

/*
 * Handle client connection.
 * @param client net.Conn - connection to client
 */
func (s *HttpSrv) Process (client net.Conn) {

	// allocate input buffer
	inData := make ([]byte, 32768)
	// create reader/writer instance
	b := bufio.NewReadWriter (bufio.NewReader(client), bufio.NewWriter(client))
	
	// handle session
	for {
		// get data from client.
		n,err := b.Read (inData)
		if err != nil {
			log.Println ("[http] Read failed: " + err.String())
			continue
		}
		inStr := string(inData)
		
		// optional logging
		if verbose {
			log.Printf ("[http] %d bytes read.\n", n)
			log.Println ("[http]: '" + inStr + "'")
		}
		
		
	}
	client.Close()
}

//---------------------------------------------------------------------
/*
 * Check for TCP protocol.
 * @param protocol string - connection protocol
 * @return bool - protcol handled?
 */
func (s *HttpSrv) CanHandle (protocol string) bool {
	return strings.HasPrefix (protocol, "tcp")
}

//---------------------------------------------------------------------
/*
 * Check for local connection: Only connections from the local
 * TOR exit node are accepted.
 * @param add string - remote address
 * @return bool - local address?
 */
func (s *HttpSrv) IsAllowed (addr string) bool {
	return strings.HasPrefix (addr, "127.0.0.1")
}

//---------------------------------------------------------------------
/*
 * Get service name.
 * @return string - name of control service (for logging purposes)
 */
func (s *HttpSrv) GetName() string {
	return "http"
}
