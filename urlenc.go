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

type structfield struct {
	// FieldName is the name of the field, so we can use FieldByName to
	// index into the struct. This is better (albeit less efficient) than
	// using numeric indices, because then we can work directly with
	// embedded structs
	FieldName string
	// KeyName is the name that is used in the resulting query for this field
	KeyName string
	// If true, the field is not included in the query if its value is
	// equal to the zero value of the field type
	OmitEmpty bool
	// Type is the type of this struct field
	Type reflect.Type
}

var t2f = type2fields{
	types: make(map[reflect.Type][]structfield),
}

type type2fields struct {
	lock  sync.RWMutex
	types map[reflect.Type][]structfield
}

func isStringOrNumeric(rk reflect.Kind) bool {
	switch rk {
	case reflect.String:
		return true
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Float32, reflect.Float64:
		return true
	}
	return false
}

func isSupportedType(rt reflect.Type, recurse bool) bool {
	switch rk := rt.Kind(); rk {
	case reflect.Slice, reflect.Array:
		if !recurse {
			return false
		}
		ok := isSupportedType(rt.Elem(), false)
		if !ok {
			return false
		}
		return true
	default:
		return isStringOrNumeric(rk)
	}
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

func (tkm type2fields) getStructFields(t reflect.Type) ([]structfield, error) {
	if t.Kind() != reflect.Struct {
		return nil, errors.New("target is not a struct (Kind: " + t.Kind().String() + ")")
	}

	tkm.lock.RLock()

	km, ok := tkm.types[t]
	if ok {
		tkm.lock.RUnlock()
		return km, nil
	}

	// the fields did not exist in the registry. create and register
	km = make([]structfield, 0, t.NumField())
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)

		var keyname string
		var omitempty bool
		if f.Tag == "" {
			// no tag at all. Use the name of the field as-is
			keyname = f.Name
		} else {
			st := f.Tag.Get("urlenc")
			if st == "" {
				// tag exists, but is empty. Use the name of the field as-is
				keyname = f.Name
			}

			// strings, numbers, and slices of those two are allowed
			if ok := isSupportedType(f.Type, true); !ok {
				return nil, errors.New("urlenc: unsupported type on struct field " + f.Name)
			}

			// urlenc:"foo,omitempty"
			parts := strings.SplitN(st, ",", 2)
			keyname = strings.TrimSpace(parts[0])
			if len(parts) > 1 && strings.TrimSpace(parts[1]) == "omitempty" {
				omitempty = true
			}
		}

		sf := structfield{
			FieldName: f.Name,
			KeyName:   keyname,
			OmitEmpty: omitempty,
			Type:      f.Type,
		}
		km = append(km, sf)
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

// Marshal encodes the given value into a query string. Only structs and maps
// with string keys and several types of types as values are supported.
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

func addValue(uv *url.Values, name string, fv reflect.Value, ft reflect.Type) error {
	if isStringOrNumeric(ft.Kind()) {
		s, err := convertToString(fv)
		if err != nil {
			return err
		}
		uv.Add(name, s)
	} else {
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

		if ok := isSupportedType(fv.Type(), true); !ok {
			return nil, errors.New("urlenc: unsupported type on map element " + key.String())
		}

		if err := addValue(&uv, key.String(), fv, fv.Type()); err != nil {
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
		fv := rv.FieldByName(f.FieldName)

		// Check for empty values
		if f.OmitEmpty {
			if !fv.IsValid() {
				continue
			}

			switch ft := fv.Type(); ft.Kind() {
			case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
				if fv.IsNil() {
					continue
				}
			default:
				switch {
				case fv == zeroval:
					continue
				case fv.CanInterface() && fv.Interface() == reflect.Zero(f.Type).Interface():
					continue
				}
			}
		}

		if err := addValue(&uv, f.KeyName, fv, f.Type); err != nil {
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
		return errors.New("pointer value required")
	}

	// Get the value beyond the pointer
	rv = rv.Elem()

	switch rv.Kind() {
	case reflect.Map:
		if kk := rv.Type().Key().Kind(); kk != reflect.String {
			return errors.New("urlenc.Unmarshal: map key must be string type (Kind: " + kk.String() + ")")
		}
		return unmarshalMap(data, rv)
	case reflect.Struct:
		return unmarshalStruct(data, rv)
	default:
		return errors.New("urlenc.Unmarshal: unsupported type (Kind: " + rv.Kind().String() + ")")
	}
}

func unmarshalMap(data []byte, rv reflect.Value) error {
	q, err := url.ParseQuery(string(data))
	if err != nil {
		return err
	}

	for k, v := range q {
		if len(v) == 1 {
			rv.SetMapIndex(reflect.ValueOf(k), reflect.ValueOf(v[0]))
		} else {
			rv.SetMapIndex(reflect.ValueOf(k), reflect.ValueOf(v))
		}
	}

	return nil
}

func unmarshalStruct(data []byte, rv reflect.Value) error {
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
		values := q[f.KeyName]
		if len(values) <= 0 {
			continue
		}
		fv := rv.FieldByName(f.FieldName)
		switch rk := f.Type.Kind(); rk {
		case reflect.Slice, reflect.Array:
			et := f.Type.Elem() // slice/array element type
			ek := et.Kind()     // slice/array element kind
			av := reflect.MakeSlice(reflect.SliceOf(et), len(values), len(values))
			for i := 0; i < len(values); i++ {
				ev := av.Index(i)
				cv, err := convertFromString(ek, values[i])
				if err != nil {
					return err
				}
				ev.Set(cv)
			}
			fv.Set(av)
		default:
			if !isStringOrNumeric(rk) {
				return errors.New("urlenc.Unmarshal: unsupported type for field " + f.FieldName + " (Kind: " + rk.String() + ")")
			}
			cv, err := convertFromString(f.Type.Kind(), values[0])
			if err != nil {
				return err
			}
			fv.Set(cv)
		}
	}
	return nil
}