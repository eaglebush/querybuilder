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

	dhl "github.com/NarsilWorks-Inc/datahelperlite"
	cfg "github.com/eaglebush/config"
	ssd "github.com/shopspring/decimal"
)

type Command uint8
type Sort uint8
type Limit uint8

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
	ErrNoTableSpecified  = errors.New("table or view was not specified")
	ErrNoColumnSpecified = errors.New("no columns were specified")
)

// ValueOption options for adding values
type ValueOption struct {
	SQLString   bool // Sets if the value is an SQL string. When true, this value is enclosed by the database client in single quotes to represent as string
	Default     any  // When set to non-nil, this is the default value when the value encounters a nil
	MatchToNull any  // When the primary value matches with this value, the resulting value will be set to NULL
}

type QueryColumn struct {
	Name   string // name of the column
	Length int    // length of the column
}

type queryValue struct {
	column      string // Name of the column
	value       any    // value of the column
	defvalue    any    // default value
	matchtonull any    // when primary value is matched by this value, it will set the value to NULL
	sqlstring   bool   // indicates if the value is an SQL string
	skip        bool   // skip this query value
	forcenull   bool   // forced to null
}

type queryFilter struct {
	expression    string // Column name or expression of the filter
	value         any    // Value of the filter if the expression is a column name
	containsvalue bool   // indicates that the filter has a separate value, not a filter expression
}

type querySort struct {
	column string
	order  Sort
}

// QueryBuilder is a structure to build SQL queries
type QueryBuilder struct {
	TableName              string                                                      // Table or view name of the query
	CommandType            Command                                                     // Command type
	Distinct               bool                                                        // Sets the option to return distinct values
	Columns                []QueryColumn                                               // Columns of the query
	Values                 []queryValue                                                // Values of the columns
	Order                  []querySort                                                 // Order by columns
	Group                  []string                                                    // Group by columns
	Filter                 []queryFilter                                               // Query filter
	StringEnclosingChar    string                                                      // Gets or sets the character that encloses a string in the query
	StringEscapeChar       string                                                      // Gets or Sets the character that escapes a reserved character such as the character that encloses a s string
	ReservedWordEscapeChar string                                                      // Reserved word escape	chars. For escaping with different opening and closing characters, just set to both. Example. `[]` for SQL server
	ParameterChar          string                                                      // Gets or sets the character placeholder for prepared statements
	ParameterInSequence    bool                                                        // Sets of the placeholders will be generated as a sequence of placeholder. Example, for SQL Server, @p0, @p1 @p2
	SkipNilWriteColumn     bool                                                        // Sets the condition that the Nil columns in an INSERT or UPDATE command would be skipped, instead of being set.
	ResultLimitPosition    Limit                                                       // The position of the row limiting statement in a query. For SQL Server, the limiting is set at the SELECT clause such as TOP 1. Later versions of SQL server supports OFFSET and FETCH.
	ResultLimit            string                                                      // The value of the row limit
	InterpolateTables      bool                                                        // When true, all table name with {} around it will be prepended with schema
	Schema                 string                                                      // When the database info is not applied, this value will be used
	ParameterOffset        int                                                         // The parameter sequence offset
	FilterFunc             func(offset int, char string, inSeq bool) ([]string, []any) // returns filter from outside functions like filterbuilder
	dbInfo                 *cfg.DatabaseInfo
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
		ParameterChar:          `?`,
		ReservedWordEscapeChar: `"`,
		ResultLimitPosition:    REAR,
		ResultLimit:            "",
		InterpolateTables:      false,
	}
}

// NewQueryBuilderWithConfig - builds a new QueryBuilder object with a table name, command type and a configuration DatabaseInfo
func NewQueryBuilderWithConfig(table string, commandType Command, config cfg.DatabaseInfo) *QueryBuilder {
	return newConfigBuilder(table, commandType, config, false)
}

// NewSelect is a shortcut builder for Select queries
func NewSelect(table string, config cfg.DatabaseInfo) *QueryBuilder {
	return newConfigBuilder(table, SELECT, config, false)
}

