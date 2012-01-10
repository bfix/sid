/*
 * ====================================================================
 *    Server In Disguise (SID)  --  Main application star-up code
 * ====================================================================
 * Start-up connection handlers for HTTP, HTTPS and control services.
 * Parameters are defined in a configuration file or defined/overridden
 * directly on the command line; some of the parameters  can later be
 * modified using the local control service.  
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
	"sid"
	"strconv"
	"gospel/network"
)

///////////////////////////////////////////////////////////////////////
// Main application start-up code.

func main() {

	//-----------------------------------------------------------------
	// Handle SID configuration: read configuration data from config
	// file 'sid.cfg' in current directory (can be overridden by the
	// '-c <file>' option on the command line. If no configuration file
	// exists, default values for all config options are used.
	// Configuration options used on the command line will override
	// options defined in the config file (or default options). 
	//-----------------------------------------------------------------
	
	// handle configuration file and command line options
	// (turns on file-based logging if specified on command line) 
	sid.InitConfig ()
	
	// create control service.
	ch := make (chan bool)
	ctrl := &sid.ControlSrv { ch }
	
	// create HTTP service
	http := sid.NewHttpSrv()
	
	// start network services
	network.RunService ("tcp", ":" + strconv.Itoa(sid.CfgData.CtrlPort), ctrl)
	network.RunService ("tcp", ":" + strconv.Itoa(sid.CfgData.HttpPort), http)
	
	// wait for termination
	<-ch
}
