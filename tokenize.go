package xg

import (
	"fmt"
	"strings"
)

type ErrCode int

const (
	ErrCodeOk = ErrCode(iota)
	ErrCustom
	ErrCodeUnexpectedEOF
	ErrCodeUnexpectedContent
	ErrCodeUnterminatedQStr
	ErrCodeUnterminatedCDATA
	ErrCodeUnsupportedFeature
	ErrCodeExpectedAttrName
	ErrCodeExpectedEQ
	ErrCodeExpectedQStr
	ErrCodeExpectedOTag
	ErrCodeInvalidSchema
	ErrCodeInvalidXmlDecl
	ErrMismatchingTag
	ErrCodeMissingRoot
	ErrUnterminatedComment
	ErrInvalidComment
	ErrUnterminatedPI
)

var ecstr = map[ErrCode]string{
	ErrCodeOk:                 "no error",
	ErrCustom:                 "custom error",
	ErrCodeUnexpectedEOF:      "unexpected end of file",
	ErrCodeUnexpectedContent:  "unexpected content",
	ErrCodeUnterminatedQStr:   "unterminated string",
	ErrCodeUnterminatedCDATA:  "unterminated cdata",
	ErrCodeUnsupportedFeature: "unsupported feature",
	ErrCodeExpectedAttrName:   "attribute name expected",
	ErrCodeExpectedEQ:         "equals sign expected",
	ErrCodeExpectedQStr:       "string expected",
	ErrCodeExpectedOTag:       "opening tag expected",
	ErrCodeInvalidSchema:      "invalid schema",
	ErrCodeInvalidXmlDecl:     "invalid xml declaration",
	ErrMismatchingTag:         "mismatching tag",
	ErrCodeMissingRoot:        "missing root",
	ErrUnterminatedComment:    "unterminated comment",
	ErrInvalidComment:         "invalid comment",
	ErrUnterminatedPI:         "unterminated processing instruction",
}

func (ec ErrCode) String() string {
	m, ok := ecstr[ec]
	if !ok {
		m = "unknown error"
	}
	return m
}

func (ec ErrCode) Succeeded() bool {
	return ec == ErrCodeOk
}
func (ec ErrCode) Failed() bool {
	return ec != ErrCodeOk
}

type errImpl struct {
	ec     ErrCode
	offset int // offset within the original buffer
	line   int
	pos    int
}

func (e *errImpl) Error() string {
	return fmt.Sprintf("xml parser [%d:%d]: %s", e.line+1, e.pos+1, e.ec)
}

func NewError(ec ErrCode, buf string, offset int) error {
	ret := &errImpl{ec: ec, offset: offset}
	ret.line, ret.pos = CalcLocation(buf, offset)
	return ret
}

type TokenKind int

const (
	sof           = TokenKind(iota) // start of file (buffer)
	EOF                             // end of file (buffer)
	Err                             // error
	XmlDecl                         // <?xml version="1.0" encoding="UTF-8"?>
	DocTypeDecl                     // <!DOCTYPE ... >
	Tag                             // <identifier
	CloseEmptyTag                   // />
	BeginContent                    // >
	EndContent                      // </identifier>
	Attrib                          // identifier="qstring" or identifier='qstring'
	SData                           // content string data
	CData                           // cdata tag content
	Comment                         // <!-- comment -->
	PI                              // <?name value?>
)

func (t TokenKind) String() string {
	switch t {
	case sof:
		return "SOF"
	case EOF:
		return "EOF"
	case XmlDecl:
		return "XmlDecl"
	case DocTypeDecl:
		return "DocTypeDecl"
	case Tag:
		return "Tag"
	case BeginContent:
		return "BeginContent"
	case EndContent:
		return "EndContent"
	case CloseEmptyTag:
		return "CloseEmptyTag"
	case Attrib:
		return "Attrib"
	case SData:
		return "SData"
	case CData:
		return "CData"
	case Comment:
		return "Comment"
	case PI:
		return "PI"
	default:
		return "UNKNOWN_TOKEN"
	}
}

