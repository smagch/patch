package patch

import (
	"sort"
)

// byFieldIndex implements sort.Interface interface so that Fields can be in
// order of struct field index.
type byFieldIndex []Field

func (f byFieldIndex) Len() int {
	return len(f)
}

func (f byFieldIndex) Swap(i, j int) {
	f[i], f[j] = f[j], f[i]
}

func (f byFieldIndex) Less(i, j int) bool {
	return f[i].index < f[j].index
}

// field represents a parsed JSON field.
type Field struct {
	Key   string
	Value interface{}
	index int
}

// Fields is a slice of parsed struct fields.
type Fields []Field

// sort make fields in order of struct field index.
func (f Fields) sort() {
	sort.Sort(byFieldIndex(f))
}

// Keys returns a slice of key strings.
func (f Fields) Keys() []string {
	keys := make([]string, len(f))
	for i, data := range f {
		keys[i] = data.Key
	}
	return keys
}

// Values returns a slice of values.
func (f Fields) Values() []interface{} {
	values := make([]interface{}, len(f))
	for i, data := range f {
		values[i] = data.Value
	}
	return values
}

// Map converts fields as map.
func (f Fields) Map() map[string]interface{} {
	m := make(map[string]interface{})
	for _, data := range f {
		m[data.Key] = data.Value
	}
	return m
}

// getIndex returns an index with the given key name.
func (f Fields) getIndex(name string) int {
	for i, data := range f {
		if name == data.Key {
			return i
		}
	}
	return -1
}

// Get returns a value with the given key name.
func (f Fields) Get(name string) (interface{}, bool) {
	for _, data := range f {
		if name == data.Key {
			return data.Value, true
		}
	}
	return nil, false
}

// Set appends the given value with the given name.
// It overwrite the value if name exists.
func (f *Fields) Set(name string, value interface{}) {
	i := f.getIndex(name)
	elem := *f

	if i != -1 {
		elem[i].Value = value
	} else {
		v := Field{name, value, -1}
		*f = append(elem, v)
	}
}

// Remove removes field data with the given key name.
func (f *Fields) Remove(name string) interface{} {
	index := f.getIndex(name)
	if index == -1 {
		return nil
	}
	elem := *f
	v := elem[index]
	*f = append(elem[:index], elem[index+1:]...)
	return v.Value
}

// SQL returns a SQL with the given Field.
func (f Fields) SQL() *SQL {
	return &SQL{Fields: f}
}
