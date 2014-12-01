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
	ErrNoInput = &ValidationError{Err: errors.New("There is no input")}
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
	f = new(structField)
	if name := v.Tag.Get("patch"); name != "" {
		f.columnName = name
	} else {
		f.columnName = propName
	}
	f.dec = getUnmarshaler(v.Type)
	return f, propName
}

type unmarshalerDecoder struct {
	typ reflect.Type
}

func (dec *unmarshalerDecoder) Decode(b []byte) (interface{}, error) {
	u := reflect.New(dec.typ.Elem()).Interface().(json.Unmarshaler)
	if err := u.UnmarshalJSON(b); err != nil {
		return nil, err
	}
	return u, nil
}

type unmarshalerPtrDecoder struct {
	typ reflect.Type
}

func (dec *unmarshalerPtrDecoder) Decode(b []byte) (interface{}, error) {
	u := reflect.New(dec.typ).Interface().(json.Unmarshaler)
	if err := u.UnmarshalJSON(b); err != nil {
		return nil, err
	}
	return reflect.ValueOf(u).Elem().Interface(), nil
}

type defaultDecoder struct {
	typ reflect.Type
}

func (dec *defaultDecoder) Decode(b []byte) (interface{}, error) {
	v := reflect.Zero(dec.typ).Interface()
	if err := json.Unmarshal(b, &v); err != nil {
		return nil, err
	}
	return v, nil
}

func getUnmarshaler(typ reflect.Type) decoder {
	if typ.Kind() == reflect.Ptr {
		_, ok := reflect.New(typ.Elem()).Interface().(json.Unmarshaler)
		if ok {
			return &unmarshalerDecoder{typ}
		}
	}
	_, ok := reflect.New(typ).Interface().(json.Unmarshaler)
	if ok {
		return &unmarshalerPtrDecoder{typ}
	}
	// TODO slice or map could possibly unmarshal without using a pointer receiver
	// _, ok := reflect.Zero(typ).Interface().(json.Unmarshaler)
	if dec, ok := numberDecoders[typ.Kind()]; ok {
		return dec
	}
	return &defaultDecoder{typ}
}

// structField
type structField struct {
	columnName string
	dec        decoder
}

// Patcher
type Patcher struct {
	fields map[string]*structField
	driver Driver
}

// trimCommaLeft omits strings after ","
func trimCommaLeft(s string) string {
	i := strings.IndexRune(s, ',')
	if i != -1 {
		return s[:i]
	}
	return s
}

// decoder is the interface of unmarshalling of json object
type decoder interface {
	Decode(b []byte) (interface{}, error)
}

// parseStruct
func parseStruct(src interface{}) map[string]*structField {
	typ := reflect.TypeOf(src)
	fields := make(map[string]*structField)
	for i := 0; i < typ.NumField(); i++ {
		v := typ.Field(i)
		// if it's a pointer type, then v.(json.Marshaler)
		// if it's not a pointer, examine its pointerV.(json.Marshaler)
		// if either has marshaler interface, then
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

// numberDecoders decodes Numbers except for Float64
// TODO Int8, Int16, Int32, Int64, Uint, Unit8, Uint16, Uint32, Uint64, Float32
var numberDecoders = map[reflect.Kind]decoder{
	reflect.Int: decoderFunc(decodeInt),
}

type decoderFunc func(b []byte) (interface{}, error)

// Decode just calls f
func (f decoderFunc) Decode(b []byte) (interface{}, error) {
	return f(b)
}

// decodeInt decode the given byte to int.
func decodeInt(b []byte) (interface{}, error) {
	var i int
	err := json.Unmarshal(b, &i)
	return i, err
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
				Key:    prop,
				reason: "Unexpected object property",
			}
		}
		v, err := f.dec.Decode(msg)
		if err != nil {
			return nil, &ValidationError{Err: err, Key: prop, reason: "Decode Error"}
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
