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
 * It encapsulates a list of POST content definitions with a key
 * value that is the boundary specification of the POST content.
 */
type ImgonHandler struct {
	posts	map[string]([]byte)	// list of cover POST replacements
}

///////////////////////////////////////////////////////////////////////
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
						posts:	make(map[string]([]byte)),
					},
	}
	// initialize instance
	// (define replacement pages)
	cover.htmls["/"] = "[UPLOAD]"
	return cover
}

///////////////////////////////////////////////////////////////////////
/*
 * Get client-side upload form for next cover content.
 * @return string - upload form page content
 *
 * =====================================
 * POST request format for cover server:
 * =====================================
 *
 *-----------------------------<boundary>
 *Content-Disposition: form-data; name="imgUrl"
 *<nl>
 *<nl>
 *-----------------------------<boundary>
 *Content-Disposition: form-data; name="fileName[]"
 *<nl>
 *<nl>
 *-----------------------------<boundary>
 *Content-Disposition: form-data; name="file[]"; filename="<name>"
 *Content-Type: <mime>
 *<nl>
 *<content>
 *-----------------------------<boundary>
 *Content-Disposition: form-data; name="alt[]"
 *<nl>
 *<description>
 *-----------------------------<boundary>
 *Content-Disposition: form-data; name="new_width[]"
 *<nl>
 *<nl>
 *-----------------------------<boundary>
 *Content-Disposition: form-data; name="new_height[]"
 *<nl>
 *<nl>
 *-----------------------------<boundary>
 *Content-Disposition: form-data; name="submit"
 *<nl>
 *Upload
 *-----------------------------<boundary>--
 *<nl>
 */
func (i *ImgonHandler) getForm() string {

	// get random image and boundary id
	img := GetNextImage()
	boundary := CreateId (30)
	
	// build POST content suitable for upload to cover site
	// and save it in the handler structure
	lb := "\r\n"
	lb2 := lb + lb
	lb3 := lb2 + lb
	sep := "-----------------------------" + boundary
	post :=
		sep + lb +
		"Content-Disposition: form-data; name=\"imgUrl\"" + lb3 +
		sep + lb +
		"Content-Disposition: form-data; name=\"fileName[]\"" + lb3 +
		sep + lb +
		"Content-Disposition: form-data; name=\"file[]\"; filename=\"" + img.name + "\"" + lb +
 		"Content-Type: " + img.mime + lb2 +
 		string(GetUploadContent (img.path)) + lb +
		sep + lb +
		"Content-Disposition: form-data; name=\"alt[]\"\n\n" +
 		img.comment + lb +
		sep + lb +
 		"Content-Disposition: form-data; name=\"new_width[]\"" + lb3 +
		sep + lb +
		"Content-Disposition: form-data; name=\"new_height[]\"" + lb3 +
		sep + lb +
		"Content-Disposition: form-data; name=\"submit\"" + lb2 + "Upload" + lb +
		sep + "--" + lb2
	
	i.posts[boundary] = []byte(post)

	// create upload form
	return CreateUploadForm ("/upload/" + boundary, len(i.posts[boundary])+32)
} 

//=====================================================================
/*
 * get cover site POST content for given boundary id.
 * @param id string - boundary id (key used to store POST content)
 * @return []byte - POST content
 */
func (i *ImgonHandler) getPostContent (id string) []byte {
	if post,ok := i.posts[id]; ok {
		// delete POST from list
		i.posts[id] = nil,false
		return post
	}
	return nil
}


