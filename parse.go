package xg

import (
	"errors"
	"fmt"
)

func ParseTokens(buf string, ontoken func(t *Token) error) error {
	tt := tokenizer{buf: buf}
	t := tt.Next()
	for {
		if t.Kind == EOF {
			return nil
		} else if t.Kind == Err {
			return t.Error
		}
		if ontoken != nil {
			err := ontoken(t)
			if err != nil {
				return nil
			}
		}
		t = tt.Next()
	}
}

type AttributeList []*Token

func (aa AttributeList) Attr(name string) (string, bool) {
	for _, a := range aa {
		if string(a.Name) == name {
			return a.Value.Unscrambled(), true
		}
	}
	return "", false
}

type ContentHandler = func(t *Token) error
type TagHandler func(tag *Token, attrs AttributeList, content *Content) error

type Content struct {
	tt       *tokenizer
	t        *Token
	err      error
	finished bool
	locked   bool
}

var ErrNoMoreContent = errors.New("no more content available")

func (ci *Content) Err() error {
	return ci.err
}

func (ci *Content) Next() bool {
	if ci == nil || ci.finished || ci.err != nil {
		return false
	}

	if ci.locked {
		panic("outer content is locked while handling child tags")
	}

	if ci.t != nil && ci.t.Kind == Tag {
		skipTag(ci.tt)
	}

	ci.t = ci.tt.Next()
	switch ci.t.Kind {
	case Err:
		ci.err = ci.t.Error
		ci.t = nil
		return false
	case EndContent:
		ci.t = nil
		ci.finished = true
		return false
	case EOF:
		return false
	case XmlDecl, DocTypeDecl, Tag, SData, CData, Comment, PI:
		return true
	default:
		ci.t = nil
	}
	// we should not end up being here
	panic("unexpected token " + ci.t.Kind.String())
}

func (ci *Content) NextTag() bool {
	for {
		ok := ci.Next()
		if !ok {
			return false
		}
		if ci.IsTag() {
			return true
		}
	}
}

func (ci *Content) Kind() TokenKind {
	if ci.t != nil {
		return ci.t.Kind
	}
	return Err
}
func (ci *Content) Name() NameString {
	if ci == nil || ci.t == nil {
		return ""
	}
	return ci.t.Name
}
func (ci *Content) Value() RawString {
	if ci == nil || ci.t == nil {
		return ""
	}
	return ci.t.Value
}
func (ci *Content) Raw() (whitePrefix, tokenStr string) {
	if ci == nil || ci.t == nil {
		return "", ""
	}
	return ci.t.WhitePrefix, ci.t.Raw
}
func (ci *Content) IsXmlDecl() bool {
	return ci.t != nil && ci.t.Kind == XmlDecl
}
func (ci *Content) IsTag() bool {
	return ci.t != nil && ci.t.Kind == Tag
}
func (ci *Content) IsSData() bool {
	return ci.t != nil && ci.t.Kind == SData
}
func (ci *Content) IsCData() bool {
	return ci.t != nil && ci.t.Kind == CData
}
func (ci *Content) IsComment() bool {
	return ci.t != nil && ci.t.Kind == Comment
}
func (ci *Content) IsPI() bool {
	return ci.t != nil && ci.t.Kind == PI
}
func (ci *Content) MakeError(prefix, msg string) error {
	line, pos := CalcLocation(ci.tt.buf, ci.tt.cur)
	if prefix == "" {
		prefix = "xml parser"
	}
	return fmt.Errorf("%s [%d:%d]: %s", prefix, line+1, pos+1, msg)
}
func (ci *Content) HandleTag(callback func(attrs AttributeList, content *Content) error) {
	if ci == nil || ci.t == nil || ci.t.Kind != Tag {
		return
	}

	if ci.locked {
		panic("outer content is locked while handling child tags")
	}

	if callback == nil {
		ci.err = skipTag(ci.tt)
		ci.t = nil
		return
	}

	// collect attributes
	var t *Token
	attrs := AttributeList{}
	// collect attributes
	for {
		t = ci.tt.Next()
		if t.Kind == Attrib {
			attrs = append(attrs, t)
		} else {
			break
		}
	}
	if t.IsError() {
		ci.t = nil
		ci.err = t.Error
		return
	}

	if t.Kind == CloseEmptyTag {
		ci.t = nil
		ci.err = callback(attrs, nil)
	}

	ci.locked = true // make sure nobody calls ci.Next() while handling our content
	defer func() { ci.locked = false; ci.t = nil }()

	if t.Kind == BeginContent {
		content := &Content{tt: ci.tt}
		err := callback(attrs, content)
		if err != nil {
			ci.err = err
			return
		}
		if content.finished {
			return
		}
		for {
			t = ci.tt.Next()
			switch t.Kind {
			case EndContent, EOF:
				ci.finished = true
				return
			case Err:
				ci.err = t.Error
				return
			case Tag:
				err := skipTag(ci.tt)
				if err != nil {
					ci.err = t.Error
					return
				}
				continue
			case SData, CData, Comment, PI:
				continue
			}
			// we should not end up being here
			panic("unexpected token " + t.Kind.String())
		}
	}

	// we should not end up being here
	panic("unexpected token " + t.Kind.String())
}

// ChildStringContent extracts subnode string content
//
// If subnode is empty, or its content is not a simple string, this function
// returns empty string.
//
// This is useful for parsing <tag>string-content</tag> nodes
//
func (ci *Content) ChildStringContent() RawString {
	if ci == nil || ci.t == nil || ci.t.Kind != Tag {
		return ""
	}
	if ci.locked {
		panic("outer content is locked while handling child tags")
	}

	var t *Token
	for {
		// all attributes are ignored
		t = ci.tt.Next()
		if t.Kind != Attrib {
			break
		}
	}

	if t.Kind == CloseEmptyTag {
		return ""
	}
	if t.Kind == BeginContent {
		t = ci.tt.Next()
		if t.Kind == SData {
			ret := t.Value
			t = ci.tt.Next()
			if t.Kind == EndContent {
				ci.t = nil
				return ret
			}
		}
		// skip the rest of child nodes
		for {
			t = ci.tt.Next()
			switch t.Kind {
			case EndContent, EOF:
				ci.t = nil
				return ""
			case Err:
				ci.err = t.Error
				return ""
			case Tag:
				err := skipTag(ci.tt)
				if err != nil {
					ci.err = t.Error
					return ""
				}
				continue
			case SData, CData, Comment, PI:
				continue
			}
			// we should not end up being here
			panic("unexpected token " + t.Kind.String())
		}
	}

	// we should not end up being here
	panic("unexpected token " + t.Kind.String())
}

func Open(buf string) *Content {
	return &Content{tt: &tokenizer{buf: buf}}
}

func skipTag(tt *tokenizer) error {
	var t *Token
	for {
		t = tt.Next()
		if t.Kind != Attrib {
			break
		}
	}
	if t.Kind == CloseEmptyTag {
		return nil
	}
	if t.Kind == BeginContent {
		for {
			t = tt.Next()
			switch t.Kind {
			case EndContent, EOF:
				return nil
			case Err:
				return t.Error
			case Tag:
				err := skipTag(tt)
				if err != nil {
					return err
				}
				continue
			case SData, CData, Comment, PI:
				continue
			}
			// we should not end up being here
			panic("unexpected token " + t.Kind.String())
		}
	}

	// we should not end up being here
	panic("unexpected token " + t.Kind.String() + "while skipping tag")
}