type NameString string

type RawString string

func (rs RawString) Unscrambled() string {
	return unscramble(string(rs))
}

type Token struct {
	Kind        TokenKind
	Error       error
	Name        NameString
	Value       RawString
	WhitePrefix string
	Raw         string
	SrcPos      int
}

func (t *Token) IsError() bool {
	return t.Kind == Err
}

func (t *Token) IsDone() bool {
	return t.Kind == EOF
}

type state int

const (
	stateStart = state(iota)
	stateProlog
	stateAttribs
	stateContent
	stateEpilog
)

type tokenizer struct {
	buf   string
	cur   int
	state state
	stack []NameString
}

func (tt *tokenizer) Next() *Token {
	whiteStart := tt.cur
	if tt.state != stateContent {
		tt.skipWhite()
	}
	rawStart := tt.cur
	atEOF := tt.cur == len(tt.buf)

	mkerr := func(ec ErrCode) *Token {
		if ec == ErrCodeUnexpectedContent && atEOF {
			ec = ErrCodeUnexpectedEOF
		}
		return &Token{
			Kind:        Err,
			Error:       NewError(ec, tt.buf, rawStart),
			Name:        "",
			Value:       "",
			WhitePrefix: tt.buf[whiteStart:rawStart],
			Raw:         tt.buf[rawStart:tt.cur],
			SrcPos:      rawStart,
		}
	}

	mktoken := func(k TokenKind, n NameString, v RawString) *Token {
		return &Token{
			Kind:        k,
			Error:       nil,
			Name:        n,
			Value:       v,
			WhitePrefix: tt.buf[whiteStart:rawStart],
			Raw:         tt.buf[rawStart:tt.cur],
			SrcPos:      rawStart,
		}
	}

	if atEOF {
		if tt.state == stateEpilog {
			return mktoken(EOF, "", "")
		}
		return mkerr(ErrCodeUnexpectedEOF)
	}

	if tt.state == stateAttribs {
		if tt.skipStr("/>") {
			if len(tt.stack) > 0 {
				tt.stack = tt.stack[:len(tt.stack)-1]
			}
			if len(tt.stack) == 0 {
				tt.state = stateEpilog
			} else {
				tt.state = stateContent
			}
			return mktoken(CloseEmptyTag, "", "")
		}
		if tt.skipByte('>') {
			tt.state = stateContent
			return mktoken(BeginContent, "", "")
		}
		n, v, ec := tt.readAttrPair()
		if ec != ErrCodeOk {
			return mkerr(ec)
		}
		return mktoken(Attrib, n, v)
	}

	if tt.state == stateStart {
		// bom
		if tt.skipStr("\xef\xbb\xbf") {
			tt.skipWhite()
		}
		if tt.skipStr("<?xml") {
			// xmlspec:XMLDecl
			tt.skipWhite()
			n, v, ec := tt.readAttrPair()
			if ec == ErrCodeOk && n == "version" {
				// xmlspec:VersionInfo
				// ignore actual version number, should be '1.0' or '1.1'
				tt.skipWhite()
				n, v, ec = tt.readAttrPair()
			}
			if ec != ErrCodeOk {
				return mkerr(ec)
			}
			if n != "encoding" {
				return mkerr(ErrCodeInvalidXmlDecl)
			}
			tt.skipWhite()
			if !tt.skipStr("?>") {
				return mkerr(ErrCodeInvalidXmlDecl)
			}
			tt.state = stateProlog
			return mktoken(XmlDecl, "", v)
		}
		tt.state = stateProlog
	}

	if tt.skipStr("<!--") {
		s, ec := tt.readComment()
		if ec != ErrCodeOk {
			return mkerr(ec)
		}
		return mktoken(Comment, "", RawString(s))
	}

	if tt.skipStr("<?") {
		// processing instruction
		n, c, ec := tt.readPI()
		if ec != ErrCodeOk {
			return mkerr(ec)
		}
		return mktoken(PI, n, RawString(c))
	}

	if tt.state == stateProlog {
		if !tt.skipByte('<') {
			return mkerr(ErrCodeUnexpectedContent)
		}
		if tt.skipStr("!DOCTYPE") {
			if tt.state != stateProlog {
				return mkerr(ErrCodeUnexpectedContent)
			}
			// todo: implement doctype parsing or skipping
			n, s, ec := tt.readDocTypeDecl()
			if ec != ErrCodeOk {
				return mkerr(ec)
			}
			return mktoken(DocTypeDecl, n, RawString(s))
		}
		// open-tag token
		n := tt.readName()
		if len(n) == 0 {
			return mkerr(ErrCodeUnexpectedContent)
		}
		tt.stack = append(tt.stack, n)
		tt.state = stateAttribs
		return mktoken(Tag, n, "")
	}

	if tt.state == stateEpilog {
		return mkerr(ErrCodeUnexpectedContent)
	}

	if tt.state != stateContent {
		panic("internal parser error: unexpected state")
	}
	n := strings.IndexByte(tt.buf[tt.cur:], '<')
	if n < 0 {
		return mkerr(ErrCodeUnexpectedEOF)
	}
	if n > 0 {
		o := tt.cur
		tt.cur += n
		return mktoken(SData, "", RawString(tt.buf[o:tt.cur]))
	}

	tt.cur++ // skip over the '<'
	if tt.skipStr("![CDATA[") {
		n := strings.Index(tt.buf[tt.cur:], "]]>")
		if n < 0 {
			return mkerr(ErrCodeUnterminatedCDATA)
		}
		o := tt.cur
		tt.cur += n + 3
		return mktoken(CData, "", RawString(tt.buf[o:tt.cur-3]))
	}
	if tt.skipByte('/') {
		// closing tag
		cname := tt.readName()
		if len(cname) == 0 {
			return mkerr(ErrCodeUnexpectedContent)
		}
		if len(tt.stack) == 0 {
			return mkerr(ErrMismatchingTag)
		}
		oname := tt.stack[len(tt.stack)-1]
		tt.stack = tt.stack[:len(tt.stack)-1]
		if oname != cname {
			return mkerr(ErrMismatchingTag)
		}
		tt.skipWhite()
		if !tt.skipByte('>') {
			return mkerr(ErrCodeUnexpectedContent)
		}
		if len(tt.stack) == 0 {
			tt.state = stateEpilog
		} else {
			tt.state = stateContent
		}
		return mktoken(EndContent, cname, "")
	}
	oname := tt.readName()
	if len(oname) == 0 {
		return mkerr(ErrCodeUnexpectedContent)
	}
	tt.stack = append(tt.stack, oname)
	tt.state = stateAttribs
	return mktoken(Tag, oname, "")
}

