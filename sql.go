package patch

import (
	"bytes"
	"strconv"
)

// mergeArgs concat given three slice of interfaces.
func mergeArgs(pre, middle, post []interface{}) []interface{} {
	lpre, lmid := len(pre), len(middle)
	args := make([]interface{}, lpre+lmid+len(post))
	copy(args[0:lpre], pre)
	copy(args[lpre:lpre+lmid], middle)
	copy(args[lpre+lmid:], post)
	return args
}

// SQL provides a way to build SQL statement.
type SQL struct {
	Fields
	postArgs []interface{}
	preArgs  []interface{}
}

// Prepend SQL arguments in case the query statement has placeholders before
// patch keys and values.
// It'd be rarely used with Postgres driver.
func (s *SQL) Prepend(args ...interface{}) {
	s.preArgs = append(s.preArgs, args...)
}

// Query returns pieace of SQL statement (key1=?,key2=?) and arguments appending
// the given SQL arguments.
func (s *SQL) Query(appends ...interface{}) (query string, args []interface{}) {
	s.postArgs = append(s.postArgs, appends...)
	var buf bytes.Buffer
	values := make([]interface{}, len(s.Fields))
	for i, f := range s.Fields {
		if i != 0 {
			buf.WriteString(",")
		}
		buf.WriteString(f.Key)
		buf.WriteString("=?")
		values[i] = f.Value
	}
	return buf.String(), mergeArgs(s.preArgs, values, s.postArgs)
}

// QueryPostgres returns pieace of SQL statement (key1=$1,key2=$2) and arguments
// appending the given SQL arguments.
func (s *SQL) QueryPostgres(appends ...interface{}) (query string, args []interface{}) {
	s.postArgs = append(s.postArgs, appends...)
	var buf bytes.Buffer
	offset := len(s.preArgs) + len(s.postArgs)
	values := make([]interface{}, len(s.Fields))
	for i, f := range s.Fields {
		if i != 0 {
			buf.WriteString(",")
		}
		buf.WriteString(f.Key)
		buf.WriteString("=$")
		buf.WriteString(strconv.Itoa(offset + i + 1))
		values[i] = f.Value
	}
	return buf.String(), mergeArgs(s.preArgs, s.postArgs, values)
}
