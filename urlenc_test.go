package urlenc_test

import (
	"errors"
	"net/url"
	"reflect"
	"testing"

	"github.com/lestrrat/go-urlenc"
	"github.com/stretchr/testify/assert"
)

type MaybeString struct {
	Valid  bool
	String string
}

type MaybeStringSlice struct {
	Valid bool
	Slice []string
}

func (m MaybeString) Value() interface{} {
	return m.String
}

func (m *MaybeString) Set(v interface{}) error {
	switch v.(type) {
	case string:
		m.Valid = true
		m.String = v.(string)
	default:
		return errors.New("expected string (got: " + reflect.TypeOf(v).String() + ")")
	}
	return nil
}

func (m MaybeStringSlice) Value() interface{} {
	return m.Slice
}

func (m *MaybeStringSlice) Set(v interface{}) error {
	switch v.(type) {
	case []string:
		m.Valid = true
		m.Slice = v.([]string)
	default:
		return errors.New("expected string slice (got: " + reflect.TypeOf(v).String() + ")")
	}
	return nil
}

type Foo struct {
	Bar          string           `urlenc:"bar"`
	Baz          int              `urlenc:"baz"`
	Qux          []string         `urlenc:"qux"`
	Corge        []float64        `urlenc:"corge"`
	Grault       bool             `urlenc:"grault"`
	Garply       []bool           `urlenc:"garply"`
	Special      MaybeString      `urlenc:"special,omitempty,string"`
	SpecialSlice MaybeStringSlice `urlenc:"sslice,omitempty,[]string"`
}

type FooJ struct {
	BarJ string `json:"bar"`
}

func TestUnmarshal(t *testing.T) {
	const src = `bar=one&baz=2&qux=three&qux=4&corge=1.41421356237&corge=2.2360679775&garply=true&garply=false&grault=true&special=special&sslice=five&sslice=6`

	var foo Foo
	if !assert.NoError(t, urlenc.Unmarshal([]byte(src), &foo), "Unmarshal should succeed") {
		return
	}

	if !assert.Equal(t, foo.Bar, "one", "Bar is 'one'") {
		return
	}
	if !assert.Equal(t, foo.Baz, 2, "Baz is '2'") {
		return
	}
	if !assert.Equal(t, foo.Qux, []string{"three", "4"}, "Qux is 'three, 4'") {
		return
	}
	if !assert.Equal(t, foo.Corge, []float64{1.41421356237, 2.2360679775}, "Corge is '1.41421356237, 2.2360679775'") {
		return
	}
	if !assert.Equal(t, foo.Special.String, "special", "Spcial is 'special'") {
		return
	}
	if !assert.Equal(t, foo.SpecialSlice.Slice, []string{"five", "6"}, `SpcialSlice is '"five", "6"'`) {
		return
	}

	if !assert.Equal(t, foo.Grault, true, `Grault is true`) {
		return
	}
	if !assert.Equal(t, foo.Garply, []bool{true, false}, `Garply is true, false`) {
		return
	}
}

func TestMarshal(t *testing.T) {
	const src = `bar=one&baz=2&qux=three&qux=4&corge=1.41421356237&corge=2.2360679775&grault=false&special=special&sslice=five&sslice=6`

	foo := Foo{
		Bar:          "one",
		Baz:          2,
		Qux:          []string{"three", "4"},
		Corge:        []float64{1.41421356237, 2.2360679775},
		Special:      MaybeString{Valid: true, String: "special"},
		SpecialSlice: MaybeStringSlice{Valid: true, Slice: []string{"five", "6"}},
	}
	buf, err := urlenc.Marshal(foo)
	if !assert.NoError(t, err, "Marshal should succeed") {
		return
	}

	produced, err := url.ParseQuery(string(buf))
	if !assert.NoError(t, err, "ParseQuery should succeed") {
		return
	}
	expected, err := url.ParseQuery(src)
	if !assert.NoError(t, err, "ParseQuery should succeed") {
		return
	}

	if !assert.Equal(t, produced, expected, "Marshal produces the same result") {
		return
	}
}

