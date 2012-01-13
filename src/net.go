/*
 * Network helper functions: Send and receive data over socket buffer.
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
	"gospel/logger"
)

///////////////////////////////////////////////////////////////////////
// Public functions

/*
 * Sent data over network connection (stream-oriented).
 * @param conn net.Conn - network connection
 * @param data []byte - data to be sent
 * @param srv string - send on behalf of specified service
 * @return bool - successful operation (or connection closed/to be closed)
 */
func sentData (conn net.Conn, data []byte, srv string) bool {

	count := len(data)		// total length of data
	start := 0				// start position of slice
	retry := 0				// retry conter
	
	// write data to socket buffer
	for count > 0 {
		// get (next) chunk to be sent
		chunk := data [start:start+count] 
		if num,err := conn.Write (chunk); err == nil {
			// advance slice on partial write
			start += num
			count -= num
			retry = 0
		} else {
			// handle error condition
			switch err.(type) {
				case net.Error:
					// network error: retry...
					nerr := err.(net.Error)
					if nerr.Timeout() || nerr.Temporary() {
						retry++
						if retry == 3 {
							logger.Printf (logger.ERROR, "[%s] Write failed after retries: %s\n", srv, err.String())
							return false
						}
					}
				default:
					// we are in real trouble...
					logger.Printf (logger.ERROR, "[%s] Write failed finally: %s\n", srv, err.String())
					return false
			}
		}
	}
	// report success
	return true
}

//---------------------------------------------------------------------
/*
 * Receive data over network connection (stream-oriented).
 * @param conn net.Conn - network connection
 * @param data []byte - data buffer
 * @param srv string - receive on behalf of specified service
 * @return int - number of bytes read
 * @return bool - successful operation (or connection closed/to be closed)
 */
func rcvData (conn net.Conn, data []byte, srv string) (int, bool) {

	for retry := 0; retry < 3; {
		// read data from socket buffer
		n,err := conn.Read (data)
		if err != nil {
			// handle error condition
			switch err.(type) {
				case net.Error:
					// network error: retry...
					nerr := err.(net.Error)
					if nerr.Timeout() {
						return 0, true
					} else if nerr.Temporary() {
						retry++
						continue
					}
				default:
					// we are in real trouble...
					logger.Printf (logger.ERROR, "[%s] Read failed finally: %s\n", srv, err.String())
					return 0,false
			}
		}
		// report success
		return n,true
	}
	// retries failed
	logger.Printf (logger.ERROR, "[%s] Read failed after retries...\n", srv)
	return 0, false
}
