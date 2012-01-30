/*
 * Cover server implementation: "imgon.net"
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
)

///////////////////////////////////////////////////////////////////////
// Public types

/*
 * Handler for cover content for "imgon.net" cover site.
 */
type ImgonHandler struct {
	path		string			// path to cover data
}

///////////////////////////////////////////////////////////////////////
// Public functions

/*
 * Create a new cover server instance (imgon.net:80)
 * @return *Cover - pointer to cover server instance
 */
func NewCvrImgon() *Cover {

	// allocate cover server instance
	cover := &Cover {
		server:		"imgon.net:80",
		states:		make (map[net.Conn]*State),
		htmls:		make (map[string]string),
		hdlr:		&ImgonHandler{
						path:	"./images",
					},
	}
	// initialize instance
	// (define replacement pages)
	cover.htmls["/"] = "[UPLOAD]"
	return cover
}

//=====================================================================
/*
 * Get client-side upload form for next cover content.
 * @return string - upload form page content
 */
func (i *ImgonHandler) getForm() string {

	// get next image size
	size := 1234567
	
	// create upload form
	return CreateUploadForm ("upload", size)  
} 