type ZeroInt struct {
	Limit int `urlenc:"limit,omitempty"`
}

func TestMarshalZeroInt(t *testing.T) {
	buf, err := urlenc.Marshal(ZeroInt{})
	if !assert.NoError(t, err, "Marshal should succeed") {
		return
	}

	if !assert.Equal(t, "", string(buf), "zero values don't get marshaled") {
		return
	}
}

func TestMarshalMap(t *testing.T) {
	m := make(map[string]interface{})
	m["bar"] = "one"
	m["baz"] = 2
	m["qux"] = []string{"three", "4"}
	m["corge"] = []float64{1.41421356237, 2.2360679775}

	buf, err := urlenc.Marshal(m)
	if !assert.NoError(t, err, "Marshal should succeed") {
		return
	}

	produced, err := url.ParseQuery(string(buf))
	if !assert.NoError(t, err, "ParseQuery should succeed") {
		return
	}
	const src = `bar=one&baz=2&qux=three&qux=4&corge=1.41421356237&corge=2.2360679775`
	expected, err := url.ParseQuery(src)
	if !assert.NoError(t, err, "ParseQuery should succeed") {
		return
	}
	if !assert.Equal(t, produced, expected, "Marshal produces the same result") {
		return
	}
}

func TestUnmarshalMap(t *testing.T) {
	const src = `bar=one&baz=2&qux=three&qux=4&corge=1.41421356237&corge=2.2360679775`

	m := make(map[string]interface{})
	if !assert.NoError(t, urlenc.Unmarshal([]byte(src), &m), "Unmarshal succeeds") {
		return
	}

	expected := make(map[string]interface{})
	expected["bar"] = "one"
	expected["baz"] = "2" // Note, not integer 2, but string "2", because we can't tell them apart
	expected["qux"] = []string{"three", "4"}
	expected["corge"] = []string{"1.41421356237", "2.2360679775"} // Same as "baz"

	if !assert.Equal(t, m, expected, "Unmarshal produces the expected result") {
		return
	}
}

func TestMarshalJSONFallback(t *testing.T) {
	s := FooJ{
		BarJ: "one",
	}

	buf, err := urlenc.Marshal(s)
	if !assert.NoError(t, err, "Marshal succeeds") {
		return
	}

	produced, err := url.ParseQuery(string(buf))
	if !assert.NoError(t, err, "ParseQuery should succeed") {
		return
	}
	const src = `bar=one`
	expected, err := url.ParseQuery(src)
	if !assert.NoError(t, err, "ParseQuery should succeed") {
		return
	}
	if !assert.Equal(t, produced, expected, "Marshal produces the same result") {
		return
	}
}

func TestUnmarshalJSONFallback(t *testing.T) {
	const src = `bar=one`
	var s FooJ
	if !assert.NoError(t, urlenc.Unmarshal([]byte(src), &s), "Unmarshal succeeds") {
		return
	}

	expected := FooJ{
		BarJ: "one",
	}
	if !assert.Equal(t, s, expected, "Unmarshal produces the expected result") {
		return
	}
}

type RackStylePayload struct {
	Foo   string   `urlenc:"foo"`
	Names []string `urlenc:"names[]"`
}

func TestRackStyleQuery(t *testing.T) {
	const src = `foo=bar&names[]=foo&names[]=bar`
	var s RackStylePayload
	if !assert.NoError(t, urlenc.Unmarshal([]byte(src), &s), "Unmarshal succeeds") {
		return
	}
	expected := RackStylePayload{
		Foo:   "bar",
		Names: []string{"foo", "bar"},
	}
	if !assert.Equal(t, s, expected, "Unmarshal produces the expected result") {
		return
	}
}
