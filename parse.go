package xg

import "fmt"

type TagHandler struct {
	OnAttr         func(n NameString, v RawString, raw string) error
	OnStartContent func(raw string) error
	OnChildSD      func(v RawString, raw string) error
	OnChildCD      func(v RawString, raw string) error
	OnChildTag     TagHandlerProc
	OnClose        func(empty bool, raw string) error
}

type TagHandlerProc func(n NameString, raw string) (TagHandler, error)

func mkerr(ec ErrCode) error {
	return fmt.Errorf("xml syntax error: %s", ec)
}

func Parse(buf string, onroot TagHandlerProc) error {
	tt := &tokenizer{buf: buf}
	tok := tt.Next()
	for {
		if tok.Kind == OTagToken {
			break
		}
		if tok.IsError() {
			return mkerr(tok.EC)
		}
		if tok.IsDone() {
			return mkerr(ErrCodeMissingRoot)
		}
		tok = tt.Next()
	}
	return handleTag(tt, tok, onroot)
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
	for t.Kind == AttribToken {
		if h.OnAttr != nil {
			err := h.OnAttr(t.Name, t.Str, t.Raw)
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
			case SDataToken:
				if h.OnChildSD != nil {
					h.OnChildSD(t.Str, t.Raw)
				}
			case CDataToken:
				if h.OnChildCD != nil {
					h.OnChildCD(t.Str, t.Raw)
				}
			case OTagToken:
				err = handleTag(tt, t, h.OnChildTag)
				if err != nil {
					return err
				}
			case CTagToken:
				if h.OnClose != nil {
					err := h.OnClose(false, t.Raw)
					if err != nil {
						return err
					}
				}
				return nil
			case ErrToken:
				return mkerr(t.EC)
			case DoneToken:
				return mkerr(ErrCodeUnexpectedEOF)
			default:
				return mkerr(ErrCodeUnexpectedContent)
			}
			t = tt.Next()
		}

	} else if t.Kind == ETagToken {
		// done with this node
		if h.OnClose != nil {
			h.OnClose(true, t.Raw)
		}
		return nil
	} else {
		return mkerr(ErrCodeUnexpectedContent)
	}
}
