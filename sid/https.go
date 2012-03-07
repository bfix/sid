/*
 * Handle HTTPS session for non-TOR document uploads.
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
	"http"
	"strconv"
	"strings"
	"sid_custom"
	"gospel/logger"
)

///////////////////////////////////////////////////////////////////////
/*
 * Handle HTTPS requests.
 * @param resp http.ResponseWriter - response buffer
 * @param req *http.Request - request data
 */
func handler (resp http.ResponseWriter, req *http.Request) {

	// call custom handler first
	if sid_custom.HandleCustomResources (resp, req) {
		return
	}

	// get requested resource reference
	ref := req.URL.String()
	mth := req.Method
	logger.Println (logger.DBG_ALL, "[https] " + mth + " " + ref)
	
	//-----------------------------------------------------------------
	// check for POST request (document upload)
	//-----------------------------------------------------------------
	if mth == "POST" {
		// get upload data
		rdr,_,err := req.FormFile ("file")
		if err != nil {
			logger.Println (logger.INFO, "[https] Error accessing uploaded file: " + err.String())
			// show error page
			ref = "/upload_err.html"
			mth = "GET"
		} else {
			content := make ([]byte, 0)
			if err = processStream (rdr, 4096, func (data []byte) bool {
				content = append (content, data...)
				return true
			}); err != nil {
				logger.Println (logger.INFO, "[https] Error accessing uploaded file: " + err.String())
				// show error page
				ref = "/upload_err.html"
				mth = "GET"
			} else {
				// post-process uploaded document
				PostprocessUploadData (content)
				// set resource ref to response page
				ref = "/upload_resp.html"
				mth = "GET"
			}
		}
	}
	
	//-----------------------------------------------------------------
	// handle GET requests
	//-----------------------------------------------------------------
	if mth == "GET" {
		// set default page
		if ref == "/" {
			ref = "/index.html"
		}
		// handle resource file
		switch {
			case strings.HasSuffix (ref, ".html"):		resp.Header().Set ("Content-Type", "text/html")
		}
		if err := processFile ("./www" + ref, 4096, func (data []byte) bool {
			// append data to response buffer
			resp.Write (data)
			return true
		}); err != nil {
			logger.Println (logger.ERROR, "[https] Resource failure: " + err.String())
		}
	}
}

///////////////////////////////////////////////////////////////////////
/*
 * Start-up the HTTPS server instance.
 */
func httpsServe() {

	// define handlers
	http.HandleFunc ("/", handler)
	
	// start server
	logger.Println (logger.INFO, "[https] Starting server.")
	addr := ":" + strconv.Itoa(CfgData.HttpsPort)
	if err := http.ListenAndServeTLS (addr, CfgData.HttpsCert, CfgData.HttpsKey, nil);  err != nil {
		logger.Println (logger.ERROR, "[https] " + err.String())
	}
}
