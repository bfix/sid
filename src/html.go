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
	"html"
	"gospel/logger"
)

///////////////////////////////////////////////////////////////////////
// Helper functions and methods.

/*
 * Handle HTML-like content: since we can't be sure that the body is error-
 * free* HTML, we need a lazy parser for the fields we are interested in.
 * The parser builds a list of inline ressources referenced in the HTML file;
 * these are the ressources the client browser will request when loading the
 * HTML page, so that it behaves like a genuine access if monitored by an
 * eavesdropper. This function adds to an existing list of ressources (from
 * a previous cover server response).
 * @param rdr *bufio.Reader - buffered reader for parsing
 * @param list map[string]string - (current) list of inline ressources
 */
func parseHTML (rdr *bufio.Reader, list []*Tag) {
	
	// try to use GO html tokenizer to parse the content
	tk := html.NewTokenizer (rdr)
	for {
		// get next HTML tag
		toktype := tk.Next()
		if toktype == html.ErrorToken {
			// parsing error: most probable case is that the tag spans
			// fragments. This is only a problem if it concerns a tag
			// for possible translation (currently unhandled)
			logger.Println (logger.ERROR, "[html] Error parsing content")
			break
		}
		if toktype == html.StartTagToken || toktype == html.SelfClosingTagToken {
			// we are only interessted in certain tags
			tag,_ := tk.TagName()
			name := string(tag)
			switch name {
				//-----------------------------------------------------
				// external script files
				//-----------------------------------------------------
				case "script":
					attrs := getAttrs (tk)
					if _,ok := attrs["src"]; ok {
						// add external reference to script file
						t := NewTag ("script", attrs)
						list = append (list, t)
						logger.Println (logger.DBG, "[html] => " + t.String())
					}

				//-----------------------------------------------------
				// external image
				//-----------------------------------------------------
				case "img":
					attrs := getAttrs (tk)
					t := NewTag ("img", attrs)
					list = append (list, t)
					logger.Println (logger.DBG, "[html] => " + t.String())
					
				//-----------------------------------------------------
				// unknown
				//-----------------------------------------------------
				default:
					logger.Println (logger.DBG_ALL, "*** " + name)
			}
		}
	}
}

//---------------------------------------------------------------------
/*
 * Get list of attributes for a tag
 * @param tk *html.Tokenizer - tokenizer instance
 * @return map[string]string - list of attributes
 */
func getAttrs (tk *html.Tokenizer) map[string]string {
	list := make (map[string]string)			
	for {
		key,val,more := tk.TagAttr()
		list[string(key)] = string(val)
		if !more {
			break
		}	
	}
	return list
}

//---------------------------------------------------------------------
/*
 * Generate HTML padding sequence.
 * @param size int - length of padding sequence
 * @return string - padding sequence 
 */
func padding (size int) string {
	// small paddings are simple spaces...
	if size < 9 {
		s := ""
		for n := 0; n < size; n++ {
			s += " "
		}
		return s
	}
	// lomger paddings are wrapped into a HTML comment
	size -= 9
	s := "<!-- "
	for n := 0; n < size; n++ {
		s += "?"
	}
	return s + " -->"
}
