// Package QueryBuilder
//
// Builds SQL query based on the inputs
//
// TODO:
//	- figure out how to cache queries to save iteration of columns and values

package querybuilder

import (
	"errors"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	cfg "github.com/eaglebush/config"
)

// Command - the type of command to perform
type Command int

// Sort - sort type
type Sort int

// Limit is the position of the row limiting command
type Limit int

// CommandType enum
const (
	SELECT Command = 0 // Select record type
	INSERT Command = 1 // Insert record type
	UPDATE Command = 2 // Update record type
	DELETE Command = 3 // Delete record type
)

// Sort enum
const (
	ASC  Sort = 0
	DESC Sort = 1
)

// Limit enum
const (
	FRONT Limit = 0
	REAR  Limit = 1
)

// errors
var (
	ErrNoTableSpecified  = errors.New("Table or view was not specified")
	ErrNoColumnSpecified = errors.New("No columns were specified")
)

// ValueOption options for adding values
type ValueOption struct {
	SQLString   bool        // Sets if the value is an SQL string. When true, this value is enclosed by the database client in single quotes to represent as string
	Default     interface{} // When set to non-nil, this is the default value when the value encounters a nil
	MatchToNull interface{} // When the primary value matches with this value, the resulting value will be set to NULL
}

type queryColumn struct {
	name   string // name of the column
	length int    // length of the column
}

type queryValue struct {
	column      string      // Name of the column
	value       interface{} // value of the column
	defvalue    interface{} // default value
	matchtonull interface{} // when primary value is matched by this value, it will set the value to NULL
	sqlstring   bool        // indicates if the value is an SQL string
	skip        bool        // skip this query value
}

type queryFilter struct {
	expression    string      // Column name or expression of the filter
	value         interface{} // Value of the filter if the expression is a column name
	containsvalue bool        // indicates that the filter has a separate value, not a filter expression
}

type querySort struct {
	column string
	order  Sort
}

// QueryBuilder - a class to build SQL queries
type QueryBuilder struct {
	TableName                   string        // Table or view name of the query
	CommandType                 Command       // Command type
	Columns                     []queryColumn // Columns of the query
	Values                      []queryValue  // Values of the columns
	Order                       []querySort   // Order by columns
	Group                       []string      // Group by columns
	Filter                      []queryFilter // Query filter
	StringEnclosingChar         string        // Gets or sets the character that encloses a string in the query
	StringEscapeChar            string        // Gets or Sets the character that escapes a reserved character such as the character that encloses a s string
	ReservedWordEscapeChar      string        // Reserved word escape	chars. For escaping with different opening and closing characters, just set to both. Example. `[]` for SQL server
	PreparedStatementChar       string        // Gets or sets the character placeholder for prepared statements
	PreparedStatementInSequence bool          // Sets of the placeholders will be generated as a sequence of placeholder. Example, for SQL Server, @p0, @p1 @p2
	SkipNilWriteColumn          bool          // Sets the condition that the Nil columns in an INSERT or UPDATE command would be skipped, instead of being set.
	ResultLimitPosition         Limit         // The position of the row limiting statement in a query. For SQL Server, the limiting is set at the SELECT clause such as TOP 1. Later versions of SQL server supports OFFSET and FETCH.
	ResultLimit                 string        // The value of the row limit
	InterpolateTables           bool          // When true, all table name with {} around it will be prepended with schema
	Schema                      string        // When the database info is not applied, this value will be used
	dbinfo                      *cfg.DatabaseInfo
}

// NewQueryBuilder - builds a new QueryBuilder object
func NewQueryBuilder(table string) *QueryBuilder {

	qb := NewQueryBuilderBare()
	qb.TableName = table

	return qb
}

// NewQueryBuilderWithCommandType - builds a new QueryBuilder object with table name and command type
func NewQueryBuilderWithCommandType(table string, commandType Command) *QueryBuilder {

	qb := NewQueryBuilderBare()
	qb.TableName = table
	qb.CommandType = commandType

	return qb
}

// NewQueryBuilderBare - builds a new QueryBuilder object without a table name
func NewQueryBuilderBare() *QueryBuilder {
	return &QueryBuilder{
		StringEnclosingChar:    `'`,
		StringEscapeChar:       `\`,
		PreparedStatementChar:  `?`,
		ReservedWordEscapeChar: `"`,
		ResultLimitPosition:    REAR,
		ResultLimit:            "",
		InterpolateTables:      false,
	}
}

