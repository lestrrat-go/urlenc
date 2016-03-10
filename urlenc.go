package urlenc

// Package urlenc provides a standard-lib type Marshal/Unmarshal interface
// to structs that can encode/decode themselves to URL query strings.

import (
	"errors"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"sync"
)

const (
	stringType = iota + 1
	numberType
	stringSliceType
	numberSliceType
)

type field struct {
	ArrayElementKind reflect.Kind
	ArrayElementType reflect.Type
	Index            int
	Kind             reflect.Kind
	OmitEmpty        bool
	Name             string
	Type             int
}

var t2f = type2fields{
	types: make(map[reflect.Type][]field),
}

type type2fields struct {
	lock  sync.RWMutex
	types map[reflect.Type][]field
}

func isSupportedType(rt reflect.Type, recurse bool) (int, bool) {
	switch rt.Kind() {
	case reflect.String:
		return stringType, true
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Float32, reflect.Float64:
		return numberType, true
	case reflect.Slice:
		if recurse {
			et, ok := isSupportedType(rt.Elem(), false)
			if !ok {
				return 0, false
			}
			switch et {
			case stringType:
				return stringSliceType, true
			case numberType:
				return numberSliceType, true
			}
		}
	}
	return 0, false
}

func convertToString(rv reflect.Value) (string, error) {
	switch rv.Kind() {
	case reflect.String:
		return rv.String(), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(rv.Int(), 10), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return strconv.FormatUint(rv.Uint(), 10), nil
	case reflect.Float32, reflect.Float64:
		return strconv.FormatFloat(rv.Float(), 'f', -1, 64), nil
	}

	return "", errors.New("unsupported type")
}

func convertFromString(k reflect.Kind, v string) (reflect.Value, error) {
	switch k {
	case reflect.String:
		return reflect.ValueOf(v), nil
	case reflect.Int:
		nv, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return zeroval, err
		}
		return reflect.ValueOf(int(nv)), nil
	case reflect.Int64:
		nv, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return zeroval, err
		}
		return reflect.ValueOf(nv), nil
	case reflect.Int8:
		nv, err := strconv.ParseInt(v, 10, 8)
		if err != nil {
			return zeroval, err
		}
		return reflect.ValueOf(int8(nv)), nil
	case reflect.Int16:
		nv, err := strconv.ParseInt(v, 10, 16)
		if err != nil {
			return zeroval, err
		}
		return reflect.ValueOf(int16(nv)), nil
	case reflect.Int32:
		nv, err := strconv.ParseInt(v, 10, 32)
		if err != nil {
			return zeroval, err
		}
		return reflect.ValueOf(int32(nv)), nil
	case reflect.Uint:
		nv, err := strconv.ParseUint(v, 10, 64)
		if err != nil {
			return zeroval, err
		}
		return reflect.ValueOf(uint(nv)), nil
	case reflect.Uint64:
		nv, err := strconv.ParseUint(v, 10, 64)
		if err != nil {
			return zeroval, err
		}
		return reflect.ValueOf(nv), nil
	case reflect.Uint8:
		nv, err := strconv.ParseUint(v, 10, 8)
		if err != nil {
			return zeroval, err
		}
		return reflect.ValueOf(uint8(nv)), nil
	case reflect.Uint16:
		nv, err := strconv.ParseUint(v, 10, 16)
		if err != nil {
			return zeroval, err
		}
		return reflect.ValueOf(uint16(nv)), nil
	case reflect.Uint32:
		nv, err := strconv.ParseUint(v, 10, 32)
		if err != nil {
			return zeroval, err
		}
		return reflect.ValueOf(uint32(nv)), nil
	case reflect.Float64:
		nv, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return zeroval, err
		}
		return reflect.ValueOf(float64(nv)), nil
	case reflect.Float32:
		nv, err := strconv.ParseFloat(v, 32)
		if err != nil {
			return zeroval, err
		}
		return reflect.ValueOf(float32(nv)), nil
	default:
		return zeroval, errors.New("unsupported type")
	}
}

func (tkm type2fields) getStructFields(t reflect.Type) ([]field, error) {
	if t.Kind() != reflect.Struct {
		return nil, errors.New("target is not a struct (Kind: " + t.Kind().String() + ")")
	}

	tkm.lock.RLock()

	km, ok := tkm.types[t]
	if ok {
		tkm.lock.RUnlock()
		return km, nil
	}

	km = make([]field, 0, t.NumField())
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if f.Tag == "" {
			continue
		}
		st := f.Tag.Get("urlenc")
		if st == "" {
			continue
		}

		// strings, numbers, and slices of those two are allowed
		typ, ok := isSupportedType(f.Type, true)
		if !ok {
			return nil, errors.New("urlenc: unsupported type on struct field " + f.Name)
		}

		// urlenc:"foo,omitempty"
		parts := strings.SplitN(st, ",", 2)
		key := strings.TrimSpace(parts[0])
		var omitempty bool
		if len(parts) > 1 && strings.TrimSpace(parts[1]) == "omitempty" {
			omitempty = true
		}

		kf := field{
			Index:     i,
			Kind:      f.Type.Kind(),
			Name:      key,
			OmitEmpty: omitempty,
			Type:      typ,
		}
		switch typ {
		case stringSliceType, numberSliceType:
			kf.ArrayElementType = f.Type.Elem()
			kf.ArrayElementKind = f.Type.Elem().Kind()
		}

		km = append(km, kf)
	}

	tkm.lock.RUnlock()
	tkm.lock.Lock()
	defer tkm.lock.Unlock()

	tkm.types[t] = km
	return km, nil
}

