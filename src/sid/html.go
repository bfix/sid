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

package sid

///////////////////////////////////////////////////////////////////////
// Import external declarations.

import (
	"exp/html"
	"gospel/logger"
	"io"
	"strings"
)

///////////////////////////////////////////////////////////////////////
// Constants

const (
	htmlIntro = "<!DOCTYPE HTML>\n<html>\n"
	htmlOutro = "</body>\n</html>\n"
)

///////////////////////////////////////////////////////////////////////
/*
 * Tag represents all HTML tags from a cover server response (content)
 * that refer to an external resource and therefore must be conserved
 * and translated to match the profile of a "normal" usage of the cover
 * site. (Resources are replaces by "innocent" and "unharnful" content
 * on the fly during the response handling for non-HTML resources)
 */
type Tag struct {
	name  string
	attrs map[string]string
}

//---------------------------------------------------------------------
/*
 * Instantiate a new Tag object with given parameters.
 * @param n string - name of tag
 * @param a map[string]string - list of attributes
 * @return *Tag - pointer to new instance
 */
func NewTag(n string, a map[string]string) *Tag {
	return &Tag{
		name:  n,
		attrs: a,
	}
}

//---------------------------------------------------------------------
/*
 * Stringify tag
 * @return string - string representation of tag
 */
func (t *Tag) String() string {

	// create tag representation with name and attributes
	res := "<" + t.name
	for key, val := range t.attrs {
		res += " " + key + "=\"" + val + "\""
	}
	// BUG FIX: for whatever reason Firefox don't like self-enclosing
	// script tags - so this is a work around.
	if t.name == "script" {
		return res + "></script>"
	}
	// create self-enclosing tag
	return res + "/>"
}

//---------------------------------------------------------------------
/*
 * Get value of tag attribute.
 * @param name string - attribute name
 * @return string - attribute value
 */
func (t *Tag) GetAttr(name string) string {
	return t.attrs[name]
}

///////////////////////////////////////////////////////////////////////
/*
 * List of tags.
 */
type TagList struct {
	list []*Tag
}

//---------------------------------------------------------------------
/*
 * Create a new (empty) list of tags.
 * @return *TagList - reference to new tag list
 */
func NewTagList() *TagList {
	return &TagList{
		list: make([]*Tag, 0),
	}
}

//---------------------------------------------------------------------
/*
 * Add a new tag to the list.
 * @param name string - name of tag
 * @param attr map[string]string - list of attributes
 */
