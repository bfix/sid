/*
 * Cover server implementation: porn picture post
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
// Public functions

/*
 * Create a new cover server instance (www.picpost.com:80)
 * @return *Cover - pointer to cover server instance
 */
func NewCvrPicpost() *Cover {

	// allocate cover server instance
	cover := &Cover {
		server:		"www.picpost.com:80",
		states:		make (map[net.Conn]*State),
		htmls:		make (map[string]string),
		htmlIn:		"<html><body>",
		htmlOut:	"</body></html>",
	}
	
	// initialize instance
	cover.htmls["/"] = "" +
		"<h1>Welcome to the survey upload</h1>" +
		"<p>Please use the following link to progress to the upload form:<p>" +
		"<a href=\"upload.html\">Link</a>"
	
	return cover
}
