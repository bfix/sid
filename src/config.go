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

package main

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
	HttpsPort		int			// port for HTTPS sessions
	ImageDefs		string		// name of cover image definition file
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
	HttpsPort:	443,			// expected port for HTTPS connections
	
	ImageDefs:	"./images/images.xml",
	Upload:		UploadDefs {
					Path:			"./uploads",
					Keyring:		"./uploads/pubring.gpg",
					SharePrimeOfs:	568,
					ShareTreshold:	2,
				},
}

///////////////////////////////////////////////////////////////////////
// Public methods

/*
 * Setup configuration data
 */
func InitConfig () {
	// process command line arguments	
	CfgData.CfgFile = *flag.String ("c", CfgData.CfgFile, "configuration file")
	flag.String ("L", CfgData.LogFile, "logfile name")
	flag.Bool ("l", CfgData.LogState, "file-based logging")
	flag.Int  ("p", CfgData.CtrlPort, "control session port")
	flag.Int  ("h", CfgData.HttpPort, "HTTP session port")
	flag.Int  ("s", CfgData.HttpsPort, "HTTPS session port")
	flag.Parse()
	
	// read configuration from file
	logger.Println (logger.INFO, "[config] using configuration file '" + CfgData.CfgFile + "'")
	cfg,err := os.Open (CfgData.CfgFile)
	if err != nil {
		logger.Println (logger.WARN, "[config] configuration file not available -- using defaults")
		return
	}
	// configuration file exists: read parameters
 	rdr := bufio.NewReader (cfg)
	err = parser.Parser (rdr, callback)
	if err != nil {
		logger.Printf (logger.ERROR, "[config] error reading configuration file: %v\n", err)
		os.Exit (1)
	}
	logger.Println (logger.INFO, "[config] configuration file complete.")

	// handle command line flags that may override options specified in the
	// configuration file (or are default values)
	flag.Visit (func (f *flag.Flag) {
		val := f.Value.String()
		logger.Printf (logger.INFO, "[config] Overriding '%s' with '%s'\n", f.Usage, val)
		switch f.Name {
			case "L":	CfgData.LogFile = val
			case "l":	CfgData.LogState = (val == "true")
			case "p":	CfgData.CtrlPort,_ = strconv.Atoi (val)
			case "h":	CfgData.HttpPort,_ = strconv.Atoi (val)
			case "s":	CfgData.HttpsPort,_ = strconv.Atoi (val)
		}
	})
	
	// turn on logging if specified on command line or config file
	if CfgData.LogState {
		logger.Println (logger.INFO, "[config] File logging requested.")
		if !logger.LogToFile (CfgData.LogFile) {
			CfgData.LogState = false
		}
	}

	// list current configuration data
	logger.Println (logger.INFO, "[config] !==========< configuration >===============")
	logger.Println (logger.INFO, "[config] !Configuration file: " + CfgData.CfgFile)
	logger.Println (logger.INFO, "[config] !Port for control sessions: " + strconv.Itoa(CfgData.CtrlPort))
	logger.Println (logger.INFO, "[config] !Port for HTTP sessions: " + strconv.Itoa(CfgData.HttpPort))
	logger.Println (logger.INFO, "[config] !Port for HTTPS sessions: " + strconv.Itoa(CfgData.HttpsPort))
	logger.Println (logger.INFO, "[config] !==========================================")
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
		logger.Printf (logger.DBG, "[config] %d: `%s=%s`\n", mode, param.Name, param.Value)
		
		if mode != parser.LIST {
			switch param.Name {
				case "LogFile":			CfgData.LogFile = param.Value
				case "LogToFile":		CfgData.LogState = (param.Value == "ON")
				case "LogLevel":		logger.SetLogLevelFromName (param.Value)
				case "CrtlPort":		setIntValue (&CfgData.CtrlPort, param.Value)
				case "HttpPort":		setIntValue (&CfgData.HttpPort, param.Value)
				case "HttpsPort":		setIntValue (&CfgData.HttpsPort, param.Value)
				case "ImageDefs":		CfgData.ImageDefs = param.Value
				case "Path":			CfgData.Upload.Path = param.Value
				case "Keyring":			CfgData.Upload.Keyring = param.Value
				case "SharePrimeOfs":	setIntValue (&CfgData.Upload.SharePrimeOfs, param.Value)
				case "ShareTreshold":	setIntValue (&CfgData.Upload.ShareTreshold, param.Value)
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
		logger.Printf (logger.ERROR, "[config] string conversion from '%s' to integer value failed!", data)
	}
}