func (t *TagList) Put(tag *Tag) {
	t.list = append(t.list, tag)
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
 * Lookup all tags with "name" attribute.
 * @param name string - attribute name to look for
 * @return []*Tag - list of matching tags
 */
func (t *TagList) Lookup(name string) []*Tag {
	out := make([]*Tag, 0)
	for _, e := range t.list {
		if _, ok := e.attrs[name]; ok {
			out = append(out, e)
		}
	}
	return out
}

//---------------------------------------------------------------------
/*
 * Lookup all tags with "name=value" attribute.
 * @param name string - attribute name to look for
 * @param value string - attribute value to look for
 * @return []*Tag - list of matching tags
 */
func (t *TagList) LookupPair(name, value string) []*Tag {
	out := make([]*Tag, 0)
	for _, e := range t.list {
		if e.attrs[name] == value {
			out = append(out, e)
		}
	}
	return out
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
 * The parser builds a list of inline resources referenced in the HTML file;
 * these are the resources the client browser will request when loading the
 * HTML page, so that it behaves like a genuine access if monitored by an
 * eavesdropper. This function adds to an existing list of resources (from
 * a previous cover server response).
 * @param rdr *io.Reader - buffered reader for parsing
 * @param links *TagList - (current) list of inline resources (header)
 * @param list *TagList - (current) list of inline resources (body)
 * @param xtra *TagList - (current) list of inline resources (special interest tags)
 * @return bool - end of parsing (HTML closed)?
 */
func parseHTML(rdr io.Reader, links *TagList, list *TagList, xtra *TagList) bool {

	// try to use GO html tokenizer to parse the content
	tk := html.NewTokenizer(rdr)
	stack := make([]*Tag, 0)
	var tag *Tag
	closed := false
loop:
	for {
		// get next HTML tag
		toktype := tk.Next()
		if toktype == html.ErrorToken {
			// parsing error: most probable case is that the tag spans
			// fragments. This is only a problem if it concerns a tag
			// for possible translation (currently unhandled)
			switch tk.Err() {
			case io.ErrUnexpectedEOF:
				logger.Println(logger.ERROR, "[sid.html] Error parsing content: "+tk.Err().Error())
				return false
			case io.EOF:
				break loop
			}
		}
		if toktype == html.StartTagToken {
			// we are starting a tag. push to stack until EndTagToken is encountered.
			tag = readTag(tk)
			if tag != nil {
				logger.Println(logger.DBG_ALL, "[cover] tag pushed to stack: "+tag.String())
				stack = append(stack, tag)
			}
			continue loop
		} else if toktype == html.EndTagToken {
			n, _ := tk.TagName()
			name := string(n)
			pos := len(stack) - 1
			if pos >= 0 && stack[pos].name == name {
				// found matching tag
				tag = stack[pos]
				stack = stack[0:pos]
				logger.Println(logger.DBG_ALL, "[cover] tag popped from stack: "+tag.String())
			} else {
				if name == "html" {
					logger.Println(logger.DBG_ALL, "body ==> </html>")
					closed = true
				}
				continue loop
			}
		} else if toktype == html.SelfClosingTagToken {
			tag = readTag(tk)
			if tag == nil {
				continue loop
			}
			//logger.Println(logger.DBG_ALL, "[sid.html] direct tag : "+tag.String())
		} else {
			//logger.Println(logger.DBG_ALL, "[sid.html] ? "+toktype.String())
			continue loop
		}

		// post-process tag and add to appropriate tag list		
		switch {
		case tag.name == "img":
			// add/replace dimensions
			tag.attrs["width"] = "1"
			tag.attrs["height"] = "1"
			// add to list
			list.Put(tag)
			logger.Println(logger.DBG, "[sid.html] body => "+tag.String())

		case tag.name == "script":
			list.Put(tag)
			logger.Println(logger.DBG, "[sid.html] body => "+tag.String())

		case tag.name == "link":
			links.Put(tag)
			logger.Println(logger.DBG, "[sid.html] hdr => "+tag.String())

		case tag.name == "input" && tag.attrs["type"] == "hidden":
			xtra.Put(tag)
			logger.Println(logger.DBG, "[sid.html] xtra => "+tag.String())

			//		default:
			//			logger.Println (logger.DBG_ALL, "*** " + tag.String())
		}
	}
	return closed
}

//---------------------------------------------------------------------
/*
 * Read current tag with attributes.
 * @param tk *html.Tokenizer - tokenizer instance
 * @return *Tag - reference to read tag
 */
func readTag(tk *html.Tokenizer) *Tag {

	// we are only interested in certain tags
	tag, _ := tk.TagName()
	name := string(tag)
	switch name {
	//-----------------------------------------------------
	// external script files
	//-----------------------------------------------------
	case "script":
		attrs := getAttrs(tk)
		if attrs != nil {
			if _, ok := attrs["src"]; ok {
				// add external reference to script file
				return NewTag("script", attrs)
			}
		}

	//-----------------------------------------------------
	// external image
	//-----------------------------------------------------
	case "img":
		attrs := getAttrs(tk)
		if attrs != nil {
			return NewTag("img", attrs)
		}

	//-----------------------------------------------------
	// external links (style sheets)
	//-----------------------------------------------------
	case "link":
		attrs := getAttrs(tk)
		if attrs != nil {
			if _, ok := attrs["href"]; ok {
				// add external reference to link
				return NewTag("link", attrs)
			}
		}

	//-----------------------------------------------------
	// input fields
	//-----------------------------------------------------
	case "input":
		attrs := getAttrs(tk)
		if attrs != nil {
			if _, ok := attrs["type"]; ok {
				// add external reference to link
				return NewTag("input", attrs)
			}
		}
	}
	//-----------------------------------------------------
	// ignore all other tags (no tag processed).
	//-----------------------------------------------------
	return nil
}

//---------------------------------------------------------------------
/*
 * Get list of attributes for a tag.
 * If the tag is at the end of a HTML fragment and not all attributes
 * can be read by the tokenizer, this call terminates with a "nil"
 * map to indicate failure. The tag is than dropped (for an eavesdropper
 * this looks like a cached resource)
 * @param tk *html.Tokenizer - tokenizer instance
 * @return map[string]string - list of attributes
 */
func getAttrs(tk *html.Tokenizer) (list map[string]string) {

	// handle panic during parsing
	defer func() {
		if r := recover(); r != nil {
			logger.Printf(logger.WARN, "[sid.html] Skipping fragmented tag: %v\n", r)
			list = nil
		}
	}()

	// parse attributes from HTML text
	list = make(map[string]string)
	for {
		key, val, more := tk.TagAttr()
		list[string(key)] = string(val)
		if !more {
			break
		}
	}
	return
}

//---------------------------------------------------------------------
/*
 * Generate HTML padding sequence.
 * @param size int - length of padding sequence
 * @return string - padding sequence 
 */
func padding(size int) string {
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
func errorBody(severe bool) string {
	if severe {
		return "<h1>Severe error occurred</h1>\n" +
			"Please return to previous page and try again later!\n"
	}
	return "<h1>Error occurred</h1>\n" +
		"Please return to previous page and try again.\n"
}

//=====================================================================
/*
 * Translate URI (external <-> local): 
 * Any URI of the form "<scheme>://<server>/<path>/<to>/<resource...>"
 * is transformed to an absolute path on on the sending server (that is
 * the SID instance) that can later be translated back to its original
 * form; it looks like "/&<scheme>/<server>/<path>/<to>/<resource...>"
 * @param uri string - incoming uri
 * @return string - translated uri
 */
func translateURI(uri string) string {

	if pos := strings.Index(uri, "://"); pos != -1 {
		// we have an absolute URI that needs translation
		scheme := string(uri[0:pos])
		res := string(uri[pos+2:])
		return "/&" + scheme + res
	} else if strings.HasPrefix(uri, "/&") {
		// we have a local URI that needs translation
		pos := strings.Index(string(uri[2:]), "/")
		scheme := string(uri[2 : pos+2])
		res := string(uri[pos+2:])
		return scheme + ":/" + res
	}
	// no translation necessary
	return uri
}