// NewInsert is a shortcut builder for Insert queries
func NewInsert(table string, config cfg.DatabaseInfo) *QueryBuilder {
	return newConfigBuilder(table, INSERT, config, false)
}

// NewUpdate is a shortcut builder for Update queries
func NewUpdate(table string, config cfg.DatabaseInfo, skipnull bool) *QueryBuilder {
	return newConfigBuilder(table, UPDATE, config, skipnull)
}

// NewDelete is a shortcut builder for Delete queries
func NewDelete(table string, config cfg.DatabaseInfo) *QueryBuilder {
	return newConfigBuilder(table, DELETE, config, false)
}

func newConfigBuilder(table string, commandType Command, config cfg.DatabaseInfo, skipnull bool) *QueryBuilder {
	return &QueryBuilder{
		TableName:              table,
		CommandType:            commandType,
		StringEnclosingChar:    *config.StringEnclosingChar,
		StringEscapeChar:       *config.StringEscapeChar,
		ParameterChar:          config.ParameterPlaceholder,
		ParameterInSequence:    config.ParameterInSequence,
		ResultLimitPosition:    REAR,
		ReservedWordEscapeChar: *config.ReservedWordEscapeChar,
		InterpolateTables:      *config.InterpolateTables,
		ResultLimit:            ``,
		SkipNilWriteColumn:     skipnull,
		dbInfo:                 &config,
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
func (qb *QueryBuilder) AddValue(Name string, Value any, vo *ValueOption) *QueryBuilder {
	var (
		sqlstr      bool
		defVal      any
		matchToNull any
	)
	sqlstr = true
	defVal = nil
	matchToNull = nil
	if vo != nil {
		sqlstr = vo.SQLString
		defVal = vo.Default
		matchToNull = vo.MatchToNull
	}
	return qb.setColumnValue(qb.addColumn(Name, 8000), Value, sqlstr, defVal, matchToNull)
}

// SetColumnValue - sets the column value
func (qb *QueryBuilder) SetColumnValue(Name string, Value any) *QueryBuilder {
	if qb.CommandType == DELETE {
		return qb
	}
	for i, v := range qb.Values {
		if strings.EqualFold(Name, v.column) {
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
func (qb *QueryBuilder) AddFilter(Column string, Value any) *QueryBuilder {
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
func (qb *QueryBuilder) Build() (query string, args []any, err error) {

	var sb strings.Builder

	args = make([]any, 0)

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
		if qb.Distinct {
			sb.WriteString("DISTINCT ")
		}
		if len(qb.ResultLimit) > 0 && qb.ResultLimitPosition == FRONT {
			sb.WriteString(" TOP " + qb.ResultLimit + " ")
		}
	case INSERT:
		sb.WriteString("INSERT INTO " + tbn + " (")
	case UPDATE:
		sb.WriteString("UPDATE " + tbn + " SET ")
	case DELETE:
		sb.WriteString("DELETE \rFROM " + tbn)
	}

	// build columns (with placeholder for update )
	cma := ""
	pchar := ""
	paramcnt := qb.ParameterOffset
	columncnt := 0

	for idx, v := range qb.Values {
		qb.Values[idx].forcenull = false
		isnl := isNil(v.value)
		// If value is nil, get defvalue
		if isnl && !isNil(v.defvalue) {
			v.value = v.defvalue
			isnl = false
		}
		// If matchtonull is true, column value is nil
		if !isnl && !isNil(v.matchtonull) && v.matchtonull == v.value {
			isnl = true
			qb.Values[idx].forcenull = true
			qb.Values[idx].sqlstring = true
		}
		// Skip columns to render if the SkipNilWriteColumn is true and value is nil
		qb.Values[idx].skip = qb.SkipNilWriteColumn && isnl
		switch qb.CommandType {
		case SELECT:
			sb.WriteString(cma + v.column)
			cma = ", "
			columncnt++
		case INSERT:
			if qb.Values[idx].skip && !qb.Values[idx].forcenull {
				break
			}
			sb.WriteString(cma + v.column)
			cma = ", "
			columncnt++
		case UPDATE:
			if qb.Values[idx].skip && !qb.Values[idx].forcenull {
				break
			}
			sb.WriteString(cma + v.column)
			pchar = " = "
			if isnl {
				pchar += "NULL"
			} else {
				if v.sqlstring {
					pchar += qb.ParameterChar
					if qb.ParameterInSequence {
						paramcnt++
						pchar += strconv.Itoa(paramcnt)
					}
				} else {
					switch t := v.value.(type) {
					case string:
						pchar += t
					case int:
						pchar += strconv.Itoa(t)
					case int64:
						pchar += strconv.FormatInt(t, 10)
					case bool:
						if t {
							pchar += "1"
						} else {
							pchar += "0"
						}
					case float32:
						pchar += strconv.FormatFloat(float64(t), 'E', -1, 32)
					case float64:
						pchar += strconv.FormatFloat(t, 'E', -1, 64)
					}
				}
			}
			sb.WriteString(pchar)
			cma = ", "
			columncnt++
		}
	}

	// Append table name for SELECT
	if qb.CommandType == SELECT {
		sb.WriteString(" \rFROM " + tbn)
	}

	// build value place holder for insert
	if qb.CommandType == INSERT {
		cma = ""
		pchar = ""
		inscnt := 0
		q := make([]string, columncnt)
		for _, v := range qb.Values {
			if v.skip && !v.forcenull {
				continue
			}
			pchar = "NULL"
			if !isNil(v.value) && !v.forcenull {
				if !v.sqlstring {
					pchar, _ = v.value.(string)
				} else {
					pchar = qb.ParameterChar
					if qb.ParameterInSequence {
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
	if qb.CommandType == SELECT || qb.CommandType == UPDATE || qb.CommandType == DELETE {
		cma = ""
		var tsb strings.Builder
		for _, c := range qb.Filter {
			if !isNil(c.value) {
				pchar = qb.ParameterChar
				if qb.ParameterInSequence {
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
			cma = "\r\t\t AND "
		}
		if qb.FilterFunc != nil {
			fbs, _ := qb.FilterFunc(paramcnt, qb.ParameterChar, qb.ParameterInSequence)
			if len(fbs) > 0 {
				for _, fb := range fbs {
					tsb.WriteString(cma + fb)
					cma = "\r\t\t AND "
				}
			}
		}
		if tsb.Len() > 0 {
			sb.WriteString("\r\t WHERE " + tsb.String())
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
			isNil(v.value) ||
			v.forcenull {
			continue
		}
		args = append(args, v.value)
	}

	// build filter values
	for _, v := range qb.Filter {
		if (qb.CommandType == SELECT || qb.CommandType == UPDATE || qb.CommandType == DELETE) && !isNil(v.value) {
			args = append(args, v.value)
		}
	}

	if qb.FilterFunc != nil {
		fbs, fbargs := qb.FilterFunc(paramcnt, qb.ParameterChar, qb.ParameterInSequence)
		if len(fbs) > 0 {
			args = append(args, fbargs...)
		}
	}

	query = sb.String()

	if qb.InterpolateTables {
		sch := ``
		// if there is a dbinfo, get the schema
		if qb.dbInfo != nil {
			sch = qb.dbInfo.Schema
		}
		// If there is a schema defined, it will prevail
		if qb.Schema != "" {
			sch = qb.Schema
		}
		// replace table names marked with {table}
		query = InterpolateTable(query, sch)
	}

	qb.ParameterOffset = paramcnt
	return
}

func (qb *QueryBuilder) addColumn(name string, length int) int {
	for i, v := range qb.Columns {
		if !strings.EqualFold(name, v.Name) {
			continue
		}
		return i
	}
	qb.Columns = append(qb.Columns, QueryColumn{Name: name, Length: length})
	return len(qb.Columns) - 1
}

func (qb *QueryBuilder) setColumnValue(index int, value any, sqlString bool, defValue, matchToNull any) *QueryBuilder {
	for i, v := range qb.Values {
		if !strings.EqualFold(qb.Columns[index].Name, v.column) {
			continue
		}
		qb.Values[i].sqlstring = sqlString
		qb.Values[i].defvalue = defValue
		qb.Values[i].matchtonull = matchToNull
		qb.Values[i].value = value
		return qb
	}
	qb.Values = append(qb.Values, queryValue{
		column:      qb.Columns[index].Name,
		sqlstring:   sqlString,
		defvalue:    defValue,
		matchtonull: matchToNull,
		value:       value,
	})
	return qb
}

// // Original isNil code
// func isNil(value any) bool {
// 	if value == nil {
// 		return true
// 	}
// 	if t := reflect.TypeOf(value); t == nil {
// 		return true
// 	}
// 	if v := reflect.ValueOf(value); v.IsZero() {
// 		if k := v.Kind(); k == reflect.Map ||
// 			k == reflect.Func ||
// 			k == reflect.Ptr ||
// 			k == reflect.Slice ||
// 			k == reflect.Interface {
// 			return v.IsNil()
// 		}
// 	}
// 	return false
// }

// ChatGPT optimized isNil
func isNil(value any) bool {
	if value == nil {
		return true
	}

	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
		return v.IsNil()
	}
	return false
}

// // Original realvalue code
// // converts the value to a basic interface as nil or non-nil
// func realvalue(value any) any {
// 	if isNil(value) {
// 		return nil
// 	}
// 	var ret any
// 	switch t := value.(type) {
// 	case *any:
// 		v2 := *t
// 		if v2 != nil {
// 			// we stop checking the *any here
// 			switch t2 := v2.(type) {
// 			default:
// 				ret = getv(t2)
// 			}
// 		}
// 	default:
// 		ret = getv(t)
// 	}
// 	return ret
// }

// ChatGPT refactored
// converts the value to a basic interface as nil or non-nil
func realvalue(value any) any {
	if isNil(value) {
		return nil
	}

	// Unwrap pointer to interface if applicable
	if ptr, ok := value.(*any); ok && ptr != nil {
		if v2 := *ptr; v2 != nil {
			return getv(v2)
		}
		return nil
	}

	return getv(value)
}

// // Original code
// func getv(input any) (ret any) {
// 	switch t := input.(type) {
// 	case string, int, int8, int16, int32,
// 		int64, float32, float64, time.Time, bool,
// 		byte, []byte, ssd.Decimal:
// 		ret = t
// 	case *string:
// 		ret = *t
// 	case *int:
// 		ret = *t
// 	case *int8:
// 		ret = *t
// 	case *int16:
// 		ret = *t
// 	case *int32:
// 		ret = *t
// 	case *int64:
// 		ret = *t
// 	case *float32:
// 		ret = *t
// 	case *float64:
// 		ret = *t
// 	case *time.Time:
// 		ret = *t
// 	case *bool:
// 		ret = *t
// 	case *byte:
// 		ret = *t
// 	case *[]byte:
// 		ret = *t
// 	case *ssd.Decimal:
// 		ret = *t
// 	case dhl.VarChar, dhl.VarCharMax, dhl.NVarCharMax:
// 		ret = t
// 	}
// 	return
// }

// ChatGPT refactored code
func getv(input any) any {
	switch t := input.(type) {
	case string, int, int8, int16, int32,
		int64, float32, float64, time.Time, bool,
		byte, []byte, ssd.Decimal,
		dhl.VarChar, dhl.VarCharMax, dhl.NVarCharMax:
		return t
	case *string:
		if t != nil {
			return *t
		}
	case *int:
		if t != nil {
			return *t
		}
	case *int8:
		if t != nil {
			return *t
		}
	case *int16:
		if t != nil {
			return *t
		}
	case *int32:
		if t != nil {
			return *t
		}
	case *int64:
		if t != nil {
			return *t
		}
	case *float32:
		if t != nil {
			return *t
		}
	case *float64:
		if t != nil {
			return *t
		}
	case *time.Time:
		if t != nil {
			return *t
		}
	case *bool:
		if t != nil {
			return *t
		}
	case *byte:
		if t != nil {
			return *t
		}
	case *[]byte:
		if t != nil {
			return *t
		}
	case *ssd.Decimal:
		if t != nil {
			return *t
		}
	}
	return nil
}

// ParseReserveWordsChars always returns two-element array of opening and closing escape chars
func ParseReserveWordsChars(ec string) []string {
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
