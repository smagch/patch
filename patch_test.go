package patch

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
)

func sliceToMap(columns []string, args []interface{}) map[string]interface{} {
	m := make(map[string]interface{})
	for i, name := range columns {
		m[name] = args[i]
	}
	return m
}

type Weekday int

// UnmarshalJSON implements json.Unmarshaler interface.
func (d *Weekday) UnmarshalJSON(b []byte) error {
	s := string(b)
	if len(s) < 5 {
		return errors.New("Invalid Weekday: " + s)
	}
	day, err := ParseWeekday(s[1 : len(s)-1])
	if err != nil {
		return err
	}
	if d == nil {
		return errors.New("nil pointer")
	}
	*d = day
	return nil
}

const (
	Sunday    = Weekday(time.Sunday)
	Monday    = Weekday(time.Monday)
	Tuesday   = Weekday(time.Tuesday)
	Wednesday = Weekday(time.Wednesday)
	Thursday  = Weekday(time.Thursday)
	Friday    = Weekday(time.Friday)
	Saturday  = Weekday(time.Saturday)
)

// ParseWeekday parses string to Weekday
func ParseWeekday(str string) (d Weekday, err error) {
	switch strings.ToLower(str) {
	case "sunday":
		d = Sunday
	case "monday":
		d = Monday
	case "tuesday":
		d = Tuesday
	case "wednesday":
		d = Wednesday
	case "thursday":
		d = Thursday
	case "friday":
		d = Friday
	case "saturday":
		d = Saturday
	default:
		err = errors.New("invalid weekday: " + str)
	}
	return
}

func TestInvalidProperties(t *testing.T) {
	type user struct {
		JsonIgnore  int `json:"-"`
		PatchIgnore int `patch:"-"`
		Id          int
	}
	p := New("postgres", user{})

	testCases := []struct {
		body string
		key  string
	}{
		{`{"Id": 100, "JsonIgnore": 1}`, "JsonIgnore"},
		{`{"PatchIgnore": 1000}`, "PatchIgnore"},
	}

	for i, tc := range testCases {
		_, err := p.Parse([]byte(tc.body))
		v, ok := err.(*ValidationError)
		if !ok {
			t.Fatal(i, "Expected ValidationError")
		}
		if tc.key != v.Key {
			t.Error(i, "Unexpected Key: ", v.Key, " want ", tc.key)
		}
		if v.Err != ErrInvalidProperty {
			t.Error(i, "Expected ErrInvalidProperty. got ", v.Err)
		}
	}
}

func TestNew(t *testing.T) {
	type user struct {
		Id          int
		Name        string     `json:"name"`
		Email       string     `json:"email" patch:"email_address"`
		Active      bool       `patch:"active"`
		Day         Weekday    `json:"day"`
		DayPtr      *Weekday   `json:"day_ptr"`
		Span        [2]Weekday `json:"span"`
		Days        []Weekday  `json:"days"`
		NullableInt *int
	}
	p := New("postgres", user{})
	testCases := []struct {
		body string
		args map[string]interface{}
	}{
		{
			`{"name": "sunday"}`,
			map[string]interface{}{"name": "sunday"},
		},
		{
			`{"name": "foo", "Active": true}`,
			map[string]interface{}{"name": "foo", "active": true},
		},
		{
			`{"NullableInt": null}`,
			map[string]interface{}{"NullableInt": (*int)(nil)},
		},
		{
			`{"Id": 1}`,
			map[string]interface{}{"Id": int(1)},
		},
		{
			`{"day":"wednesday"}`,
			map[string]interface{}{"day": Wednesday},
		},
		{
			`{"day_ptr":"friday"}`,
			map[string]interface{}{"day_ptr": Friday},
		},
		{
			`{"span": ["sunday", "wednesday"]}`,
			map[string]interface{}{"span": [2]Weekday{Sunday, Wednesday}},
		},
		{
			`{"days": ["sunday", "monday", "friday", "saturday"]}`,
			map[string]interface{}{"days": []Weekday{Sunday, Monday, Friday, Saturday}},
		},
	}

	for i, tc := range testCases {
		d, err := p.Parse([]byte(tc.body))
		if err != nil {
			t.Fatal(i, ":", err)
		}
		args := d.Args()
		columns := d.Columns()
		if len(args) != len(columns) {
			t.Fatal(i, ":Should be equal length: ", args, columns)
		}
		v := sliceToMap(columns, args)
		if ptrDay, ok := tc.args["day_ptr"]; ok {
			dp, ok := v["day_ptr"].(*Weekday)
			if !ok {
				t.Fatal(i, ":Unexpected type")
			}
			if *dp != ptrDay {
				t.Fatal(i, ":Unexpected pointer day: ", *dp)
			}
			delete(v, "day_ptr")
			delete(tc.args, "day_ptr")
			if len(tc.args) == 0 {
				continue
			}
		}
		if !reflect.DeepEqual(v, tc.args) {
			t.Fatalf("%d:want %#v got %#v", i, tc.args, v)
		}
	}
}