func (tt *tokenizer) readName() NameString {
	o := tt.cur
	if isNameStart(tt.buf[tt.cur]) {
		for tt.cur++; tt.cur < len(tt.buf) && isNameChar(tt.buf[tt.cur]); tt.cur++ {
		}
	}
	return NameString(tt.buf[o:tt.cur])
}

func (tt *tokenizer) readComment() (string, ErrCode) {
	// comment
	o := tt.cur
	n := strings.Index(tt.buf[tt.cur:], "--")
	if n < 0 {
		return "", ErrUnterminatedComment
	}
	tt.cur += n + 2
	if !tt.skipByte('>') {
		return "", ErrInvalidComment
	}
	return tt.buf[o : tt.cur-3], ErrCodeOk
}

func (tt *tokenizer) readPI() (name NameString, content string, ec ErrCode) {
	name = tt.readName()
	if len(name) == 0 {
		ec = ErrCodeUnexpectedContent
		return
	}
	tt.skipWhite()
	o := tt.cur
	n := strings.Index(tt.buf[tt.cur:], "?>")
	if n < 0 {
		ec = ErrUnterminatedPI
		return
	}
	tt.cur += n + 2
	content = tt.buf[o : tt.cur-2]
	return
}

func (tt *tokenizer) readQStr() (s string, ec ErrCode) {
	if tt.cur >= len(tt.buf) {
		ec = ErrCodeExpectedQStr
		return
	}
	quote := tt.buf[tt.cur]
	if quote != '\'' && quote != '"' {
		ec = ErrCodeExpectedQStr
		return
	}
	n := strings.IndexByte(tt.buf[tt.cur+1:], quote)

	if n < 0 {
		ec = ErrCodeUnterminatedQStr
		return
	}

	s = tt.buf[tt.cur+1 : tt.cur+1+n]
	tt.cur += n + 2
	return
}

