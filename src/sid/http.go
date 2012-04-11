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

package sid

///////////////////////////////////////////////////////////////////////
// Import external declarations.

import (
	"gospel/logger"
	"gospel/network"
	"net"
	"strings"
)

///////////////////////////////////////////////////////////////////////
// Public types

/*
 * HTTP service instance: In the current version there is only one
 * Cover server instance available that is shared among all HTTP
 * go-routines.
 */
type HttpSrv struct {
	hndlr *Cover // cover server reference
}

///////////////////////////////////////////////////////////////////////
/*
 * Create and initialize new HTTP service instance
 * @param cover *Cover - refderence to cover server instance
 */
func NewHttpSrv(cover *Cover) *HttpSrv {
	return &HttpSrv{
		//	Instantiate Cover (content transformer)
		hndlr: cover,
	}
}

///////////////////////////////////////////////////////////////////////
// HTTP service methods (implements Service interface)

/*
 * Handle client connection.
 * @param client net.Conn - connection to client
 */
func (s *HttpSrv) Process(client net.Conn) {

	// close client connection on function exit
	defer client.Close()

	// allocate buffer
	data := make([]byte, 32768)

	// open a new connection to the cover server
	cover := s.hndlr.connect()
	if cover == nil {
		// failed to open connection to cover server
		return
	}
	// close connection to cover server on exit
	defer s.hndlr.disconnect(cover)
	// get associated state info
	state := s.hndlr.GetState(cover)

	// handle session loop
	for {
		//-------------------------------------------------------------
		//	Upstream message passing
		//-------------------------------------------------------------

		// get data from cover server.
		n, ok := network.RecvData(cover, data, "http")
		if !ok {
			// epic fail: terminate session
			return
		}
		// send pending response to client
		if n > 0 {
			// transform response
			resp := s.hndlr.xformResp(state, data, n)
			// send incoming response data to client
			if !network.SendData(client, resp, "http") {
				// terminate session on failure
				logger.Println(logger.ERROR, "[sid.http] Failed to send data to client.")
				return
			}
		}

		//-------------------------------------------------------------
		//	Downstream message passing
		//-------------------------------------------------------------

		// get data from client.
		n, ok = network.RecvData(client, data, "http")
		if !ok {
			// epic fail: terminate session
			return
		}
		// send pending client request
		if n > 0 {
			// transform request
			req := s.hndlr.xformReq(state, data, n)
			// send request to cover server
			if !network.SendData(cover, req, "http") {
				// terminate session on failure
				logger.Println(logger.ERROR, "[sid.http] Failed to send data to cover.")
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
func (s *HttpSrv) CanHandle(protocol string) bool {
	rc := strings.HasPrefix(protocol, "tcp")
	if !rc {
		logger.Println(logger.INFO, "[sid.http] Unsupported protocol '"+protocol+"'")
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
func (s *HttpSrv) IsAllowed(addr string) bool {
	idx := strings.Index(addr, ":")
	ip := addr[:idx]
	if strings.Index(CfgData.HttpAllow, ip) == -1 {
		logger.Println(logger.WARN, "[sid.http] Invalid remote address '"+addr+"'")
		return false
	}
	return true
}

//---------------------------------------------------------------------
/*
 * Get service name.
 * @return string - name of control service (for logging purposes)
 */
func (s *HttpSrv) GetName() string {
	return "sid.http"
}
