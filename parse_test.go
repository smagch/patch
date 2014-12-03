package patch

import (
	"errors"
	"reflect"
	"strconv"
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

var (
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

func assertParseError(t *testing.T, p *Patcher, v string) *ParseError {
	_, err := p.Unmarshal([]byte(v))
	if _, ok := err.(*ParseError); !ok {
		t.Fatal("want *ParseError: ", err)
	}
	_, err = p.Decode(strings.NewReader(v))
	pErr, ok := err.(*ParseError)
	if !ok {
		t.Fatal("want *ParseError: ", err)
	}
	return pErr
}

func TestInvalidJSON(t *testing.T) {
	type User struct {
		ID int
	}
	p := New(User{})
	testCases := []string{
		`{"ID": ""`,
		"{",
		"{}}",
	}
	for _, tc := range testCases {
		assertParseError(t, p, tc)
	}
}

func TestInvalidProperties(t *testing.T) {
	type user struct {
		JSONIgnore  int `json:"-"`
		PatchIgnore int `patch:"-"`
		ID          int
	}
	p := New(user{})

	testCases := []struct {
		body string
		key  string
	}{
		{`{"ID": 100, "JSONIgnore": 1}`, "JSONIgnore"},
		{`{"PatchIgnore": 1000}`, "PatchIgnore"},
	}

	for i, tc := range testCases {
		err := assertParseError(t, p, tc.body)
		if tc.key != err.Key {
			t.Error(i, "Unexpected Key: ", err.Key, " want ", tc.key)
		}
	}
}

func TestNewPatch(t *testing.T) {
	type user struct {
		ID          int
		Name        string     `json:"name"`
		Email       string     `json:"email" patch:"email_address"`
		Active      bool       `patch:"active"`
		Day         Weekday    `json:"day"`
		DayPtr      *Weekday   `json:"day_ptr"`
		Span        [2]Weekday `json:"span"`
		Days        []Weekday  `json:"days"`
		NullableInt *int
		Replacable  int `json:"rep" patch:"rep_str"`
	}
	p := New(user{})

	testCases := []struct {
		body   string
		keys   []string
		values []interface{}
	}{
		{
			`{"ID": 1}`,
			[]string{"ID"},
			[]interface{}{1},
		},
		{
			`{"name": "sunday"}`,
			[]string{"name"},
			[]interface{}{"sunday"},
		},
		{
			// patch tag name doesn't override json property name
			`{"name": "foo", "Active": true}`,
			[]string{"name", "active"},
			[]interface{}{"foo", true},
		},
		{
			// should be in the order of struct field index
			`{"email": "@.org", "Active": false,"ID": 100, "name": "name"}`,
			[]string{"ID", "name", "email_address", "active"},
			[]interface{}{100, "name", "@.org", false},
		},
		{
			`{"NullableInt": null}`,
			[]string{"NullableInt"},
			[]interface{}{(*int)(nil)},
		},
		{
			`{"day":"wednesday"}`,
			[]string{"day"},
			[]interface{}{Wednesday},
		},
		{
			`{"day_ptr":"friday"}`,
			[]string{"day_ptr"},
			[]interface{}{(&Friday)},
		},
		{
			`{"span": ["sunday", "wednesday"]}`,
			[]string{"span"},
			[]interface{}{[2]Weekday{Sunday, Wednesday}},
		},
		{
			`{"days": ["sunday", "monday", "friday", "saturday"]}`,
			[]string{"days"},
			[]interface{}{[]Weekday{Sunday, Monday, Friday, Saturday}},
		},
		{
			`{"rep": 101}`,
			[]string{"rep_str"},
			[]interface{}{"101"},
		},
	}

	replaceInt := func(f Fields) {
		if rep, ok := f.Get("rep_str"); ok {
			f.Set("rep_str", strconv.Itoa(rep.(int)))
		}
	}

	for i, tc := range testCases {
		f, err := p.Unmarshal([]byte(tc.body))
		if err != nil {
			t.Fatal(i, ":", err)
		}
		replaceInt(f)

		keys := f.Keys()
		values := f.Values()
		if len(keys) != len(values) {
			t.Fatal(i, ":Should be equal length: ", keys, values)
		}
		if !reflect.DeepEqual(keys, tc.keys) {
			t.Fatalf("%d:want %v, got %v", i, tc.keys, keys)
		}
		if !reflect.DeepEqual(values, tc.values) {
			t.Fatalf("%d:want %#v got %#v", i, tc.values, values)
		}

		f2, err := p.Decode(strings.NewReader(tc.body))
		if err != nil {
			t.Fatal(i, ":", err)
		}
		replaceInt(f2)
		if !reflect.DeepEqual(f, f2) {
			t.Fatal("should deep equal: ", f, f2)
		}
	}
}
