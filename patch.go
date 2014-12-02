package patch

import (
	"bytes"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"unicode"
	"unicode/utf8"
)

var (
	// ErrNoInput is an error for empty JSON  input
	ErrNoInput         = &ValidationError{Err: errors.New("There is no input")}
	ErrInvalidProperty = errors.New("Invalid property")
)

var drivers = map[string]Driver{
	"postgres": Postgres,
}

// SetDriver registers the driver with the given driver name
func SetDriver(driverName string, driver Driver) {
	drivers[driverName] = driver
}

// ValidationError
type ValidationError struct {
	// Object property name fo the error
	Key string
	// Err is the original error
	Err    error
	reason string
}

func (err *ValidationError) Error() string {
	var b bytes.Buffer
	b.WriteString("ValidationError: ")
	if err.reason != "" {
		b.WriteString(err.reason)
	}
	if err.Key != "" {
		b.WriteString(" with key " + err.Key)
	}
	if err.Err != nil {
		b.WriteString(": ")
		b.WriteString(err.Err.Error())
	}
	return b.String()
}

func isFirstRuneUpper(s string) bool {
	r, _ := utf8.DecodeRuneInString(s)
	return unicode.IsUpper(r)
}

// newField
func newField(v reflect.StructField) (f *structField, propName string) {
	if !isFirstRuneUpper(v.Name) {
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

	patchName := v.Tag.Get("patch")
	if patchName == "-" {
		return
	}

	f = new(structField)
	if patchName != "" {
		f.columnName = patchName
	} else {
		f.columnName = propName
	}

	f.dec = &fieldDecoder{v.Type}
	return f, propName
}

type fieldDecoder struct {
	typ reflect.Type
}

func (dec *fieldDecoder) Decode(b []byte) (interface{}, error) {
	v := reflect.New(dec.typ).Interface()
	if err := json.Unmarshal(b, v); err != nil {
		return nil, err
	}
	return reflect.ValueOf(v).Elem().Interface(), nil
}

// structField
type structField struct {
	columnName string
	dec        *fieldDecoder
	callback   Callback
}

// Patcher
type Patcher struct {
	fields map[string]*structField
	driver Driver
}

// Callback transforms data with the given input
type Callback func(src interface{}) (interface{}, error)

// On register a replacer function with the given json property name
func (p *Patcher) On(jsonProp string, fn Callback) *Patcher {
	f, ok := p.fields[jsonProp]
	if !ok {
		panic("Cannot listen to invalid property name: " + jsonProp)
	}
	f.callback = fn
	return p
}

// trimCommaLeft omits strings after ","
func trimCommaLeft(s string) string {
	i := strings.IndexRune(s, ',')
	if i != -1 {
		return s[:i]
	}
	return s
}

// parseStruct
func parseStruct(src interface{}) map[string]*structField {
	typ := reflect.TypeOf(src)
	fields := make(map[string]*structField)
	for i := 0; i < typ.NumField(); i++ {
		v := typ.Field(i)
		f, propName := newField(v)
		if f != nil {
			fields[propName] = f
		}
	}
	return fields
}

// New returns a pointer of Patcher
func New(driverName string, src interface{}) *Patcher {
	d, ok := drivers[driverName]
	if !ok || d == nil {
		panic("Unsupported driver: " + driverName)
	}
	// validate if struct or struct of pointer
	return &Patcher{
		fields: parseStruct(src),
		driver: d,
	}
}

// Parse decodes the given byte
func (p Patcher) Parse(src []byte) (*Data, error) {
	values := make(map[string]json.RawMessage)
	if err := json.Unmarshal(src, &values); err != nil {
		return nil, &ValidationError{Err: err, reason: "Invalid JSON Format"}
	}
	if len(values) == 0 {
		return nil, ErrNoInput
	}
	var i int
	data := p.newData(len(values))
	for prop, msg := range values {
		f, ok := p.fields[prop]
		if !ok {
			return nil, &ValidationError{
				Key: prop,
				Err: ErrInvalidProperty,
			}
		}
		v, err := f.dec.Decode(msg)
		if err != nil {
			return nil, &ValidationError{Err: err, Key: prop, reason: "Decode Error"}
		}
		if f.callback != nil {
			v, err = f.callback(v)
			if err != nil {
				return nil, &ValidationError{Err: err, Key: prop, reason: "ReplacerError"}
			}
		}
		data.columns[i] = f.columnName
		data.args[i] = v
		i++
	}
	return data, nil
}

// newData retuns a pointer of Data with the given property length
func (p Patcher) newData(length int) *Data {
	return &Data{
		columns: make([]string, length),
		args:    make([]interface{}, length),
		driver:  p.driver,
	}
}