func (tt *tokenizer) readAttrPair() (name NameString, value RawString, ec ErrCode) {
	name = tt.readName()
	if len(name) == 0 {
		ec = ErrCodeExpectedAttrName
		return
	}
	tt.skipWhite()
	if !tt.skipByte('=') {
		ec = ErrCodeExpectedEQ
		return
	}
	tt.skipWhite()
	if tt.cur >= len(tt.buf) {
		ec = ErrCodeExpectedQStr
		return
	}
	var s string
	s, ec = tt.readQStr()
	value = RawString(s)
	return
}

func (tt *tokenizer) readDocTypeDecl() (name NameString, content string, ec ErrCode) {
	if !tt.skipWhite() {
		ec = ErrCodeUnexpectedContent
		return
	}
	name = tt.readName()
	if len(name) == 0 {
		ec = ErrCodeUnexpectedContent
		return
	}
	end := len(tt.buf)

	if tt.skipWhite() {
		o := tt.cur
		quote := uint8(0)
		depth := 0 // we already had an opening '<' inside "<!DOCTYPE" prefix
		// skip over the entire thing counting opening and closing brackets,
		// ignore those inside comments and quoted strings
		for {
			if tt.cur >= end {
				ec = ErrCodeUnexpectedEOF
				return
			}
			b := tt.buf[tt.cur]
			tt.cur++

			if quote == 0 && b == '>' && depth == 0 {
				content = tt.buf[o : tt.cur-1]
				return
			}

			switch {
			case b == quote:
				quote = 0
			case quote != 0: // keep going
			case b == '\'' || b == '"':
				quote = b
			case b == '>' && quote == 0:
				depth--
			case b == '<' && quote == 0:
				if tt.skipStr("!--") {
					_, ec = tt.readComment()
					if ec != ErrCodeOk {
						return
					}
				} else {
					depth++
				}
			}
		}
	} else if tt.skipByte('>') {
		return
	}

	ec = ErrCodeUnexpectedContent
	return
}

func (tt *tokenizer) skipWhite() bool {
	o := tt.cur
	for ; tt.cur < len(tt.buf) && isWhite(tt.buf[tt.cur]); tt.cur++ {
	}
	return tt.cur > o
}

func (tt *tokenizer) skipByte(c byte) bool {
	if tt.cur < len(tt.buf) && tt.buf[tt.cur] == c {
		tt.cur++
		return true
	}
	return false
}

func (tt *tokenizer) skipStr(c string) bool {
	if strings.HasPrefix(tt.buf[tt.cur:], c) {
		tt.cur += len(c)
		return true
	}
	return false
}

func isAsciiAlpha(cp byte) bool {
	return ('A' <= cp && cp <= 'Z') || ('a' <= cp && cp <= 'z')
}

func isDecDigit(cp byte) bool {
	return '0' <= cp && cp <= '9'
}

func isNameStart(cp byte) bool {
	return isAsciiAlpha(cp) || cp == ':' || cp == '_' || cp >= 128
}

func isNameChar(cp byte) bool {
	return isNameStart(cp) || isDecDigit(cp) || cp == '-' || cp == '.'
}

func isWhite(cp byte) bool {
	return cp <= ' ' && (cp == ' ' || cp == '\t' || cp == '\r' ||
		cp == '\n' || cp == '\v' || cp == '\f')
}
