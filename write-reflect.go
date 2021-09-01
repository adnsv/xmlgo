package xg

import (
	"encoding"
	"reflect"
	"strconv"
)

var (
	marshalerType     = reflect.TypeOf((*Marshaler)(nil)).Elem()
	textMarshalerType = reflect.TypeOf((*encoding.TextMarshaler)(nil)).Elem()
)

type Marshaler interface {
	MarshalXG(w *Writer) error
}

func toStr(v interface{}) (string, error) {
	return marshalToStr(reflect.ValueOf(v))
}

func marshalToStr(val reflect.Value) (string, error) {
	if !val.IsValid() {
		return "", nil
	}

	for val.Kind() == reflect.Interface || val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return "", nil
		}
		val = val.Elem()
	}

	kind := val.Kind()
	typ := val.Type()

	/*

		if val.CanInterface() && typ.Implements(marshalerType) {
			return w.handleMarshaler(val.Interface().(Marshaler))
		}
		if val.CanAddr() {
			pv := val.Addr()
			if pv.CanInterface() && pv.Type().Implements(marshalerType) {
				return w.handleMarshaler(pv.Interface().(Marshaler))
			}
		}
	*/

	if val.CanInterface() && typ.Implements(textMarshalerType) {
		return marshalTextMarshalerToStr(val.Interface().(encoding.TextMarshaler))
	}
	if val.CanAddr() {
		pv := val.Addr()
		if pv.CanInterface() && pv.Type().Implements(textMarshalerType) {
			return marshalTextMarshalerToStr(pv.Interface().(encoding.TextMarshaler))
		}
	}

	if (kind == reflect.Slice || kind == reflect.Array) && typ.Elem().Kind() != reflect.Uint8 {
		// concatenate
		ss := ""
		for i, n := 0, val.Len(); i < n; i++ {
			s, err := marshalToStr(val.Index(i))
			if err != nil {
				return "", err
			}
			ss += s
		}
		return ss, nil
	}

	if kind == reflect.Struct {
		// not supported yet
		return "", nil
	}

	s, err := marshalSimple(typ, val)
	return s, err
}

func toContent(w *Writer, v interface{}) error {
	return marshalToContent(w, reflect.ValueOf(v))
}

func marshalToContent(w *Writer, val reflect.Value) error {
	if !val.IsValid() {
		return nil
	}

	for val.Kind() == reflect.Interface || val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return nil
		}
		val = val.Elem()
	}

	kind := val.Kind()
	typ := val.Type()

	if val.CanInterface() && typ.Implements(marshalerType) {
		v := val.Interface().(Marshaler)
		return v.MarshalXG(w)
	}
	if val.CanAddr() {
		pv := val.Addr()
		if pv.CanInterface() && pv.Type().Implements(marshalerType) {
			v := pv.Interface().(Marshaler)
			return v.MarshalXG(w)
		}
	}

	if val.CanInterface() && typ.Implements(textMarshalerType) {
		s, err := marshalTextMarshalerToStr(val.Interface().(encoding.TextMarshaler))
		if err != nil {
			return err
		}
		w.scramblestr(s)
		return nil
	}
	if val.CanAddr() {
		pv := val.Addr()
		if pv.CanInterface() && pv.Type().Implements(textMarshalerType) {
			s, err := marshalTextMarshalerToStr(pv.Interface().(encoding.TextMarshaler))
			if err != nil {
				return err
			}
			w.scramblestr(s)
			return nil
		}
	}

	if (kind == reflect.Slice || kind == reflect.Array) && typ.Elem().Kind() != reflect.Uint8 {
		for i, n := 0, val.Len(); i < n; i++ {
			if err := marshalToContent(w, val.Index(i)); err != nil {
				return err
			}
		}
		return nil
	}

	if kind == reflect.Struct {
		// not supported yet
		return nil
	}

	s, err := marshalSimple(typ, val)
	if err != nil {
		return err
	}
	w.scramblestr(s)
	return nil
}

func marshalTextMarshalerToStr(v encoding.TextMarshaler) (string, error) {
	b, e := v.MarshalText()
	return string(b), e
}

func marshalSimple(typ reflect.Type, val reflect.Value) (string, error) {
	switch val.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(val.Int(), 10), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return strconv.FormatUint(val.Uint(), 10), nil
	case reflect.Float32, reflect.Float64:
		return strconv.FormatFloat(val.Float(), 'g', -1, val.Type().Bits()), nil
	case reflect.String:
		return val.String(), nil
	case reflect.Bool:
		return strconv.FormatBool(val.Bool()), nil
	case reflect.Array:
		if typ.Elem().Kind() != reflect.Uint8 {
			break
		}
		// [...]byte
		var bytes []byte
		if val.CanAddr() {
			bytes = val.Slice(0, val.Len()).Bytes()
		} else {
			bytes = make([]byte, val.Len())
			reflect.Copy(reflect.ValueOf(bytes), val)
		}
		return string(bytes), nil
	case reflect.Slice:
		if typ.Elem().Kind() != reflect.Uint8 {
			break
		}
		// []byte
		return string(val.Bytes()), nil
	}
	return "", &UnsupportedTypeError{typ}
}
