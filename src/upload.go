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
	"os"
	"xml"
	"rand"
	"time"
	"gospel/logger"
)

///////////////////////////////////////////////////////////////////////
// static variables 

var rnd	= rand.New (rand.NewSource (time.UTC().Nanoseconds()))

///////////////////////////////////////////////////////////////////////
// Public types

/*
 * UploadHandler: Generate next client-side upload form that matches
 * the next cover content for upload to the cover server.
 */
type UploadHandler interface {
	getForm()					string		// get upload form
	getPostContent (id string)	[]byte		// get cover POST content
}

///////////////////////////////////////////////////////////////////////
// Image handler: Provide access to (annotated) image content
// to be used as cover content in upload procedures. Annotations
// include mime type, comments, etc. pp.
///////////////////////////////////////////////////////////////////////
/*
 * The XML definition file for image references looks like this:
 *
 *<?xml version="1.0" encoding="UTF-8"?>
 *<images>
 *    <image>
 *        <name>Test</name>
 *        <comment>blubb</comment>
 *        <path>./image/img001.gif</path>
 *        <mime>image/gif</mime>
 *    </image>
 *</images>
 */

//=====================================================================
/*
 * List of images (for XML parsing)
 */
type ImageList struct {
	Image	[]ImageDef
}
//=====================================================================
/*
 * Image definition.
 */
type ImageDef struct {
	//-----------------------------------------------------------------
	// XML mapped fields
	//-----------------------------------------------------------------
	Name	string
	Comment	string
	Path	string
	Mime	string
	//-----------------------------------------------------------------
	// additional fields
	//-----------------------------------------------------------------
	size	int
} 

//=====================================================================
/*
 * List of known image references.
 */
var imgList []*ImageDef

//---------------------------------------------------------------------
/*
 * Initialize image handler: read image definitions from the file
 * specified by the "defs" argument.
 * @param defs string - name of XML-based image definitions 
 */
func InitImageHandler (defs string) {

	// prepare parsing of image references
	imgList = make ([]*ImageDef, 0)
	rdr,err := os.Open (defs)
	if err != nil {
		// terminate application in case of failure
		logger.Println (logger.ERROR, "[upload] Can't read image definitions -- terminating!")
		os.Exit (1)
	}
	defer rdr.Close()

	// parse XML file and build image reference list
	var list ImageList
	xml.Unmarshal (rdr, &list)	
	for _,img := range list.Image {
		logger.Println (logger.DBG, "[upload]: image=" + img.Name)
		// get size of image file
		fi,err := os.Stat (img.Path)
		if err != nil {
			logger.Println (logger.ERROR, "[upload] image '" + img.Path + "' missing!")
			continue
		}
		img.size = int(fi.Size)
		// add to image list
		imgList = append (imgList, &img)
	}
	logger.Printf (logger.INFO, "[upload] %d images available\n", len(imgList))
}

//---------------------------------------------------------------------
/*
 * Get next (random) image from repository
 * @return *ImageDef - reference to (random) image
 */
func GetNextImage() *ImageDef {
	return imgList [rnd.Int() % len(imgList)] 
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
					"e=document.u.file.value.length;" +
					"c=Math.ceil(3*(b.substring(b.indexOf(\",\")+1).length+3)/4);" +
					"d=\"\";for(i=0;i<" + strconv.Itoa(total) + "-c-e-307;i++){d+=b.charAt(i%c)}" +
					"document.u.rnd.value=d;" +
					"document.u.submit();" +
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
			"<hr/>\n"
}

//=====================================================================
/*
 * Create a boundary name for multipart POST contents.
 * @param size int - number of digits in boundary id
 * @return string - new boundary name
 */
func CreateBoundary (size int) string {
	boundary := ""
	for len(boundary) < size {
		boundary += string('1' + (rnd.Int() % 9))
	}
	return boundary
}

//=====================================================================
/*
 * Assemble base64-encoded upload content string.
 * @param fname string - name of file with content data
 * @return []byte - binary content
 */
func GetUploadContent (fname string) []byte {

	rdr,err := os.Open (fname)
	if err != nil {
		logger.Println (logger.ERROR, "[upload] Failed to open upload file: " + fname)
		return nil
	}
	defer rdr.Close()
	data := make([]byte, 32768)
	content := make ([]byte, 0)
	for {
		// read next chunk of data
		num,_ := rdr.Read (data)
		if num == 0 {
			break
		}
		content = append (content, data[0:num]...)
	}
	return content
}

//=====================================================================
/*
 * Client upload data received.
 * @param data string - uploaded data
 */
func PostprocessUploadData (data string) {
	logger.Println (logger.INFO, "[upload] Client upload received:\n" + data + "\n")
}
