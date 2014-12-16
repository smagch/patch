package patch

import (
	"encoding/json"
	"errors"
	"io"
	"reflect"
	"strings"
	"unicode"
	"unicode/utf8"
)

var (
	// errNoInput represents an error for empty JSON input
	errNoInput = errors.New("input is an empty JSON")
	// errInvalidJSONFormat describes that JSON format is invalid
	errInvalidJSONFormat = errors.New("invalid JSON format")
	// errUnexpectedField describes unknow field name
	errUnexpectedField = errors.New("unexpected field")
	// unmarshalling field failed
	errUnmarshalField = errors.New("cannot unmarshal field")
)

// ParseError describes an error for parsing JSON input
type ParseError struct {
	// JSON property name that produced the error
	Key string
	// reason of the error.
	err error
	// original error message
	detail string
}

// Error implements error interface.
func (e *ParseError) Error() string {
	s := "patch:"
	if e.err != nil {
		s += e.err.Error()
	}
	if e.Key != "" {
		s += " on key '" + e.Key + "'"
	}
	if e.detail != "" {
		s += ", " + e.detail
	}
	return s
}

// newField parses reflect.StructField to set of structField and json property
func parseField(v reflect.StructField) (name, propName string, ok bool) {
	if r, _ := utf8.DecodeRuneInString(v.Name); !unicode.IsUpper(r) {
		return
	}

	jsonTagName := trimCommaLeft(v.Tag.Get("json"))
	if jsonTagName != "" {
		if jsonTagName == "-" {
			return
		}
		propName = jsonTagName
	} else {
		propName = v.Name
	}

	name = v.Tag.Get("patch")
	if name == "-" {
		return
	}
	if name == "" {
		name = propName
	}

	return name, propName, true
}

// structField
type structField struct {
	// name of the field.
	name string
	typ  reflect.Type
	// index of struct field
	index int
}

// unmarshal takes the given bytes to its type.
func (f *structField) unmarshal(b []byte) (interface{}, error) {
	v := reflect.New(f.typ)
	if err := json.Unmarshal(b, v.Interface()); err != nil {
		return nil, err
	}
	return v.Elem().Interface(), nil
}

// Patcher is a json parser that takes fileds partially.
type Patcher struct {
	fields map[string]*structField
}

// trimCommaLeft omits strings after ","
func trimCommaLeft(s string) string {
	i := strings.IndexRune(s, ',')
	if i != -1 {
		return s[:i]
	}
	return s
}

// parseStruct parse struct fields.
func parseStruct(src interface{}) map[string]*structField {
	typ := reflect.TypeOf(src)
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	if typ.Kind() != reflect.Struct {
		panic("patch: src should be a struct. But " + typ.Kind().String() + " is given")
	}
	fields := make(map[string]*structField)
	for i := 0; i < typ.NumField(); i++ {
		v := typ.Field(i)
		name, propName, ok := parseField(v)
		if ok {
			fields[propName] = &structField{
				name:  name,
				typ:   v.Type,
				index: i,
			}
		}
	}
	return fields
}

// New returns a pointer of Patcher with the given struct value.
// It panics when type of src isn't struct or pointer of struct.
func New(src interface{}) *Patcher {
	return &Patcher{parseStruct(src)}
}

// Unmarshal unmarshal the given bytes to Fields that is sorted in order of
// struct index.
func (p *Patcher) Unmarshal(src []byte) (Fields, error) {
	v := make(map[string]json.RawMessage)
	if err := json.Unmarshal(src, &v); err != nil {
		return nil, &ParseError{err: errInvalidJSONFormat, detail: err.Error()}
	}
	return p.parseFields(v)
}

// Decode decodes the given read stream to Fields.
func (p *Patcher) Decode(r io.Reader) (Fields, error) {
	v := make(map[string]json.RawMessage)
	if err := json.NewDecoder(r).Decode(&v); err != nil {
		return nil, &ParseError{err: errInvalidJSONFormat, detail: err.Error()}
	}
	return p.parseFields(v)
}

// parseFields parses the given map of json.RawMessages to values looking up
// Patcher's pre-parsed types.
func (p *Patcher) parseFields(values map[string]json.RawMessage) (Fields, error) {
	if len(values) == 0 {
		return nil, &ParseError{err: errNoInput}
	}
	var i int
	data := make(Fields, len(values))
	for prop, msg := range values {
		f, ok := p.fields[prop]
		if !ok {
			return nil, &ParseError{err: errUnexpectedField, Key: prop}
		}
		v, err := f.unmarshal(msg)
		if err != nil {
			return nil, &ParseError{err: errUnmarshalField, Key: prop, detail: err.Error()}
		}
		data[i] = Field{f.name, v, f.index}
		i++
	}
	data.sort()
	return data, nil
}
