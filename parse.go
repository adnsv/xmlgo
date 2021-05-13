package xg

import "errors"

func ParseTokens(buf string, ontoken func(t *Token) error) error {
	tt := tokenizer{buf: buf}
	t := tt.Next()
	for {
		if t.Kind == Done {
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

type ContentHandler = func(t *Token) error
type TagHandler func(tag *Token, attrs AttributeList, content *ContentIterator) error

type ContentIterator struct {
	tt       *tokenizer
	t        *Token
	err      error
	finished bool
	locked   bool
}

var ErrNoMoreContent = errors.New("no more content available")

func (ci *ContentIterator) Err() error {
	return ci.err
}

func (ci *ContentIterator) Next() bool {
	if ci == nil || ci.finished || ci.err != nil {
		return false
	}

	if ci.locked {
		panic("outer content is locked while handling child tags")
	}

	if ci.t != nil && ci.t.Kind == OpenTag {
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
	case Done:
		return false
	case XmlDecl, OpenTag, SData, CData, Comment, PI:
		return true
	default:
		ci.t = nil
	}
	// we should not end up being here
	panic("unexpected token " + ci.t.Kind.String())
}

func (ci *ContentIterator) NextTag() bool {
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

func (ci *ContentIterator) Kind() TokenKind {
	if ci.t != nil {
		return ci.t.Kind
	}
	return Err
}
func (ci *ContentIterator) Name() NameString {
	if ci == nil || ci.t == nil {
		return ""
	}
	return ci.t.Name
}
func (ci *ContentIterator) Value() RawString {
	if ci == nil || ci.t == nil {
		return ""
	}
	return ci.t.Value
}
func (ci *ContentIterator) Raw() (whitePrefix, tokenStr string) {
	if ci == nil || ci.t == nil {
		return "", ""
	}
	return ci.t.WhitePrefix, ci.t.Raw
}
func (ci *ContentIterator) IsXmlDecl() bool {
	return ci.t != nil && ci.t.Kind == XmlDecl
}
func (ci *ContentIterator) IsTag() bool {
	return ci.t != nil && ci.t.Kind == OpenTag
}
func (ci *ContentIterator) IsSData() bool {
	return ci.t != nil && ci.t.Kind == SData
}
func (ci *ContentIterator) IsCData() bool {
	return ci.t != nil && ci.t.Kind == CData
}
func (ci *ContentIterator) IsComment() bool {
	return ci.t != nil && ci.t.Kind == Comment
}
func (ci *ContentIterator) IsPI() bool {
	return ci.t != nil && ci.t.Kind == PI
}
func (ci *ContentIterator) HandleTag(callback func(attrs AttributeList, content *ContentIterator) error) {
	if ci == nil || ci.t == nil || ci.t.Kind != OpenTag {
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

	if t.Kind == StartContent {
		content := &ContentIterator{tt: ci.tt}
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
			case EndContent, Done:
				ci.finished = true
				return
			case Err:
				ci.err = t.Error
				return
			case OpenTag:
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
func (ci *ContentIterator) ChildStringContent() RawString {
	if ci == nil || ci.t == nil || ci.t.Kind != OpenTag {
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
	if t.Kind == StartContent {
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
			case EndContent, Done:
				ci.t = nil
				return ""
			case Err:
				ci.err = t.Error
				return ""
			case OpenTag:
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

func Open(buf string) *ContentIterator {
	return &ContentIterator{tt: &tokenizer{buf: buf}}
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
	if t.Kind == StartContent {
		for {
			t = tt.Next()
			switch t.Kind {
			case EndContent, Done:
				return nil
			case Err:
				return t.Error
			case OpenTag:
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
