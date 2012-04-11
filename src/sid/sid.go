/*
 * ====================================================================
 *    Server In Disguise (SID)  --  Main application start-up code
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

package sid

///////////////////////////////////////////////////////////////////////
// Import external declarations.

import (
	"gospel/logger"
	"gospel/network"
	"strconv"
)

///////////////////////////////////////////////////////////////////////
/*
 * Custom initialization method: Return cover instance to be used
 * to handle cover traffic
 */
var CustomInitialization func() *Cover = nil

/*
 * Optional HTTP fallback handler.
 */
var HttpFallback network.Service = nil

///////////////////////////////////////////////////////////////////////
// Main application start-up code.

func Startup() {

	logger.Println(logger.INFO, "[sid] ==============================")
	logger.Println(logger.INFO, "[sid] SID v0.2 -- Server In Disguise")
	logger.Println(logger.INFO, "[sid] (c) 2012 Bernd R. Fix      >Y<")
	logger.Println(logger.INFO, "[sid] ==============================")

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
	InitConfig()

	//-----------------------------------------------------------------
	//	Initialize cover-related settings
	//-----------------------------------------------------------------

	InitDocumentHandler(CfgData.Upload)

	if CustomInitialization == nil {
		logger.Println(logger.ERROR, "[sid] No custom initialization function defined -- aborting!")
		return
	}
	cover := CustomInitialization()

	//-----------------------------------------------------------------
	//	Start network services
	//-----------------------------------------------------------------

	// create control service.
	ch := make(chan bool)
	ctrl := &ControlSrv{ch}
	ctrlList := []network.Service { ctrl }

	// create HTTP service
	http := NewHttpSrv(cover)
	httpList := []network.Service { http }
	if HttpFallback != nil {
		httpList = append (httpList, HttpFallback)
	}

	// start network services
	network.RunService("tcp", ":"+strconv.Itoa(CfgData.CtrlPort), ctrlList)
	network.RunService("tcp", ":"+strconv.Itoa(CfgData.HttpPort), httpList)

	// wait for termination
	<-ch
	logger.Println(logger.INFO, "[sid] Application terminated.")
}