type Marshaller interface {
	MarshalURL() ([]byte, error)
}

type Unmarshaller interface {
	UnmarshalURL([]byte) error
}

func Marshal(v interface{}) ([]byte, error) {
	if u, ok := v.(Marshaller); ok {
		return u.MarshalURL()
	}

	rv := reflect.ValueOf(v)
	if rv == zeroval {
		return nil, errors.New("can not unmarshal into a nil value")
	}

	// This better be a pointer
	switch rv.Kind() {
	case reflect.Ptr, reflect.Interface:
		// Get the value beyond the pointer
		rv = rv.Elem()
	}

	switch rv.Kind() {
	case reflect.Map:
		if kk := rv.Type().Key().Kind(); kk != reflect.String {
			return nil, errors.New("urlenc.Marshal: map key must be string type (Kind: " + kk.String() + ")")
		}
		return marshalMap(rv)
	case reflect.Struct:
		return marshalStruct(rv)
	default:
		return nil, errors.New("urlenc.Marshal: unsupported type (Kind: " + rv.Kind().String() + ")")
	}
}

func addValue(uv *url.Values, name string, fv reflect.Value, typ int) error {
	switch typ {
	case stringType, numberType:
		// If this is a zero value, we skip
		if reflect.Zero(fv.Type()).Interface() == fv.Interface() {
			return nil
		}

		s, err := convertToString(fv)
		if err != nil {
			return err
		}
		uv.Add(name, s)
	case stringSliceType, numberSliceType:
		for i := 0; i < fv.Len(); i++ {
			ev := fv.Index(i)
			s, err := convertToString(ev)
			if err != nil {
				return err
			}
			uv.Add(name, s)
		}
	}
	return nil
}

func marshalMap(rv reflect.Value) ([]byte, error) {
	if rv.Kind() != reflect.Map {
		return nil, errors.New("target is not a map (Kind: " + rv.Kind().String() + ")")
	}

	uv := url.Values{}
	for _, key := range rv.MapKeys() {
		fv := rv.MapIndex(key)
		switch fv.Kind() {
		case reflect.Ptr, reflect.Interface:
			fv = fv.Elem()
		}

		typ, ok := isSupportedType(fv.Type(), true)
		if !ok {
			return nil, errors.New("urlenc: unsupported type on map element " + key.String())
		}

		if err := addValue(&uv, key.String(), fv, typ); err != nil {
			return nil, err
		}
	}
	return []byte(uv.Encode()), nil
}


func marshalStruct(rv reflect.Value) ([]byte, error) {
	fields, err := t2f.getStructFields(rv.Type())
	if err != nil {
		return nil, err
	}

	uv := url.Values{}
	for _, f := range fields {
		fv := rv.Field(f.Index)
		if err := addValue(&uv, f.Name, fv, f.Type); err != nil {
			return nil, err
		}
	}
	return []byte(uv.Encode()), nil
}

var zeroval = reflect.Value{}

func Unmarshal(data []byte, v interface{}) error {
	if u, ok := v.(Unmarshaller); ok {
		return u.UnmarshalURL(data)
	}

	rv := reflect.ValueOf(v)
	if rv == zeroval {
		return errors.New("can not unmarshal into a nil value")
	}

	// This better be a pointer
	if rv.Kind() != reflect.Ptr {
		return errors.New("pointer to a struct required")
	}

	// Get the value beyond the pointer
	rv = rv.Elem()

	// Grab the mapping from struct tags
	fields, err := t2f.getStructFields(rv.Type())
	if err != nil {
		return err
	}

	q, err := url.ParseQuery(string(data))
	if err != nil {
		return err
	}
	for _, f := range fields {
		values := q[f.Name]
		if len(values) <= 0 {
			continue
		}
		fv := rv.Field(f.Index)
		switch f.Type {
		case stringSliceType, numberSliceType:
			av := reflect.MakeSlice(reflect.SliceOf(f.ArrayElementType), len(values), len(values))
			for i := 0; i < len(values); i++ {
				ev := av.Index(i)
				cv, err := convertFromString(f.ArrayElementKind, values[i])
				if err != nil {
					return err
				}
				ev.Set(cv)
			}
			fv.Set(av)
		case stringType, numberType:
			cv, err := convertFromString(f.Kind, values[0])
			if err != nil {
				return err
			}
			fv.Set(cv)
		default:
			return errors.New("unknown type")
		}
	}
	return nil
}