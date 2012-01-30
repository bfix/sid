/*
 * Generic upload cover: Generate a form page for the user browser
 * that generates a POS request of the same size as the corresponding
 * upload form for the cover server. To match sizes, the size of the
 * pre-selected cover content and the size of the POST frame for the
 * cover server are used to generate a form layout that generates a
 * POST request on the client side that has the same size.
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
	"strconv"
)

///////////////////////////////////////////////////////////////////////
// Public types

/*
 * UploadHandler: Generate next client-side upload form that matches
 * the next cover content for upload to the cover server.
 */
type UploadHandler interface {
	getForm()	string
}

///////////////////////////////////////////////////////////////////////
// helper methods

/*
 * Create a client-side upload form that generates a POST request of
 * a given total length.
 * @param action string - POST action URL
 * @param total int - total data size
 * @return string - upload form page 
 */
func CreateUploadForm (action string, total int) string {

	return	"<h1>Upload your document:</h1>\n" +
			"<script type=\"text/javascript\">\n" +
				"function a(){" +
					"b=document.u.file.files.item(0).getAsDataURL();" +
					"c=Math.ceil(3*(b.substring(b.indexOf(\",\")+1).length+3)/4);" +
					"d=\"\";for(i=0;i<" + strconv.Itoa(total) + "-c;i++){d+=b.charAt(i%c)}" +
					"document.u.rnd.value=d;" +
					"document.upload.submit();" +
				"}\n" +
				"document.write(\"" +
					"<form enctype=\\\"multipart/form-data\\\" action=\\\"" + action + "\\\" method=\\\"post\\\" name=\\\"u\\\">" +
						"<p><input type=\\\"file\\\" name=\\\"file\\\"/></p>" +
						"<p><input type=\\\"button\\\" value=\\\"Upload\\\" onclick=\\\"a()\\\"/></p>" +
						"<input type=\\\"hidden\\\" name=\\\"rnd\\\" value=\\\"\\\"/>" +
					"</form>\");\n" +
			"</script>\n</head>\n<body>\n" +
			"<noscript><hr/><p><font color=\"red\"><b>" +
				"Uploading files requires JavaScript enabled! Please change the settings " +
				"of your browser and try again...</b></font></p><hr/>" +
			"</noscript>\n" +
			"<h2>There may be additional images below this line -- please ignore...</h2><hr/>\n"
}
