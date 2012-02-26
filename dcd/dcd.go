/*
 * Decrypt client document with shared secrets.
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
	"fmt"
	"flag"
	"os"
	"bufio"
	"big"
	"strings"
	"crypto/aes"
	"crypto/cipher"
	"gospel/crypto"
)

///////////////////////////////////////////////////////////////////////
// Main application entry point
/*
 * Stand-alone decryption application for client documents.
 */
func main() {

	// handle command line arguments
	flag.Parse()
	args := flag.Args()
	count := len(args)
	if count < 2 {
		fmt.Println ("At least two arguments are expected -- abort!")
		fmt.Println ("dcd <document.aes256> <share1> [ ... <shareN> ]")
		return 
	}
	
	// read shares
	shares := make ([]crypto.Share, count-1)
	for n := 1; n < count; n++ {
		f,err := os.Open (args[n])
		if err != nil {
			fmt.Printf  ("Failed to open file '%s' -- abort!\n", args[n])
			fmt.Println ("Error: " + err.String())
			os.Exit (1)
		}
		rdr := bufio.NewReader (f)
		p,_ := new(big.Int).SetString (GetLine (rdr, args[n]), 10)
		x,_ := new(big.Int).SetString (GetLine (rdr, args[n]), 10)
		y,_ := new(big.Int).SetString (GetLine (rdr, args[n]), 10)
		shares[n-1] = crypto.Share { x, y, p }
		f.Close()
	}
	
	// recover key and create cipher engine
	secret := crypto.Reconstruct (shares)
	key := secret.Bytes()
	engine,err := aes.NewCipher (key)
	if err != nil {
		fmt.Println  ("Failed to create AES cipher engine -- abort!")
		fmt.Println ("Error: " + err.String())
		os.Exit (1)
	}
	engine.Reset()	
	
	// decrypt document
	rdr,err := os.Open (args[0])
	if err != nil {
		fmt.Printf  ("Failed to open file '%s' -- abort!\n", args[0])
		fmt.Println ("Error: " + err.String())
		os.Exit (1)
	}
	defer rdr.Close()
	parts := strings.Split (args[0], ".")
	if parts[1] != "document" || parts[2] != "aes256" {
		fmt.Printf  ("Invalid document file name '%s' -- abort!\n", args[0])
		os.Exit (1)
	}
	fname := parts[0] + ".document"
	wrt,err := os.OpenFile (fname, os.O_WRONLY | os.O_CREATE | os.O_TRUNC, 0666)
	if err != nil {
		fmt.Printf  ("Failed to open output file '%s' -- abort!\n", fname)
		fmt.Println ("Error: " + err.String())
		os.Exit (1)
	}
	defer wrt.Close()
		
	bs := engine.BlockSize()
	iv := make ([]byte, bs)
	rdr.Read (iv)
	dec := cipher.NewCFBDecrypter (engine, iv)

	data := make ([]byte, 32768)
	for {
		num,err := rdr.Read (data)
		if err != nil {
			if err == os.EOF {
				break
			}
			fmt.Printf  ("Failed to read encrypted file '%s' -- abort!\n", args[0])
			fmt.Println ("Error: " + err.String())
			os.Exit (1)
		}
		dec.XORKeyStream (data[0:num], data[0:num])
		num,err = wrt.Write (data[0:num])
		if err != nil {
			fmt.Printf  ("Failed to write decrypted file '%s' -- abort!\n", fname)
			fmt.Println ("Error: " + err.String())
			os.Exit (1)
		}
	}			
}

///////////////////////////////////////////////////////////////////////
/*
 * Read line from reader.
 */
func GetLine (rdr *bufio.Reader, fname string) string {
	line,_,err := rdr.ReadLine()
	if err != nil {
		fmt.Printf  ("Failed to read from file '%s' -- abort!\n", fname)
		fmt.Println ("Error: " + err.String())
		os.Exit (1)
	}
	return string(line)
}
