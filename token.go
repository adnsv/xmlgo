package xg

import "strings"

type ErrCode int

const (
	ErrCodeOk = ErrCode(iota)
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
)

var ecstr = map[ErrCode]string{
	ErrCodeOk:                 "no error",
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

type TokenKind int

const (
	startOfFile = TokenKind(iota)
	Done
	Err
	XmlDecl
	OpenTag       // <identifier
	ChildrenToken // >
	CloseEmptyTag // />
	CloseTag      // </identifier>
	Attrib        // identifier="qstring" or identifier='qstring'
	SData         // content string data
	CData         // cdata tag content
	Comment       // <!-- comment -->
)

type NameString string

type RawString string

func (rs RawString) Unscrambled() string {
	return unscramble(string(rs))
}

type Token struct {
	Kind        TokenKind
	EC          ErrCode
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
	return t.Kind == Done
}

type tokenizerState int

const (
	tsStart = tokenizerState(iota)
	tsProlog
	tsAttribs
	tsContent
	tsEpilog
)

type tokenizer struct {
	buf   string
	cur   int
	state tokenizerState
	stack []NameString
}

func (tt *tokenizer) Next() *Token {
	tokenWhiteStart := tt.cur
	if tt.state != tsContent {
		tt.skipWhite()
	}
	tokenSrcPos := tt.cur
	tokenAtEOF := tt.cur == len(tt.buf)

	mkerr := func(ec ErrCode) *Token {
		if ec == ErrCodeUnexpectedContent && tokenAtEOF {
			ec = ErrCodeUnexpectedEOF
		}
		return &Token{
			Kind:        Err,
			Name:        "",
			Value:       "",
			WhitePrefix: tt.buf[tokenWhiteStart:tokenSrcPos],
			Raw:         tt.buf[tokenSrcPos:tt.cur],
			SrcPos:      tokenSrcPos,
		}
	}

	mktoken := func(k TokenKind, n NameString, v RawString) *Token {
		return &Token{
			Kind:        k,
			Name:        n,
			Value:       v,
			WhitePrefix: tt.buf[tokenWhiteStart:tokenSrcPos],
			Raw:         tt.buf[tokenSrcPos:tt.cur],
			SrcPos:      tokenSrcPos,
		}
	}

	if tokenAtEOF {
		if tt.state == tsEpilog {
			return mktoken(Done, "", "")
		}
		return mkerr(ErrCodeUnexpectedEOF)
	}

	readAttrPair := func() (name NameString, value RawString, ec ErrCode) {
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
		term := tt.buf[tt.cur]
		if term != '\'' && term != '"' {
			ec = ErrCodeExpectedQStr
			return
		}
		n := strings.IndexByte(tt.buf[tt.cur+1:], term)
		er := strings.IndexAny(tt.buf[tt.cur+1:], "\r")
		en := strings.IndexAny(tt.buf[tt.cur+1:], "\n")

		if n < 0 || (er > 0 && n > er) || (en > 0 && n > en) {
			ec = ErrCodeUnterminatedQStr
			return
		}
		value = RawString(tt.buf[tt.cur+1 : tt.cur+1+n])
		tt.cur += n + 2
		return
	}

	if tt.state == tsAttribs {
		if tt.skipStr("/>") {
			if len(tt.stack) > 0 {
				tt.stack = tt.stack[:len(tt.stack)-1]
			}
			if len(tt.stack) == 0 {
				tt.state = tsEpilog
			} else {
				tt.state = tsContent
			}
			return mktoken(CloseEmptyTag, "", "")
		}
		if tt.skipByte('>') {
			tt.state = tsContent
			return mktoken(ChildrenToken, "", "")
		}
		n, v, e := readAttrPair()
		if e != ErrCodeOk {
			return mkerr(e)
		}
		return mktoken(Attrib, n, v)
	}

	handleComment := func() (iscomment bool, tk *Token, ec ErrCode) {
		if tt.cur+7 > len(tt.buf) {
			return
		}
		if !tt.skipStr("<!--") {
			return
		}
		o := tt.cur
		iscomment = true
		n := strings.Index(tt.buf[tt.cur:], "--")
		if n < 0 {
			ec = ErrUnterminatedComment
			return
		}
		tt.cur += n + 2
		if !tt.skipByte('>') {
			ec = ErrInvalidComment
			return
		}
		tk = mktoken(Comment, "", RawString(tt.buf[o:tt.cur-3]))
		return
	}

	iscomment, tk, ec := handleComment()
	if iscomment {
		if tk == nil {
			return mkerr(ec)
		}
		return tk
	}

	if tt.state == tsStart {
		// bom
		if tt.skipStr("\xef\xbb\xbf") {
			tt.skipWhite()
		}
		if tt.skipStr("<?xml") {
			// xmlspec:XMLDecl
			tt.skipWhite()
			n, v, e := readAttrPair()
			if e == ErrCodeOk && n == "version" {
				// xmlspec:VersionInfo
				// ignore actual version number, should be '1.0' or '1.1'
				tt.skipWhite()
				n, v, e = readAttrPair()
			}
			if e != ErrCodeOk {
				return mkerr(e)
			}
			if n != "encoding" {
				return mkerr(ErrCodeInvalidXmlDecl)
			}
			tt.skipWhite()
			if !tt.skipStr("?>") {
				return mkerr(ErrCodeInvalidXmlDecl)
			}
			tt.state = tsProlog
			return mktoken(XmlDecl, "", v)
		}
		tt.state = tsProlog
	}

	if tt.state == tsProlog {
		if !tt.skipByte('<') {
			return mkerr(ErrCodeUnexpectedContent)
		}
		if tt.skipByte('?') {
			// todo: implement PI parsing or skipping
			return mkerr(ErrCodeUnsupportedFeature)
		}
		if tt.skipStr("!DOCTYPE") {
			// todo: implement doctype parsing or skipping
			return mkerr(ErrCodeUnsupportedFeature)
		}
		// open-tag token
		n := tt.readName()
		if len(n) == 0 {
			return mkerr(ErrCodeUnexpectedContent)
		}
		tt.stack = append(tt.stack, n)
		tt.state = tsAttribs
		return mktoken(OpenTag, n, "")
	}

	if tt.state == tsEpilog {
		if tt.skipStr("<?") {
			return mkerr(ErrCodeUnsupportedFeature)
		}
		return mkerr(ErrCodeUnexpectedContent)
	}

	if tt.state != tsContent {
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
	if tt.skipByte('?') {
		return mkerr(ErrCodeUnsupportedFeature)
	}
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
			tt.state = tsEpilog
		} else {
			tt.state = tsContent
		}
		return mktoken(CloseTag, cname, "")
	}
	oname := tt.readName()
	if len(oname) == 0 {
		return mkerr(ErrCodeUnexpectedContent)
	}
	tt.stack = append(tt.stack, oname)
	tt.state = tsAttribs
	return mktoken(OpenTag, oname, "")
}

func (tt *tokenizer) readName() NameString {
	o := tt.cur
	if isNameStart(tt.buf[tt.cur]) {
		for tt.cur++; tt.cur < len(tt.buf) && isNameChar(tt.buf[tt.cur]); tt.cur++ {
		}
	}
	return NameString(tt.buf[o:tt.cur])
}

func (tt *tokenizer) skipWhite() {
	for ; tt.cur < len(tt.buf) && isWhite(tt.buf[tt.cur]); tt.cur++ {
	}
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
