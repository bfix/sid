/*
 * Handle SID configuration data: Read configuration data from file
 * or defined as command line options.
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
	"flag"
	"log"
	"os"
	"bufio"
	"strconv"
	"gospel/parser"
)

///////////////////////////////////////////////////////////////////////
// Public configuration data

/*
 * Configuation data type
 */
type Config struct {
	CfgFile		string		// configuration file name
	LogFile		string		// logging file name
	LogState	bool		// use file-based logging?
	CtrlPort	int			// port for control sessions
	HttpPort	int			// port for HTTP sessions
	HttpsPort	int			// port for HTTPS sessions
}

/*
 * Configuration data instance (with default values)
 */
var CfgData Config = Config {
	CfgFile:	"sid.cfg",		// default config file
	LogFile:	"sid.log",		// default logging file
	LogState:	false,			// no file-based logging
	CtrlPort:	2342,			// port for local control service
	HttpPort:	80,				// expected port for HTTP connections
	HttpsPort:	443,			// expected port for HTTPS connections
}

///////////////////////////////////////////////////////////////////////
// Public methods

/*
 * Setup configuration data
 */
func InitConfig () {
	// process command line arguments	
	CfgData.CfgFile = *flag.String ("c", CfgData.CfgFile, "configuration file")
	CfgData.LogFile = *flag.String ("L", CfgData.LogFile, "log file")
	CfgData.LogState = *flag.Bool ("l", CfgData.LogState, "turn on file-based logging")
	
	// read configuration from file
	log.Println ("[config] using configuration file '" + CfgData.CfgFile + "'")
	cfg,err := os.Open (CfgData.CfgFile)
	if err != nil {
		log.Println ("[config] configuration file not available -- using defaults")
		return
	}
	// configuration file exists: read parameters
 	rdr := bufio.NewReader (cfg)
	err = parser.Parser (rdr, callback)
	if err != nil {
		log.Printf ("[config] error reading configuration file: %v\n", err)
		os.Exit (1)
	}
	log.Println ("[config] configuration complete.")

	// turn on logging if specified on command line or config file
	if CfgData.LogState {
		log.Println ("[config] file-based logging to '" + CfgData.LogFile + "'")
		if f,err := os.Create (CfgData.LogFile); err == nil {
			log.SetOutput (f)
		} else {
			log.Println ("[config] can't enable file-based logging!")
			CfgData.LogState = false
		}
	}

	// Handle additional command line arguments (options)
	CfgData.CtrlPort = *flag.Int  ("p", CfgData.CtrlPort, "control session port")
	CfgData.HttpPort = *flag.Int  ("h", CfgData.HttpPort, "HTTP session port")
	CfgData.HttpsPort = *flag.Int  ("s", CfgData.HttpsPort, "HTTPS session port")
	
	// list current configuration data
	log.Println ("[config] !==========< configuration >===============")
	log.Println ("[config] !Configuration file: " + CfgData.CfgFile)
	log.Println ("[config] !Port for control sessions: " + strconv.Itoa(CfgData.CtrlPort))
	log.Println ("[config] !Port for HTTP sessions: " + strconv.Itoa(CfgData.HttpPort))
	log.Println ("[config] !Port for HTTPS sessions: " + strconv.Itoa(CfgData.HttpsPort))
	log.Println ("[config] !==========================================")
}

//---------------------------------------------------------------------
/*
 * Handle callback from parser.
 * @param mode int - parameter mode 
 * @param param *Parameter - reference to new parameter
 * @return bool - successful operation?
 */
func callback (mode int, param *parser.Parameter) bool {
	// if parameter is specified
	if param != nil {

		// print incoming parameter
		log.Printf ("[config] %d: `%v=%v`\n", mode, param.Name, param.Value)
		
		if mode != parser.LIST {
			switch param.Name {
				case "LogFile":		CfgData.LogFile = param.Value
				case "LogState":	CfgData.LogState = (param.Value == "ON")
				case "CrtlPort":	setIntValue (&CfgData.CtrlPort, param.Value)
				case "HttpPort":	setIntValue (&CfgData.HttpPort, param.Value)
				case "HttpsPort":	setIntValue (&CfgData.HttpsPort, param.Value)
			}
		}
	} 
	return true
}

//---------------------------------------------------------------------
/*
 * Set target integer to value parsed from string.
 * @param trgt *int - pointer to target instance
 * @param data string - string representation of value
 */
func setIntValue (trgt *int, data string) {
	if val,err := strconv.Atoi(data); err == nil {
		*trgt = val
	} else {
		log.Println ("[config] string conversion to integer value failed")
	}
}
