/*
 * HTML processing helpers
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
	"bufio"
)

///////////////////////////////////////////////////////////////////////
// Helper functions and methods.

/*
 * Handle HTML-like content: since we can't be sure that the body is error-
 * free* HTML, we need a lazy parser for the fields we are interested in.
 * The parser builds a list of inline ressources referenced in the HTML file;
 * these are the ressources the client browser will request when loading the
 * HTML page, so that it behaves like a genuine access if monitored by an
 * eavesdropper.
 * @param rdr *bufio.Reader - buffered reader for parsing
 * @return []string - list of inline ressources
 */
func parseHTML (rdr *bufio.Reader) []string {
	list := make ([]string, 0)
	return list
}

//---------------------------------------------------------------------
/*
 * Assemble a response from the current state (like response header),
 * the resource list and a replacement body (addressed by the requested
 * ressource path from state).
 * @param s *state - current state info
 * @param resList []string - list of ressources to be included in response
 * @param size int - target size of response
 * @return []byte - assembled response
 */
func (c *Cover) assembleHTML (s *state, resList []string, size int) []byte {
	resp := make ([]byte, 0)
	return resp
}
