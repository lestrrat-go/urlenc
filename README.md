# go-urlenc

Marshal/Unmarshal interface for structs that can encode/decode themselves to URL query strings

[![Build Status](https://travis-ci.org/lestrrat/go-urlenc.svg?branch=master)](https://travis-ci.org/lestrrat/go-urlenc)

[![GoDoc](https://godoc.org/github.com/lestrrat/go-urlenc?status.svg)](https://godoc.org/github.com/lestrrat/go-urlenc)

# Synopsis

```go
package urlenc_test

import (
  "log"

  "github.com/lestrrat/go-urlenc"
)

type Foo struct {
  Bar   string    `urlenc:"bar"`
  Baz   int       `urlenc:"baz"`
  Qux   []string  `urlenc:"qux"`
  Corge []float64 `urlenc:"corge"`
}

func Example() {
  const src = `bar=one&baz=2&qux=three&qux=4&corge=1.41421356237&corge=2.2360679775`

  var foo Foo
  if err := urlenc.Unmarshal([]byte(src), &foo); err != nil {
    return
  }

  log.Printf("Bar = '%s'", foo.Bar)
  log.Printf("Baz = '%d'", foo.Baz)
  log.Printf("Qux = %v", foo.Qux)
  log.Printf("Corge = %v", foo.Corge)
}
```

# Struct Tags

Struct tags for this package take the following format:

```
urlenc:"name,omitempty,typename"
```

The value that you place in the location of `name` above will be used as the
query string key name. If not provided, the exact name of the struct field
is used.

In the location of `omitempty`, you may specify that exact keyword (`omitempty`)
and you can specify to remove this key/value from the query component if
the value is equal to its zero value.

Lastly, `typename` allows you to specify the type name that you are "pretending"
to use as for that field. For example, you may be using a struct to represent
a possibly uninitialized integer value like this:

```go
type Payload struct {
  Number struct {
    Valid bool // true if this value has been initialized
    Int   int  // the actual integer value
  } `urlenc:"number,omitempty,int"`
}
```

In this case you want to pretend that `Number` is an integer, but it actually isn't.
By specifying this third parameter, this package tries its best to try and handle
this value as an integer

Incidentally, if you use this option you almost always want to use the `Setter` and
`Valuer` interfaces. See elsewhere in this document for details

# Setter/Valuer interfaces

Sometimes you want to pretend as if a struct is actually a simple type that this
package can handle. But marshaling/unmarshaling to non-simple structs requires
a bit more trickery.

For values that know how to extract values out of it, implement the following
`Valuer` interface:

```go
type Valuer interface {
  Value() interface{}
}
```

Value to be encoded will be taken from the return value of this function.
Note that it must match the value you specified in `typename` field of
the urlenc struct tag.

For values that know how to set values to it, implement the following `Setter`
interface:

```go
type Setter interface {
  Set(interface{}) error
}
```

Decoded values will be passed to the Set method.