// NewQueryBuilderWithConfig - builds a new QueryBuilder object with a table name, command type and a configuration DatabaseInfo
func NewQueryBuilderWithConfig(table string, commandType Command, config cfg.DatabaseInfo) *QueryBuilder {

	return &QueryBuilder{
		TableName:                   table,
		CommandType:                 commandType,
		StringEnclosingChar:         *config.StringEnclosingChar,
		StringEscapeChar:            *config.StringEscapeChar,
		PreparedStatementChar:       config.ParameterPlaceholder,
		PreparedStatementInSequence: config.ParameterInSequence,
		ResultLimitPosition:         REAR,
		ReservedWordEscapeChar:      *config.ReservedWordEscapeChar,
		InterpolateTables:           *config.InterpolateTables,
		ResultLimit:                 ``,
		dbinfo:                      &config,
	}
}

// AddColumn - adds a column
func (qb *QueryBuilder) AddColumn(Name string) *QueryBuilder {

	if qb.CommandType == DELETE {
		return qb
	}

	return qb.setColumnValue(qb.addColumn(Name, 255), nil, true, nil, nil)
}

// AddColumnFixed - adds a column with specified length
func (qb *QueryBuilder) AddColumnFixed(Name string, Length int) *QueryBuilder {

	if qb.CommandType == DELETE {
		return qb
	}

	return qb.setColumnValue(qb.addColumn(Name, Length), nil, true, nil, nil)
}

// AddValue adds a value enclosed with string quotes when the CommandType is INSERT or UPDATE upon building
func (qb *QueryBuilder) AddValue(Name string, Value interface{}, vo *ValueOption) *QueryBuilder {

	var (
		sqlstr      bool
		defval      interface{}
		matchtonull interface{}
	)

	sqlstr = true
	defval = nil
	matchtonull = nil

	if vo != nil {
		sqlstr = vo.SQLString
		defval = vo.Default
		matchtonull = vo.MatchToNull
	}

	return qb.setColumnValue(qb.addColumn(Name, 8000), Value, sqlstr, defval, matchtonull)
}

// SetColumnValue - sets the column value
func (qb *QueryBuilder) SetColumnValue(Name string, Value interface{}) *QueryBuilder {

	if qb.CommandType == DELETE {
		return qb
	}

	c := strings.ToLower(Name)
	for i, v := range qb.Values {

		if c != strings.ToLower(v.column) {
			continue
		}

		return qb.setColumnValue(i, Value, true, nil, nil)
	}

	return qb
}

// Escape a string value to prevent unescaped errors
func (qb *QueryBuilder) Escape(Value string) string {

	if len(Value) > 0 {
		return strings.ReplaceAll(Value, qb.StringEnclosingChar, qb.StringEscapeChar+qb.StringEnclosingChar)
	}

	return Value
}

// AddFilter adds a filter with value.
func (qb *QueryBuilder) AddFilter(Column string, Value interface{}) *QueryBuilder {

	qb.Filter = append(qb.Filter, queryFilter{
		expression: Column,
		value:      Value,
	})

	return qb
}

// AddFilterExp adds a specific filter expression that could not be done with AddFilter
func (qb *QueryBuilder) AddFilterExp(Expression string) *QueryBuilder {

	qb.Filter = append(qb.Filter, queryFilter{
		expression:    Expression,
		value:         nil,
		containsvalue: true,
	})

	return qb
}

// AddOrder - adds a column to order by into the QueryBuilder for both BuildString() and BuildDataHelper() function.
func (qb *QueryBuilder) AddOrder(Column string, Order Sort) *QueryBuilder {

	qb.Order = append(qb.Order, querySort{column: Column, order: Order})

	return qb
}

// AddGroup - adds a group by clause
func (qb *QueryBuilder) AddGroup(Group string) *QueryBuilder {

	qb.Group = append(qb.Group, Group)

	return qb
}

