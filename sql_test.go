package patch

import (
	"reflect"
	"testing"
)

func TestQueryPpostgres(t *testing.T) {
	data := Fields{
		{"name", "golang", 1},
		{"email", "f@oo.bar", 2},
		{"power", 100, 3},
	}

	testCases := []struct {
		append  []interface{}
		prepend []interface{}
		query   string
		args    []interface{}
	}{
		{
			[]interface{}{123},
			nil,
			`name=$2,email=$3,power=$4`,
			[]interface{}{123, "golang", "f@oo.bar", 100},
		},
		{
			[]interface{}{100, "user_type"},
			nil,
			`name=$3,email=$4,power=$5`,
			[]interface{}{100, "user_type", "golang", "f@oo.bar", 100},
		},
		{
			[]interface{}{},
			nil,
			`name=$1,email=$2,power=$3`,
			[]interface{}{"golang", "f@oo.bar", 100},
		},
		{
			[]interface{}{2000},
			[]interface{}{"foo", "bar"},
			`name=$4,email=$5,power=$6`,
			[]interface{}{"foo", "bar", 2000, "golang", "f@oo.bar", 100},
		},
	}

	fatal := "%d: want %v, got %v"
	for i, tc := range testCases {
		s := data[:].SQL()
		s.Prepend(tc.prepend...)
		q, args := s.QueryPostgres(tc.append...)
		if q != tc.query {
			t.Fatalf(fatal, i, tc.query, q)
		}
		if !reflect.DeepEqual(args, tc.args) {
			t.Fatalf(fatal, i, tc.args, args)
		}
	}
}

func TestQuery(t *testing.T) {
	data := Fields{
		{"name", "golang", 1},
		{"email", "f@oo.bar", 2},
		{"power", 100, 3},
	}

	query := `name=?,email=?,power=?`
	testCases := []struct {
		append  []interface{}
		prepend []interface{}
		args    []interface{}
	}{
		{
			[]interface{}{123},
			nil,
			[]interface{}{"golang", "f@oo.bar", 100, 123},
		},
		{
			[]interface{}{100, "user_type"},
			nil,
			[]interface{}{"golang", "f@oo.bar", 100, 100, "user_type"},
		},
		{
			[]interface{}{},
			nil,
			[]interface{}{"golang", "f@oo.bar", 100},
		},
		{
			[]interface{}{2000},
			[]interface{}{"foo", "bar"},
			[]interface{}{"foo", "bar", "golang", "f@oo.bar", 100, 2000},
		},
	}

	fatal := "%d: want %v, got %v"
	for i, tc := range testCases {
		d := data[:].SQL()
		d.Prepend(tc.prepend...)
		q, args := d.Query(tc.append...)
		if q != query {
			t.Fatalf(fatal, i, query, q)
		}
		if !reflect.DeepEqual(args, tc.args) {
			t.Fatalf(fatal, i, tc.args, args)
		}
	}
}
