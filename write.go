package xg

import (
	"io"
	"reflect"
	"strings"
)

type Writer struct {
	out           io.Writer
	names         []string
	inOtag        bool
	indentLevel   int
	prevLineLevel int
	indentSpaces  int // when 0 (default), indent with tabs
}

func NewWriter(out io.Writer) *Writer {
	return &Writer{out: out}
}

func (w *Writer) put(s string) {
	w.out.Write([]byte(s))
}

func (w *Writer) BeginContent() {
	if w.inOtag {
		w.put(">")
		w.inOtag = false
	}
}

func (w *Writer) putIndent(level int) {
	if w.indentSpaces > 0 {
		writeSpaces(w.out, level*w.indentSpaces)
	} else {
		writeTabs(w.out, level)
	}
}

func (w *Writer) BOM() {
	w.put("\xef\xbb\xbf")
}

func (w *Writer) XmlDecl() {
	if len(w.names) > 0 {
		panic("xml writer: invalid XmlDecl placement")
	}
	w.put(`<?xml version="1.0" encoding="UTF-8"?>\n`)
}

const nolevel = -1

func (w *Writer) OTag(name string) {
	if len(name) == 0 || name == "+" {
		panic("xml writer: trying to write a tag with empty name")
	}
	prevLevel := w.prevLineLevel
	w.BeginContent()
	indent := name[0] == '+'
	w.names = append(w.names, name)
	if indent {
		name = name[1:]
		w.indentLevel++
		if prevLevel == nolevel || prevLevel > w.indentLevel {
			w.put("\n")
			w.putIndent(w.indentLevel)
		} else {
			w.putIndent(w.indentLevel - prevLevel)
		}
	}
	w.put("<")
	w.put(name)
	w.inOtag = true
	w.prevLineLevel = nolevel
}

func (w *Writer) scramblestr(s string) {
	encodeEOLs := false
	if w.inOtag {
		encodeEOLs = true
	}
	i, o, n := 0, 0, len(s)
	if n <= 0 {
		return
	}
	for i < n {
		c := s[i]
		i++
		switch c {
		case '&':
			w.put(s[o : i-1])
			w.put("&amp;")
			o = i
		case '<':
			w.put(s[o : i-1])
			w.put("&lt;")
			o = i
		case '>':
			w.put(s[o : i-1])
			w.put("&gt;")
			o = i
		case '\'':
			w.put(s[o : i-1])
			w.put("&apos;")
			o = i
		case '"':
			w.put(s[o : i-1])
			w.put("&quot;")
			o = i
		default:
			if c < ' ' {
				w.put(s[o : i-1])
				if !encodeEOLs && (c == '\r' || c == '\n') {
					w.put("\n")
					if c == '\r' && i < n && s[i] == 'n' {
						i++
					}
				} else {
					var buf [5]byte
					buf[0] = '&'
					buf[1] = '#'
					buf[2] = uint8('0') + c/10
					buf[3] = uint8('0') + c%10
					buf[4] = ';'
					w.out.Write(buf[:])
				}
				o = i
			}
		}
	}
	w.put(s[o:n])
}

func (w *Writer) String(s string) {
	w.BeginContent()
	w.scramblestr(s)
}

func (w *Writer) Write(v interface{}) {
	w.BeginContent()
	toContent(w, v)
}

func (w *Writer) Comment(s string) {
	w.BeginContent()
	w.put("<!--")
	w.put(strings.ReplaceAll(s, "--", "-")) // make sure double-dash is not written
	w.put("-->")
}

func (w *Writer) StringAttr(name string, value string) {
	if !w.inOtag {
		panic("xml writer: trying to write an attribute outside of an open tag")
	}
	w.put(" ")
	w.put(name)
	w.put(`="`)
	w.scramblestr(value)
	w.put(`"`)
}

func (w *Writer) Attr(name string, value interface{}) {
	if !w.inOtag {
		panic("xml writer: trying to write an attribute outside of an open tag")
	}
	s, _ := toStr(value)
	w.put(" ")
	w.put(name)
	w.put(`="`)
	w.scramblestr(s)
	w.put(`"`)
}

func (w *Writer) OptStringAttr(name string, value string) {
	if !w.inOtag {
		panic("xml writer: trying to write an attribute outside of an open tag")
	}
	if len(value) == 0 {
		return
	}
	w.put(" ")
	w.put(name)
	w.put(`="`)
	w.scramblestr(value)
	w.put(`"`)
}

func (w *Writer) OptAttr(name string, value interface{}) {
	if !w.inOtag {
		panic("xml writer: trying to write an attribute outside of an open tag")
	}
	s, _ := toStr(value)
	if len(s) == 0 {
		return
	}
	w.put(" ")
	w.put(name)
	w.put(`="`)
	w.scramblestr(s)
	w.put(`"`)
}

func (w *Writer) CTag() {
	if len(w.names) == 0 {
		panic("xml writer: tag stack underflow")
	}
	name := w.names[len(w.names)-1]
	w.names = w.names[:len(w.names)-1]

	indented := name[0] == '+'
	if indented {
		name = name[1:]
	}
	if w.inOtag {
		w.inOtag = false
		w.put(" />")
	} else {
		w.put("</")
		w.put(name)
		w.put(">")
	}
	if indented {
		w.indentLevel--
		w.put("\n")
		w.putIndent(w.indentLevel)
		w.prevLineLevel = w.indentLevel
	}
}

var tabs = [8]byte{'\t', '\t', '\t', '\t', '\t', '\t', '\t', '\t'}

const spaces = "                                "

func writeTabs(w io.Writer, n int) (err error) {
	bb := tabs[:]
	for n > 8 {
		_, err = w.Write(bb)
		if err != nil {
			return
		}
		n -= 8
	}
	if n > 0 {
		_, err = w.Write(bb[:n])
	}
	return
}

func writeSpaces(w io.Writer, n int) (err error) {
	ns := len(spaces)
	bb := []byte(spaces)
	for n > ns {
		_, err = w.Write(bb)
		if err != nil {
			return
		}
		n -= ns
	}
	if n > 0 {
		_, err = w.Write(bb[:n])
	}
	return
}

// UnsupportedTypeError is returned when Marshal encounters a type
// that cannot be converted into XML.
type UnsupportedTypeError struct {
	Type reflect.Type
}

func (e *UnsupportedTypeError) Error() string {
	return "xml: unsupported type: " + e.Type.String()
}
