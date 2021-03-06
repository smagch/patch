# patch

JSON marshaler to work well with HTTP `PATCH` method.

[![GoDoc](https://godoc.org/github.com/smagch/patch?status.svg)](https://godoc.org/github.com/smagch/patch)
[![Build Status](https://travis-ci.org/smagch/patch.svg)](https://travis-ci.org/smagch/patch)

Unmarshalling JSON that has partial json properties is a bit tricky when you
work with a `PATCH` API in golang. Because fields are set to zero value even if
JSON doesn't have the fields.

In order to solve this problem, `Patcher` unmarshals JSON to `map[string]json.RawMessage`
instead of a struct.

```go
type Article struct {
    ID     int    `json:"id"`
    UserID int    `json:"user_id"`
    Title  string `json:"title"`
    Desc   string `json:"desc"`
    Body   string `json:"body"`
}
p := patch.New(Article{})
f, err := p.Unmarshal([]byte(`{"title": "the gopher", "desc": "description"}`))
// f.Keys() == ["title", "desc"]
// f.Values() == ["the gopher", "description"]
```

See [godoc](http://godoc.org/github.com/smagch/patch) for more details.
