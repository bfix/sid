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
	"os"
	"bufio"
	"strconv"
	"gospel/parser"
	"gospel/logger"
)

///////////////////////////////////////////////////////////////////////
// Public configuration data

/*
 * Configuation data type.
 */
type Config struct {
	CfgFile			string		// configuration file name
	LogFile			string		// logging file name
	LogState		bool		// use file-based logging?
	CtrlPort		int			// port for control sessions
	HttpPort		int			// port for HTTP sessions
	Upload			UploadDefs	// upload-related settings
}

//---------------------------------------------------------------------
/*
 * Upload-related settings.
 */
type UploadDefs struct {
	Path			string		// directory to store client uploads
	Keyring			string		// name of OpenPGP keyring file
	SharePrimeOfs	int			// prime number offset for secret sharing
	ShareTreshold	int			// number of people required to access documents
}

//---------------------------------------------------------------------
/*
 * Configuration data instance (with default values) accessible
 * from all modules/packages of the application.
 */
var CfgData Config = Config {
	CfgFile:	"sid.cfg",		// default config file
	LogFile:	"sid.log",		// default logging file
	LogState:	false,			// no file-based logging
	CtrlPort:	2342,			// port for local control service
	HttpPort:	80,				// expected port for HTTP connections
	
	Upload:		UploadDefs {
					Path:			"./uploads",
					Keyring:		"./uploads/pubring.gpg",
					SharePrimeOfs:	568,
					ShareTreshold:	2,
				},
}

//---------------------------------------------------------------------
/*
 * Custom callback handler for non-standard configuration options.
 */
var CustomConfigHandler parser.Callback = nil

///////////////////////////////////////////////////////////////////////
// Public methods
/*
 * Setup configuration data: Handle SID configuration data and
 * call custom handler for non-standard configuration data
 */
func InitConfig () {

	// process command line arguments	
	CfgData.CfgFile = *flag.String ("c", CfgData.CfgFile, "configuration file")
	flag.String ("L", CfgData.LogFile, "logfile name")
	flag.Bool ("l", CfgData.LogState, "file-based logging")
	flag.Int  ("p", CfgData.CtrlPort, "control session port")
	flag.Int  ("h", CfgData.HttpPort, "HTTP session port")
	flag.Parse()
	
	// read configuration from file
	logger.Println (logger.INFO, "[sid.config] using configuration file '" + CfgData.CfgFile + "'")
	cfg,err := os.Open (CfgData.CfgFile)
	if err != nil {
		logger.Println (logger.WARN, "[sid.config] configuration file not available -- using defaults")
		return
	}
	// configuration file exists: read parameters
 	rdr := bufio.NewReader (cfg)
	err = parser.Parser (rdr, callback)
	if err != nil {
		logger.Printf (logger.ERROR, "[sid.config] error reading configuration file: %v\n", err)
		os.Exit (1)
	}
	logger.Println (logger.INFO, "[sid.config] configuration file complete.")

	// handle command line flags that may override options specified in the
	// configuration file (or are default values)
	flag.Visit (func (f *flag.Flag) {
		val := f.Value.String()
		logger.Printf (logger.INFO, "[sid.config] Overriding '%s' with '%s'\n", f.Usage, val)
		switch f.Name {
			case "L":	CfgData.LogFile = val
			case "l":	CfgData.LogState = (val == "true")
			case "p":	CfgData.CtrlPort,_ = strconv.Atoi (val)
			case "h":	CfgData.HttpPort,_ = strconv.Atoi (val)
		}
	})
	
	// turn on logging if specified on command line or config file
	if CfgData.LogState {
		logger.Println (logger.INFO, "[sid.config] File logging requested.")
		if !logger.LogToFile (CfgData.LogFile) {
			CfgData.LogState = false
		}
	}

	// list current configuration data
	logger.Println (logger.INFO, "[sid.config] !==========< configuration >===============")
	logger.Println (logger.INFO, "[sid.config] !Configuration file: " + CfgData.CfgFile)
	logger.Println (logger.INFO, "[sid.config] !Port for control sessions: " + strconv.Itoa(CfgData.CtrlPort))
	logger.Println (logger.INFO, "[sid.config] !Port for HTTP sessions: " + strconv.Itoa(CfgData.HttpPort))
	logger.Println (logger.INFO, "[sid.config] !==========================================")
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
		logger.Printf (logger.DBG, "[sid.config] %d: `%s=%s`\n", mode, param.Name, param.Value)
		
		if mode != parser.LIST {
			switch param.Name {
				case "LogFile":			CfgData.LogFile = param.Value
				case "LogToFile":		CfgData.LogState = (param.Value == "ON")
				case "LogLevel":		logger.SetLogLevelFromName (param.Value)
				case "CrtlPort":		SetIntValue (&CfgData.CtrlPort, param.Value)
				case "HttpPort":		SetIntValue (&CfgData.HttpPort, param.Value)
				case "Path":			CfgData.Upload.Path = param.Value
				case "Keyring":			CfgData.Upload.Keyring = param.Value
				case "SharePrimeOfs":	SetIntValue (&CfgData.Upload.SharePrimeOfs, param.Value)
				case "ShareTreshold":	SetIntValue (&CfgData.Upload.ShareTreshold, param.Value)
				default:				if CustomConfigHandler != nil {
											return CustomConfigHandler (mode, param)
										}
			}
		} else {
			if CustomConfigHandler != nil {
				return CustomConfigHandler (mode, param)
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
func SetIntValue (trgt *int, data string) {
	if val,err := strconv.Atoi(data); err == nil {
		*trgt = val
	} else {
		logger.Printf (logger.ERROR, "[sid.config] string conversion from '%s' to integer value failed!", data)
	}
}
