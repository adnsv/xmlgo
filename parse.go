package xg

type FlatHandler func(t *Token) error
type TokenHandler func(t *Token) (TokenHandler, error)

/*func mkerr(ec ErrCode) error {
	return fmt.Errorf("xml syntax error: %s", ec)
}*/

func ParseFlat(buf string, h FlatHandler) error {
	tt := tokenizer{buf: buf}
	t := tt.Next()
	for {
		if t.Kind == Done {
			return nil
		} else if t.Kind == Err {
			return t.Error
		}
		if h != nil {
			err := h(t)
			if err != nil {
				return nil
			}
		}
		t = tt.Next()
	}
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