// Build an SQL string with corresponding values
func (qb *QueryBuilder) Build() (query string, args []interface{}, err error) {

	var sb strings.Builder

	args = make([]interface{}, 0)

	if qb.TableName == "" {
		return "", nil, ErrNoTableSpecified
	}

	if len(qb.Columns) == 0 && qb.CommandType != DELETE {
		return "", nil, ErrNoColumnSpecified
	}

	// get real values of qb.Values and set them back
	for i := range qb.Values {
		qb.Values[i].value = realvalue(qb.Values[i].value)
		qb.Values[i].defvalue = realvalue(qb.Values[i].defvalue)
		qb.Values[i].matchtonull = realvalue(qb.Values[i].matchtonull)
	}

	// get real values of filter values and set them back
	for i := range qb.Filter {
		qb.Filter[i].value = realvalue(qb.Filter[i].value)
	}

	// Auto attach schema
	tbn := qb.TableName

	switch qb.CommandType {
	case SELECT:
		sb.WriteString("SELECT ")
		if len(qb.ResultLimit) > 0 && qb.ResultLimitPosition == FRONT {
			sb.WriteString(" TOP " + qb.ResultLimit + " ")
		}
	case INSERT:
		sb.WriteString("INSERT INTO " + tbn + " (")
	case UPDATE:
		sb.WriteString("UPDATE " + tbn + " SET ")
	case DELETE:
		sb.WriteString("DELETE FROM " + tbn)
	}

	// build columns (with placeholder for update )
	cma := ""
	pchar := ""
	paramcnt := 0
	columncnt := 0

	for idx, v := range qb.Values {

		isnl := isnil(v.value)

		// If value is nil, get defvalue
		if isnl && !isnil(v.defvalue) {
			v.value = v.defvalue
			isnl = false
		}

		// If matchtonull is true, column value is nil
		if !isnl && !isnil(v.matchtonull) && v.matchtonull == v.value {
			isnl = true
		}

		// Skip columns to render if the SkipNilWriteColumn is true and value is nil
		qb.Values[idx].skip = qb.SkipNilWriteColumn && isnl

		switch qb.CommandType {
		case SELECT:
			sb.WriteString(cma + v.column)
			cma = ", "
			columncnt++
		case INSERT:

			if qb.Values[idx].skip {
				break
			}

			sb.WriteString(cma + v.column)
			cma = ", "
			columncnt++
		case UPDATE:

			if qb.Values[idx].skip {
				break
			}

			sb.WriteString(cma + v.column)
			pchar = " = "

			if isnl {
				pchar += "NULL"
			} else {
				if v.sqlstring {
					pchar += qb.PreparedStatementChar
					if qb.PreparedStatementInSequence {
						paramcnt++
						pchar += strconv.Itoa(paramcnt)
					}
				} else {
					pchar += v.value.(string)
				}
			}

			sb.WriteString(pchar)
			cma = ", "
			columncnt++
		}
	}

	// Append table name for SELECT
	if qb.CommandType == SELECT {
		sb.WriteString(" FROM " + tbn)
	}

	// build value place holder for insert
	if qb.CommandType == INSERT {

		cma = ""
		pchar = ""
		inscnt := 0
		q := make([]string, columncnt)

		for _, v := range qb.Values {

			if v.skip {
				continue
			}

			pchar = "NULL"

			if !isnil(v.value) {

				if !v.sqlstring {
					pchar, _ = v.value.(string)
				} else {
					pchar = qb.PreparedStatementChar
					if qb.PreparedStatementInSequence {
						paramcnt++
						pchar += strconv.Itoa(paramcnt)
					}
				}
			}

			q[inscnt] = cma + pchar

			cma = ","
			inscnt++

		}

		sb.WriteString(") VALUES (" + strings.Join(q, "") + ")")
	}

	// build filter parameters for SELECT, UPDATE and DELETE
	if len(qb.Filter) > 0 && (qb.CommandType == SELECT || qb.CommandType == UPDATE || qb.CommandType == DELETE) {

		cma = ""
		var tsb strings.Builder

		for _, c := range qb.Filter {

			if !isnil(c.value) {

				pchar = qb.PreparedStatementChar
				if qb.PreparedStatementInSequence {
					paramcnt++
					pchar += strconv.Itoa(paramcnt)
				}
				tsb.WriteString(cma + c.expression + " = " + pchar)
			} else {

				tsb.WriteString(cma + c.expression)
				if !c.containsvalue {
					tsb.WriteString(" IS NULL")
				}
			}

			cma = " AND "

		}

		if tsb.Len() > 0 {
			sb.WriteString(" WHERE " + tsb.String())
		}
	}

	// build order bys
	if len(qb.Order) > 0 {

		sb.WriteString(" ORDER BY ")
		cma = ""

		for _, v := range qb.Order {

			sb.WriteString(cma + v.column)
			if v.order == ASC {
				sb.WriteString(" ASC")
			} else {
				sb.WriteString(" DESC")
			}

			cma = ", "
		}
	}

	// build group by
	if len(qb.Group) > 0 {
		sb.WriteString(" GROUP BY " + strings.Join(qb.Group, ", "))
	}

	if len(qb.ResultLimit) > 0 && qb.ResultLimitPosition == REAR {
		sb.WriteString(" LIMIT " + qb.ResultLimit)
	}

	sb.WriteString(";")

	// build values
	for _, v := range qb.Values {

		if v.skip ||
			!v.sqlstring ||
			!(qb.CommandType == INSERT || qb.CommandType == UPDATE) ||
			isnil(v.value) {

			continue
		}

		args = append(args, v.value)
	}

	// build filter values
	for _, v := range qb.Filter {
		if (qb.CommandType == SELECT || qb.CommandType == UPDATE || qb.CommandType == DELETE) && !isnil(v.value) {
			args = append(args, v.value)
		}
	}

	query = sb.String()

	if qb.InterpolateTables {
		sch := ``

		// if there is a dbinfo, get the schema
		if qb.dbinfo != nil {
			sch = qb.dbinfo.Schema
		}

		// If there is a schema defined, it will prevail
		if qb.Schema != "" {
			sch = qb.Schema
		}

		// replace table names marked with {table}
		query = InterpolateTable(query, sch)
	}

	return
}

