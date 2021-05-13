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
}

var ErrNoMoreContent = errors.New("no more content available")

func (ci *ContentIterator) Err() error {
	return ci.err
}

func (ci *ContentIterator) Next() bool {
	if ci == nil || ci.finished || ci.err != nil {
		return false
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
	case OpenTag, SData, CData, Comment, PI:
		return true
	default:
		ci.t = nil
	}
	// we should not end up being here
	panic("unexpected token")
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
func (ci *ContentIterator) IsSubtag() bool {
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

func Parse(buf string, onroot TagHandler, oncontent ContentHandler) error {
	tt := tokenizer{buf: buf}
	for {
		t := tt.Next()
		if t.Kind == Done {
			return nil
		} else if t.Kind == Err {
			return t.Error
		} else if t.Kind == OpenTag {
			err := processTag(&tt, t, onroot)
			if err != nil {
				return err
			}
		} else {
			if oncontent != nil {
				err := oncontent(t)
				if err != nil {
					return err
				}
			}
		}
	}
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
			case EndContent:
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
			panic("unexpected token")
		}
	}

	// we should not end up being here
	panic("unexpected token")
}

func processTag(tt *tokenizer, t *Token, ontag TagHandler) error {
	if ontag == nil {
		// skip over
		return skipTag(tt)
	}

	starttoken := t
	attrs := AttributeList{}
	// collect attributes
	for {
		t = tt.Next()
		if t.Kind == Attrib {
			attrs = append(attrs, t)
		} else {
			break
		}
	}
	if t.IsError() {
		return t.Error
	}
	if t.Kind == CloseEmptyTag {
		return ontag(starttoken, attrs, nil)
	}
	if t.Kind == StartContent {
		ci := &ContentIterator{tt: tt}
		err := ontag(starttoken, attrs, ci)
		if err != nil {
			return err
		}
		if ci.finished {
			return nil
		}
		for {
			t = tt.Next()
			switch t.Kind {
			case EndContent:
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
			panic("unexpected token")
		}
	}

	// we should not end up being here
	panic("unexpected token")
}

/*

func Parse(buf string, h TokenHandler) (err error) {
	tt := &tokenizer{buf: buf}
	for {
		tok := tt.Next()
		var th TokenHandler
		if h != nil {
			th, err = h(tok)
			if err != nil {
				return err
			}
		}
		if tok.Kind == OpenTag {

			break
		}
		if tok.IsError() {
			return mkerr(tok.EC)
		}
		if tok.IsDone() {
			return mkerr(ErrCodeMissingRoot)
		}
	}
}

func handleTag(tt *tokenizer, t *Token, ontag TagHandlerProc) error {
	var err error
	h := TagHandler{}
	if ontag != nil {
		h, err = ontag(t.Name, t.Raw)
		if err != nil {
			return err
		}
	}

	t = tt.Next()
	for t.Kind == Attrib {
		if h.OnAttr != nil {
			err := h.OnAttr(t.Name, t.Value, t.Raw)
			if err != nil {
				return err
			}
		}
		t = tt.Next()
	}
	if t.Kind == ChildrenToken {
		// process child content
		if h.OnStartContent != nil {
			err := h.OnStartContent(t.Raw)
			if err != nil {
				return err
			}
		}
		t = tt.Next()
		for {
			switch t.Kind {
			case SData:
				if h.OnChildSD != nil {
					h.OnChildSD(t.Value, t.Raw)
				}
			case CData:
				if h.OnChildCD != nil {
					h.OnChildCD(t.Value, t.Raw)
				}
			case OpenTag:
				err = handleTag(tt, t, h.OnChildTag)
				if err != nil {
					return err
				}
			case CloseTag:
				if h.OnClose != nil {
					err := h.OnClose(false, t.Raw)
					if err != nil {
						return err
					}
				}
				return nil
			case Err:
				return mkerr(t.EC)
			case Done:
				return mkerr(ErrCodeUnexpectedEOF)
			default:
				return mkerr(ErrCodeUnexpectedContent)
			}
			t = tt.Next()
		}

	} else if t.Kind == CloseEmptyTag {
		// done with this node
		if h.OnClose != nil {
			h.OnClose(true, t.Raw)
		}
		return nil
	} else {
		return mkerr(ErrCodeUnexpectedContent)
	}
}
*/
