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
	"io"
	"big"
	"xml"
	"rand"
	"time"
	"strings"
	"encoding/hex"
	"crypto/aes"
	"crypto/cipher"
	"crypto/openpgp"
	"crypto/openpgp/armor"
	"gospel/logger"
	"gospel/crypto"
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
 * Image definition (XML).
 */
type ImageDef struct {
	//-----------------------------------------------------------------
	// XML mapped fields
	//-----------------------------------------------------------------
	Name	string
	Comment	string
	Path	string
	Mime	string
}
/*
 * Image definition (List).
 */
type ImageRef struct {
	name	string
	comment	string
	path	string
	mime	string
	size	int
} 

//=====================================================================
/*
 * List of known image references.
 */
var imgList []*ImageRef

//---------------------------------------------------------------------
/*
 * Initialize image handler: read image definitions from the file
 * specified by the "defs" argument.
 * @param defs string - name of XML-based image definitions 
 */
func InitImageHandler (defs string) {

	// prepare parsing of image references
	imgList = make ([]*ImageRef, 0)
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
		// clone to reference instance
		ir := &ImageRef {
			name:		img.Name,
			comment:	img.Comment,
			path:		img.Path,
			mime:		img.Mime,
			size:		int(fi.Size),
		}
		// add to image list
		imgList = append (imgList, ir)
	}
	logger.Printf (logger.INFO, "[upload] %d images available\n", len(imgList))
}

//---------------------------------------------------------------------
/*
 * Get next (random) image from repository
 * @return *ImageRef - reference to (random) image
 */
func GetNextImage() *ImageRef {
	return imgList [rnd.Int() % len(imgList)] 
}

///////////////////////////////////////////////////////////////////////
// Document handling: Store and encrypt client uploads
///////////////////////////////////////////////////////////////////////

var uploadPath string = "./uploads" 
var reviewer openpgp.EntityList = nil
var treshold int = 2
var prime *big.Int = nil

func InitDocumentHandler (defs UploadDefs) {

	// initialize upload handling parameters
	uploadPath = defs.Path
	treshold = defs.ShareTreshold
	
	// compute prime: (2^512-1) - SharePrimeOfs
	one := big.NewInt(1)
	ofs := big.NewInt(int64(defs.SharePrimeOfs))
	prime = new(big.Int).Lsh(one, 512)
	prime = new(big.Int).Sub(prime, one)
	prime = new(big.Int).Sub(prime, ofs)
	
	// open keyring file
	rdr,err := os.Open (defs.Keyring)
	if err != nil {
		// can't read keys -- terminate!
		logger.Printf (logger.ERROR, "[upload] Can't read keyring file '%s' -- terminating!\n", defs.Keyring)
		os.Exit (1)
	}
	defer rdr.Close()
	
	// read public keys from keyring
	if reviewer,err = openpgp.ReadKeyRing (rdr); err != nil {
		// can't read keys -- terminate!
		logger.Printf (logger.ERROR, "[upload] Failed to process keyring '%s' -- terminating!\n", defs.Keyring)
		os.Exit (1)
	}
}

//=====================================================================
/*
 * Client upload data received.
 * @param doc string - uploaded document data
 * @return bool - post-processing successful?
 */
func PostprocessUploadData (doc string) bool {
	logger.Println (logger.INFO, "[upload] Client upload received")
	logger.Println (logger.DBG_ALL, "[upload] Client upload data:\n" + doc)
	
	var (
		err os.Error
		engine *aes.Cipher = nil
		wrt io.WriteCloser = nil
		ct io.WriteCloser = nil
		pt io.WriteCloser = nil
	)
	baseName := uploadPath + "/" + CreateId (16)
	
	//-----------------------------------------------------------------
	// setup AES-256 for encryption
	//-----------------------------------------------------------------
	key := make ([]byte, 32)
	for n := 0; n < 32; n++ {
		key[n] = byte(rnd.Int() & 0xFF)
	}
	if engine,err = aes.NewCipher (key); err != nil {
		// should not happen at all; epic fail if it does
		logger.Println (logger.ERROR, "[upload] Failed to setup AES cipher!")
		return false
	}
	engine.Reset()
	bs := engine.BlockSize()
	iv := make ([]byte, bs)
	for n := 0; n < bs; n++ {
		iv[n] = byte(rnd.Int() & 0xFF)
	}
	enc := cipher.NewCFBEncrypter (engine, iv)

	logger.Println (logger.DBG_ALL, "[upload] key:\n" + hex.Dump(key))
	logger.Println (logger.DBG_ALL, "[upload] IV:\n" + hex.Dump(iv))
	
	//-----------------------------------------------------------------
	// encrypt client document into file
	//-----------------------------------------------------------------
	
	// open file for output 
	fname := baseName + ".document.aes256"
	if wrt,err = os.OpenFile (fname, os.O_WRONLY | os.O_CREATE | os.O_TRUNC, 0666); err != nil {
		logger.Printf (logger.ERROR, "[upload] Can't create document file '%s'\n", fname)
		return false
	}
	// write iv first
	wrt.Write (iv)
	// encrypt binary data for the document
	data := []byte(doc)
	logger.Println (logger.DBG_ALL, "[upload] AES256 in:\n" + hex.Dump(data))
	enc.XORKeyStream (data, data)
	logger.Println (logger.DBG_ALL, "[upload] AES256 out:\n" + hex.Dump(data))
	// write to file
	wrt.Write (data)
	wrt.Close()

	//-----------------------------------------------------------------
	//	create shares from secret
	//-----------------------------------------------------------------
	secret := new(big.Int).SetBytes (key)
	n := len(reviewer)
	shares := crypto.Split (secret, prime, n, treshold)
	recipient := make ([]*openpgp.Entity, 1)
	
	for i,ent := range reviewer {
		// generate filename based on key id
		id := strconv.Uitob64 (ent.PrimaryKey.KeyId & 0xFFFFFFFF, 16)
		fname = baseName + "." + strings.ToUpper(id) + ".gpg"
		// create file for output
		if wrt,err = os.OpenFile (fname,  os.O_WRONLY | os.O_CREATE | os.O_TRUNC, 0666); err != nil {
			logger.Printf (logger.ERROR, "[upload] Can't create share file '%s'\n", fname)
			continue
		}
		// create PGP armorer
		if ct,err = armor.Encode (wrt, "PGP MESSAGE", nil); err != nil {
			logger.Printf (logger.ERROR, "[upload] Can't create armorer: %s\n", err.String())
			wrt.Close()
			continue
		}
		// encrypt share to file	
		recipient[0] = ent
		if pt,err = openpgp.Encrypt (ct, recipient, nil, nil); err != nil {
			logger.Printf (logger.ERROR, "[upload] Can't create encrypter: %s\n", err.String())
			ct.Close()
			wrt.Close()
			continue
		}
		pt.Write ([]byte(shares[i].P.String() + "\n"))
		pt.Write ([]byte(shares[i].X.String() + "\n"))
		pt.Write ([]byte(shares[i].Y.String() + "\n"))
		pt.Close()
		ct.Close()
		wrt.Close()
	}
	// report success
	return true
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
 * Create an numeric identifier of given length
 * @param size int - number of digits
 * @return string - new identifier
 */
func CreateId (size int) string {
	id := ""
	for len(id) < size {
		id += string('1' + (rnd.Int() % 9))
	}
	return id
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