func (qb *QueryBuilder) addColumn(name string, length int) int {

	c := strings.ToLower(name)
	for i, v := range qb.Columns {

		if c != strings.ToLower(v.name) {
			continue
		}

		return i
	}

	qb.Columns = append(qb.Columns, queryColumn{name: name, length: length})

	return len(qb.Columns) - 1
}

func (qb *QueryBuilder) setColumnValue(index int, value interface{}, sqlString bool, defValue interface{}, matchToNull interface{}) *QueryBuilder {

	c := strings.ToLower(qb.Columns[index].name)

	for i, v := range qb.Values {

		if c != strings.ToLower(v.column) {
			continue
		}

		qb.Values[i].sqlstring = sqlString
		qb.Values[i].defvalue = defValue
		qb.Values[i].matchtonull = matchToNull
		qb.Values[i].value = value

		return qb
	}

	qb.Values = append(qb.Values, queryValue{
		column:      qb.Columns[index].name,
		sqlstring:   sqlString,
		defvalue:    defValue,
		matchtonull: matchToNull,
		value:       value,
	})

	return qb
}

func isnil(value interface{}) bool {

	isnil := false

	// Check if the value is nil
	isnil = value == nil

	// Check if the type of the value isn't nil
	if !isnil {
		if t := reflect.TypeOf(value); t == nil {
			isnil = true
		}
	}

	// Check if the underlying value is nil
	if !isnil {
		if tv := reflect.ValueOf(value); tv.IsZero() {
			if k := tv.Kind(); k == reflect.Map ||
				k == reflect.Func ||
				k == reflect.Ptr ||
				k == reflect.Slice ||
				k == reflect.Interface {

				isnil = tv.IsNil()
			}
		}
	}

	return isnil
}

// converts the value to a basic interface as nil or non-nil
func realvalue(value interface{}) interface{} {

	if isnil(value) {
		return nil
	}

	var ret interface{}

	switch t := value.(type) {
	case *interface{}:
		v2 := *t
		if v2 != nil {
			// we stop checking the *interface{} here
			switch t2 := v2.(type) {
			default:
				ret = getv(t2)
			}
		}
	default:
		ret = getv(t)
	}

	return ret
}

func getv(input interface{}) (ret interface{}) {

	switch t := input.(type) {
	case string, int, int8, int16, int32, int64, float32, float64, time.Time, bool, byte, []byte:
		ret = t
	case *string:
		ret = *t
	case *int:
		ret = *t
	case *int8:
		ret = *t
	case *int16:
		ret = *t
	case *int32:
		ret = *t
	case *int64:
		ret = *t
	case *float32:
		ret = *t
	case *float64:
		ret = *t
	case *time.Time:
		ret = *t
	case *bool:
		ret = *t
	case *byte:
		ret = *t
	case *[]byte:
		ret = *t
	}

	return
}

// parseReserveWordsChars always returns two-element array of opening and closing escape chars
func parseReserveWordsChars(ec string) []string {

	if len(ec) == 1 {
		return []string{ec, ec}
	}

	if len(ec) >= 2 {
		return []string{ec[0:1], ec[1:2]}
	}

	return []string{`"`, `"`} // default is double quotes
}

// InterpolateTable - interpolate the tables specified with curly braces {} with a schema
func InterpolateTable(sql string, schema string) string {

	if schema != "" {
		schema = schema + `.`
	}

	re := regexp.MustCompile(`\{([a-zA-Z0-9\[\]\"\_\-]*)\}`)
	return re.ReplaceAllString(sql, schema+`$1`)
}
