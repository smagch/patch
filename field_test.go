package patch

import (
	"reflect"
	"testing"
)

func TestFields(t *testing.T) {
	f := Fields{
		{"id", 1, 1},
		{"name", "golang", 4},
		{"hashtag", "#golang", 10},
		{"static", true, 6},
	}
	f.sort()

	for _, field := range f {
		v, ok := f.Get(field.Key)
		if !ok {
			t.Fatal("should be ok")
		}
		if v != field.Value {
			t.Fatalf("want %v, got %v", field.Value, v)
		}
	}

	testCases := []struct {
		ret      interface{}
		expected interface{}
	}{
		{f.Keys(), []string{"id", "name", "static", "hashtag"}},
		{f.Values(), []interface{}{1, "golang", true, "#golang"}},
		{f.Map(), map[string]interface{}{
			"id":      1,
			"name":    "golang",
			"hashtag": "#golang",
			"static":  true,
		}},
	}

	for i, tc := range testCases {
		if !reflect.DeepEqual(tc.ret, tc.expected) {
			t.Fatalf("%d: want %v, got %v", i, tc.expected, tc.ret)
		}
	}
}

func TestFieldsSet(t *testing.T) {
	f := Fields{
		{"name", "go", 1},
		{"hello", "world", 2},
	}
	setTests := []struct {
		key      string
		value    interface{}
		expected Fields
	}{
		{
			"name", "gopher", Fields{
				{"name", "gopher", 1},
				{"hello", "world", 2},
			},
		},
		{
			"appened", true, Fields{
				{"name", "gopher", 1},
				{"hello", "world", 2},
				{"appened", true, -1},
			},
		},
	}

	for i, tc := range setTests {
		f.Set(tc.key, tc.value)
		v, ok := f.Get(tc.key)
		if !ok {
			t.Fatal(i, ":should exist key: ", tc.key)
		}
		if v != tc.value {
			t.Fatalf("%d: want %v, got %v", i, tc.value, v)
		}
		if !reflect.DeepEqual(f, tc.expected) {
			t.Fatalf("%d: want %v, got %v", i, tc.expected, f)
		}
	}
}

func TestFieldsRemove(t *testing.T) {
	f := Fields{
		{"id", 1, 1},
		{"name", "golang", 4},
		{"hashtag", "#golang", 10},
		{"static", true, 6},
	}
	f.sort()

	removeTests := []struct {
		key  string
		v    interface{}
		keys []string
	}{
		{"hashtag", "#golang", []string{"id", "name", "static"}},
		{"name", "golang", []string{"id", "static"}},
		{"id", 1, []string{"static"}},
		{"static", true, []string{}},
	}

	for i, tc := range removeTests {
		v := f.Remove(tc.key)
		if v != tc.v {
			t.Fatalf("%d: want %v, got %v", i, tc.v, v)
		}
		keys := f.Keys()
		if !reflect.DeepEqual(tc.keys, keys) {
			t.Fatalf("%d: want %v, got %v", i, tc.keys, keys)
		}
	}
}
