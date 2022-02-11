# querystring #

querystring is a Go library for encoding structs into URL query parameters.

## Usage ##

```go
import "github.com/rumis/querystring/query"
```

querystring is designed to assist in scenarios where you want to construct a
URL using a struct that represents the URL query parameters.  You might do this
to enforce the type safety of your parameters, for example, as is done in the
[go-github][] library.

Support for primitive types (bool, int, etc...), pointers, slices, arrays, maps, structs, time.Time.

A custom type can implement the Encoder interfaces to handle its own marshaling.

A struct field tag can be used to:
* Exclude a field from marshaling by specifying - as the field name (qs:"-").
* Set custom name for the field in the marshaled query string.
* Set one of the omitempty options for marshaling.

The query package exports a single `Values()` function.  A simple example:

```go
type Options struct {
  Query   string `qs:"q"`
  ShowAll bool   `qs:"all"`
  Page    int    `qs:"page"`
}

opt := Options{ "foo", true, 2 }
v, _ := query.Values(opt)
fmt.Print(v.Encode()) // will output: "q=foo&all=true&page=2"
```

See the [package godocs][] for complete documentation on supported types and
formatting options.

[go-github]: https://github.com/google/go-github/commit/994f6f8405f052a117d2d0b500054341048fbb08
[package godocs]: https://pkg.go.dev/github.com/rumis/querystring/query

## Alternatives ##

If you are looking for a library that can both encode and decode query strings,
you might consider one of these alternatives:

 - https://github.com/gorilla/schema
 - https://github.com/pasztorpisti/qs
 - https://github.com/hetiansu5/urlquery

