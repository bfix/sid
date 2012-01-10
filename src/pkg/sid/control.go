/*
 * Handle control session: Control sessions are used to administrate
 * a running SID service. Only TCP connections from localhost are
 * accepted!
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
	"os"
	"net"
	"log"
	"bufio"
	"strings"
)

///////////////////////////////////////////////////////////////////////
// Control service instance

type ControlSrv struct  {
	Ch		chan bool			// channel to invoker
}

///////////////////////////////////////////////////////////////////////
// ControlService methods (implements Service interface)

/*
 * Handle client connection.
 * @param client net.Conn - connection to client
 */
func (c *ControlSrv) Process (client net.Conn) {

	b := bufio.NewReadWriter (bufio.NewReader(client), bufio.NewWriter(client))
	for repeat := true; repeat; {
	
		// prepare state information for output
		logState := "OFF"
		if CfgData.LogState {
			logState = "ON"
		}

		// show control menu			
		b.WriteString ("\n-----------------------------------\n")
		b.WriteString ("toggle (L)ogging [" + logState + "]\n")
		b.WriteString ("(T)erminate application\n")
		b.WriteString ("e(X)it\n")
		b.WriteString ("-----------------------------------\n")
		b.WriteString ("Enter command: ")
		b.Flush()

		// get command input
		cmd,err := readCmd (b)
		if err != nil {
			break
		}

		// handle command
		log.Print ("[ctrl] command '" + cmd + "'\n")
		switch cmd {
			//-------------------------------------------------
			// Terminate application
			//-------------------------------------------------
			case "T":	b.WriteString ("Are you sure? Enter YES to continue: ")
						b.Flush()
						cmd,_ = readCmd (b)
						if cmd == "YES" {
							log.Println ("[ctrl] Terminating application")
							b.WriteString ("Terminating application...")
							b.Flush()
							c.Ch <- true
						} else {
							log.Println ("[ctrl] Response '" + cmd + "' -- Termination aborted!")
							b.WriteString ("Wrong response -- Termination aborted!")
							b.Flush()
						}
						
			//-------------------------------------------------
			//	Quit control session
			//-------------------------------------------------
			case "X":	repeat = false

			//-------------------------------------------------
			//	Unknown command
			//-------------------------------------------------
			default:	b.WriteString ("Unkonwn command '" + cmd + "'\n")
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
func (c *ControlSrv) CanHandle (protocol string) bool {
	return strings.HasPrefix (protocol, "tcp")
}

//---------------------------------------------------------------------
/*
 * Check for local connection.
 * @param add string - remote address
 * @return bool - local address?
 */
func (c *ControlSrv) IsAllowed (addr string) bool {
	return strings.HasPrefix (addr, "127.0.0.1")
}

//---------------------------------------------------------------------
/*
 * Get service name.
 * @return string - name of control service (for logging purposes)
 */
func (c *ControlSrv) GetName() string {
	return "ctrl"
}

///////////////////////////////////////////////////////////////////////
// Private helper methods

/*
 * Read command/input from connection.
 * @param b *bufioReadWriter - reader
 * @return cmd string - read input
 * @return err os.Error - error state
 */
func readCmd (b *bufio.ReadWriter) (cmd string, err os.Error) {
	line, err := b.ReadBytes ('\n')
	if err != nil {
		return "", err
	}
	// get rid of enclosing white spaces
	return strings.Trim (string(line), " \t\n\v\r"), nil
}
