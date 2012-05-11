/*
 * Utilities and helpers.
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
	"gospel/crypto"
	"io"
	"os"
)

///////////////////////////////////////////////////////////////////////
/*
 * Process the binary content of a file in chunks of specified size.
 * A callback function is invoked for every chunk. If the callback
 * returns false, the file processing is aborted.
 * @param fname string - name of file
 * @param chunkSize int - max. size of data blobs for callback handler
 * @param hdlr func (data []byte) bool - callback handler
 * @return error - error object (or nil)
 */
func ProcessFile(fname string, chunkSize int, hdlr func(data []byte) bool) error {

	// open file
	file, err := os.Open(fname)
	if err != nil {
		return err
	}
	defer file.Close()

	// process content
	return ProcessStream(file, chunkSize, hdlr)
}

///////////////////////////////////////////////////////////////////////
/*
 * Process the binary stream in chunks of specified size.
 * A callback function is invoked for every chunk. If the callback
 * returns false, the file processing is aborted.
 * @param rdr io.Reader - source reader
 * @param chunkSize int - max. size of data blobs for callback handler
 * @param hdlr func (data []byte) bool - callback handler
 * @return error - error object (or nil)
 */
func ProcessStream(rdr io.Reader, chunkSize int, hdlr func(data []byte) bool) error {

	// process file	
	data := make([]byte, chunkSize)
	for {
		// read next chunk
		n, err := rdr.Read(data)
		// end of file reached?
		if n == 0 {
			// yes: done
			break
		}
		// handle error
		if err != nil {
			return err
		}
		// let callback handle the data
		if !hdlr(data[:n]) {
			break
		}
	}
	// report success.
	return nil
}

///////////////////////////////////////////////////////////////////////
/*
 * Create a decimal number of given length to be used as an identifier.
 * @param size int - desired length of identifier
 * @return string - generated number string
 */
func CreateId(size int) string {
	id := string('1' + crypto.RandInt(0, 8))
	for len(id) < size {
		id += string('0' + crypto.RandInt(0, 9))
	}
	return id
}

//---------------------------------------------------------------------
/*
 * Create a key of given length to be used as an identifier.
 * The character set is [a-zA-Z0-9]
 * @param size int - desired length of key
 * @return string - generated number string
 */
func CreateKey(size int) string {
	id := ""
	for len(id) < size {
		v := crypto.RandInt(0, 61)
		if v < 26 {
			id += string('a' + v)
		} else if v < 52 {
			id += string('A' + v - 26)
		} else {
			id += string('0' + v - 52)
		}
	}
	return id
}
