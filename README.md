# go-urlenc

Marshal/Unmarshal interface for structs that can encode/decode themselves to URL query strings

# Synopsis

```go
package urlenc_test

import (
  "log"

  "github.com/lestrrat/go-urlenc
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
