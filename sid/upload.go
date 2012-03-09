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

package sid

///////////////////////////////////////////////////////////////////////
// Import external declarations.

import (
	"strconv"
	"os"
	"io"
	"big"
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
		logger.Printf (logger.ERROR, "[sid.upload] Can't read keyring file '%s' -- terminating!\n", defs.Keyring)
		os.Exit (1)
	}
	defer rdr.Close()
	
	// read public keys from keyring
	if reviewer,err = openpgp.ReadKeyRing (rdr); err != nil {
		// can't read keys -- terminate!
		logger.Printf (logger.ERROR, "[sid.upload] Failed to process keyring '%s' -- terminating!\n", defs.Keyring)
		os.Exit (1)
	}
}

//=====================================================================
/*
 * Client upload data received.
 * @param data []byte - uploaded document data
 * @return bool - post-processing successful?
 */
func PostprocessUploadData (data []byte) bool {
	logger.Println (logger.INFO, "[sid.upload] Client upload received")
	logger.Println (logger.DBG_ALL, "[sid.upload] Client upload data:\n" + string(data))
	
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
	key := crypto.RandBytes (32)
	if engine,err = aes.NewCipher (key); err != nil {
		// should not happen at all; epic fail if it does
		logger.Println (logger.ERROR, "[sid.upload] Failed to setup AES cipher!")
		return false
	}
	engine.Reset()
	bs := engine.BlockSize()
	iv := crypto.RandBytes (bs)
	enc := cipher.NewCFBEncrypter (engine, iv)

	logger.Println (logger.DBG_ALL, "[sid.upload] key:\n" + hex.Dump(key))
	logger.Println (logger.DBG_ALL, "[sid.upload] IV:\n" + hex.Dump(iv))
	
	//-----------------------------------------------------------------
	// encrypt client document into file
	//-----------------------------------------------------------------
	
	// open file for output 
	fname := baseName + ".document.aes256"
	if wrt,err = os.OpenFile (fname, os.O_WRONLY | os.O_CREATE | os.O_TRUNC, 0666); err != nil {
		logger.Printf (logger.ERROR, "[sid.upload] Can't create document file '%s'\n", fname)
		return false
	}
	// write iv first
	wrt.Write (iv)
	// encrypt binary data for the document
	logger.Println (logger.DBG_ALL, "[sid.upload] AES256 in:\n" + hex.Dump(data))
	enc.XORKeyStream (data, data)
	logger.Println (logger.DBG_ALL, "[sid.upload] AES256 out:\n" + hex.Dump(data))
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
			logger.Printf (logger.ERROR, "[sid.upload] Can't create share file '%s'\n", fname)
			continue
		}
		// create PGP armorer
		if ct,err = armor.Encode (wrt, "PGP MESSAGE", nil); err != nil {
			logger.Printf (logger.ERROR, "[sid.upload] Can't create armorer: %s\n", err.String())
			wrt.Close()
			continue
		}
		// encrypt share to file	
		recipient[0] = ent
		if pt,err = openpgp.Encrypt (ct, recipient, nil, nil); err != nil {
			logger.Printf (logger.ERROR, "[sid.upload] Can't create encrypter: %s\n", err.String())
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
