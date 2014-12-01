package patch

import (
	"bytes"
	"strconv"
	"strings"
	"text/template"
)

// Driver is an interface that abstracts the difference of SQL syntax of prepared
// statement.
type Driver interface {
	GetPlaceHolders(offset int, length int) []string
	MergeArgs(pre, patchArgs, post []interface{}) []interface{}
}

// Postgres is the default postgres driver
var Postgres = &PostgresDriver{"$"}

type PostgresDriver struct {
	sign string
}

// GetPlaceHolders implements Driver interface.
func (d *PostgresDriver) GetPlaceHolders(offset int, length int) []string {
	holders := make([]string, length)
	for i := 0; i < length; i++ {
		holders[i] = string(d.sign) + strconv.Itoa(offset+i+1)
	}
	return holders
}

// MergeArgs implements Driver interface.
func (d *PostgresDriver) MergeArgs(pre, patchArgs, post []interface{}) []interface{} {
	lpre, lenArg, lpos := len(pre), len(patchArgs), len(post)
	args := make([]interface{}, lpre+lenArg+lpos)
	copy(args[0:lpre], pre)
	copy(args[lpre:lpre+lpos], post)
	copy(args[lpre+lpos:], patchArgs)
	return args
}

// type Driver represents a parsed JSON inputs.
type Data struct {
	columns  []string
	driver   Driver
	args     []interface{}
	preArgs  []interface{}
	postArgs []interface{}
}

// TemplateData is
type TemplateData struct {
	// column includes comma-joined column names. e.g. id,user_id,updated_at
	Column string
	// Bind includes comma-joined bind names.
	// $1,$2,$3 for postgres
	// ?,?,? for mysql
	Bind string
}

func (d *Data) Len() int {
	return len(d.columns)
}

func (d *Data) Has(name string) bool {
	for _, n := range d.columns {
		if name == n {
			return true
		}
	}
	return false
}

func (d *Data) getIndex(name string) (interface{}, int) {
	for i, n := range d.columns {
		if name == n {
			return d.args[i], i
		}
	}
	return nil, -1
}

func (d *Data) Remove(name string) interface{} {
	v, index := d.getIndex(name)
	if index == -1 {
		return nil
	}
	d.columns = append(d.columns[:index], d.columns[index+1:]...)
	d.args = append(d.args[:index], d.args[index+1:]...)
	return v
}

func (d *Data) Get(name string) interface{} {
	v, _ := d.getIndex(name)
	return v
}

// Query executes the given template with the args
func (d *Data) Query(tmpl *template.Template, args ...interface{}) (string, error) {
	d.postArgs = args
	var b bytes.Buffer
	err := tmpl.Execute(&b, d.TemplateData())
	return b.String(), err
}

// Args returns arguments
func (d *Data) Args() []interface{} {
	return d.driver.MergeArgs(d.preArgs, d.args, d.postArgs)
}

// Columns returns column names
func (d *Data) Columns() []string {
	return d.columns[:]
}

func (d *Data) TemplateData() TemplateData {
	offset := len(d.preArgs) + len(d.postArgs)
	placeholders := d.driver.GetPlaceHolders(offset, len(d.columns))
	return TemplateData{
		Column: strings.Join(d.columns, ","),
		Bind:   strings.Join(placeholders, ","),
	}
}
