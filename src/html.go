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
	"io"
	"os"
	"html"
	"gospel/logger"
)

///////////////////////////////////////////////////////////////////////
// Constants

const (
	htmlIntro	= "<!DOCTYPE HTML>\n<html><body>"
	htmlOutro	= "</body></html>"
)

///////////////////////////////////////////////////////////////////////
/*
 * Tag represents all HTML tags from a cover server response (content)
 * that refer to an external ressource and therefore must be conserved
 * and translated to match the profile of a "normal" usage of the cover
 * site. (Resources are replaces by "innocent" and "unharnful" content
 * on the fly during the response handling for non-HTML ressources)
 */
type Tag struct {
	name	string
	attrs	map[string]string
}

//---------------------------------------------------------------------
/*
 * Instantiate a new Tag object with given parameters.
 * @param n string - name of tag
 * @param a map[string]string - list of attributes
 * @return *Tag - pointer to new instance
 */
func NewTag (n string, a map[string]string) *Tag {
	return &Tag {
		name:	n,
		attrs:	a,
	}
}

//---------------------------------------------------------------------
/*
 * Stringify tag
 * @return string - string representation of tag
 */
func (t *Tag) String() string {
	res := "<" + t.name
	for key,val := range t.attrs {
		res += " " + key + "=" + val
	}
	return res + "/>"
}

///////////////////////////////////////////////////////////////////////
/*
 * List of tags.
 */
type TagList struct {
	list	[]*Tag
}

//---------------------------------------------------------------------
/*
 * Create a new (empty) list of tags.
 * @return *TagList - reference to new tag list
 */
func NewTagList() *TagList {
	return &TagList {
		list:	make ([]*Tag, 0),	
	}
}

//---------------------------------------------------------------------
/*
 * Add a new tag to the list.
 * @param name string - name of tag
 * @param attr map[string]string - list of attributes
 */
func (t *TagList) Put (tag *Tag) {
	t.list = append (t.list, tag)
}

//---------------------------------------------------------------------
/*
 * Get and remove next tag from list.
 * @return *Tag - reference to tag
 */
func (t *TagList) Get() *Tag {
	if len(t.list) == 0 {
		return nil
	}
	tag := t.list[0]
	t.list = t.list[1:]
	return tag
}

//---------------------------------------------------------------------
/*
 * Get number of tags in list.
 * @return int - number of tags available
 */
func (t *TagList) Count() int {
	return len(t.list)
}

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
 * @param rdr *io.Reader - buffered reader for parsing
 * @param list map[string]string - (current) list of inline ressources
 */
func parseHTML (rdr io.Reader, list *TagList) {
	
	// try to use GO html tokenizer to parse the content
	tk := html.NewTokenizer (rdr)
	loop: for {
		// get next HTML tag
		toktype := tk.Next()
		if toktype == html.ErrorToken {
			// parsing error: most probable case is that the tag spans
			// fragments. This is only a problem if it concerns a tag
			// for possible translation (currently unhandled)
			switch tk.Error() {
				case io.ErrUnexpectedEOF:
					logger.Println (logger.ERROR, "[html] Error parsing content: " + tk.Error().String())
				case os.EOF:
					break loop
			}
		}
		if toktype == html.StartTagToken || toktype == html.SelfClosingTagToken {
			// we are only interested in certain tags
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
						list.Put (t)
						logger.Println (logger.DBG, "[html] => " + t.String())
					}

				//-----------------------------------------------------
				// external image
				//-----------------------------------------------------
				case "img":
					attrs := getAttrs (tk)
					t := NewTag ("img", attrs)
					list.Put (t)
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

//---------------------------------------------------------------------
/*
 * Generate an error page.
 * @param severe bool - unrecoverable error?
 * @return string - error page
 */
func errorBody (severe bool) string {
	if severe {
		return  "<h1>Severe error occurred</h1>\n" +
				"Please return to previous page and try again later!\n"
	}
	return  "<h1>Error occurred</h1>\n" +
			"Please return to previous page and try again.\n"
}
