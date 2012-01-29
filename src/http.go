/*
 * Handle HTTP session: Every session is associated with two queues
 * for incoming and outgoing stream data and a handler inbetween to
 * read/write data from these queues after applying the needed
 * transformations.
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
	"gospel/logger"
)

///////////////////////////////////////////////////////////////////////
// Public types

/*
 * HTTP service instance: In the current version there is only one
 * Cover server instance available that is shared among all HTTP
 * go-routines.
 */
type HttpSrv struct  {
	hndlr		*Cover		// cover server reference
}

///////////////////////////////////////////////////////////////////////
/*
 * Create and initialize new HTTP service instance
 */
func NewHttpSrv() *HttpSrv {
	return &HttpSrv {
		//	Instantiate Cover (content transformer)
		hndlr:	NewCover(),
	}
}

///////////////////////////////////////////////////////////////////////
// HTTP service methods (implements Service interface)

/*
 * Handle client connection.
 * @param client net.Conn - connection to client
 */
func (s *HttpSrv) Process (client net.Conn) {

	// close client connection on function exit
	defer client.Close()

	// allocate buffer
	data := make ([]byte, 32768)
	
	// open a new connection to the cover server
	cover := s.hndlr.connect ()
	if cover == nil {
		// failed to open connection to cover server
		return
	}
	// close connection to cover server on exit
	defer s.hndlr.disconnect (cover)
	// get associated state info
	state := s.hndlr.GetState (cover)

	// don't block read/write operations on socket buffers
	client.SetTimeout (1)
	cover.SetTimeout (1)
	
	// handle session loop
	for {
		//-------------------------------------------------------------
		//	Upstream message passing
		//-------------------------------------------------------------
		
		// get data from cover server.
		n,ok := rcvData (cover, data, "http")
		if !ok {
			// epic fail: terminate session
			return
		}
		// send pending response to client
		if n > 0 {
			// transform response
			resp := s.hndlr.xformResp (state, data, n) 
			// sent incoming response data to client
			if !sentData (client, resp, "http") {
				// terminate session on failure
				logger.Println (logger.ERROR, "[http] Failed to send data to client.")
				return
			}
		}

		//-------------------------------------------------------------
		//	Downstream message passing
		//-------------------------------------------------------------
				
		// get data from client.
		n,ok = rcvData (client, data, "http")
		if !ok {
			// epic fail: terminate session
			return
		}
		// send pending client request
		if n > 0 {
			// transform request
			req := s.hndlr.xformReq (state, data, n)
			// sent request to cover server
			if !sentData (cover, req, "http") {
				// terminate session on failure
				logger.Println (logger.ERROR, "[http] Failed to send data to cover.")
				return
			}
		}
	}
}

//---------------------------------------------------------------------
/*
 * Check for TCP protocol.
 * @param protocol string - connection protocol
 * @return bool - protcol handled?
 */
func (s *HttpSrv) CanHandle (protocol string) bool {
	rc := strings.HasPrefix (protocol, "tcp")
	if !rc {
		logger.Println (logger.INFO, "[http] Unsupported protocol '" + protocol + "'") 
	}
	return rc 
}

//---------------------------------------------------------------------
/*
 * Check for local connection: Only connections from the local
 * TOR exit node are accepted.
 * @param add string - remote address
 * @return bool - local address?
 */
func (s *HttpSrv) IsAllowed (addr string) bool {
	rc := strings.HasPrefix (addr, "127.0.0.1")
	if !rc {
		logger.Println (logger.WARN, "[http] Invalid remote address '" + addr + "'") 
	}
	return rc
}

//---------------------------------------------------------------------
/*
 * Get service name.
 * @return string - name of control service (for logging purposes)
 */
func (s *HttpSrv) GetName() string {
	return "http"
}
