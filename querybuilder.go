// Package QueryBuilder
//
// Builds SQL query for both prepared and literal SQL string.
// - BuildString() function - builds literal SQL string together with the supplied values. The functions tagged for BuildString() should be used for automatic formatting
// - BuildDataHelper() function - builds a prepared command query and outputs an array of interface objects for arguments. The functions tagged for BuildDataHelper should be used.

package querybuilder

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	cfg "github.com/eaglebush/config"
)

//CommandType - the type of command to perform
type CommandType int

//QuerySortDirection - sort type
type QuerySortDirection int

//ResultLimitPosition -gets/sets the position of the row limiting command
type ResultLimitPosition int

//CommandType enum
const (
	SELECT CommandType = 0 // Select record type
	INSERT CommandType = 1 // Insert record type
	UPDATE CommandType = 2 // Update record type
	DELETE CommandType = 3 // Delete record type
)

//SortDirection enum
const (
	ASC  QuerySortDirection = 0
	DESC QuerySortDirection = 1
)

//ResultLimitPosition enum
const (
	FRONT ResultLimitPosition = 0
	REAR  ResultLimitPosition = 1
)

type queryColumn struct {
	ColumnName string
	Length     int
}

//QueryValue struct
type queryValue struct {
	ColumnName      string
	Value           interface{}
	DefaultValue    interface{}
	NullDetectValue interface{}
	IsDBString      bool
	skip            bool
}

type queryFilter struct {
	ColumnNameOrExpression string
	Value                  interface{}
	IsDBString             bool
	FilterContainsValue    bool
}

//QuerySort - sort information
type querySort struct {
	ColumnName string
	Order      QuerySortDirection
}

// QueryBuilder - a class to build SQL queries
type QueryBuilder struct {
	TableName                   string              // Table name of the query
	CommandType                 CommandType         // Command type
	Columns                     []queryColumn       // Columns of the query
	Values                      []queryValue        // Values of the columns
	Order                       []querySort         // Order by columns
	Group                       []string            // Group by columns
	Filter                      []queryFilter       // Query filter
	StringEnclosingChar         string              // Gets or sets the character that encloses a string in the query
	StringEscapeChar            string              // Gets or Sets the character that escapes a reserved character such as the character that encloses a s string
	ReservedWordEscapeChar      string              // Reserved word escape	chars. For escaping with different opening and closing characters, just set to both. Example. `[]` for SQL server
	PreparedStatementChar       string              // Gets or sets the character placeholder for prepared statements
	PreparedStatementInSequence bool                // Sets of the placeholders will be generated as a sequence of placeholder. Example, for SQL Server, @p0, @p1 @p2
	SkipNilWriteColumn          bool                // Sets the condition that the Nil columns in an INSERT or UPDATE command would be skipped, instead of being set.
	ResultLimitPosition         ResultLimitPosition // The position of the row limiting statement in a query. For SQL Server, the limiting is set at the SELECT clause such as TOP 1. Later versions of SQL server supports OFFSET and FETCH.
	ResultLimit                 string              // The value of the row limit
	InterpolateTables           bool                // When true, all table name with {} around it will be prepended with schema
	Schema                      string              // When the database info is not applied, this value will be used
	dbinfo                      *cfg.DatabaseInfo
}

// NewQueryBuilder - builds a new QueryBuilder object
func NewQueryBuilder(table string) *QueryBuilder {
	return &QueryBuilder{
		TableName:              table,
		StringEnclosingChar:    `'`,
		StringEscapeChar:       `\`,
		PreparedStatementChar:  `?`,
		ReservedWordEscapeChar: `"`,
		ResultLimitPosition:    REAR,
		ResultLimit:            "",
	}
}

// NewQueryBuilderWithCommandType - builds a new QueryBuilder object with table name and command type
func NewQueryBuilderWithCommandType(table string, commandType CommandType) *QueryBuilder {
	return &QueryBuilder{
		TableName:              table,
		CommandType:            commandType,
		StringEnclosingChar:    `'`,
		StringEscapeChar:       `\`,
		PreparedStatementChar:  `?`,
		ReservedWordEscapeChar: `"`,
		ResultLimitPosition:    REAR,
		ResultLimit:            "",
	}
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
	}
}

// NewQueryBuilderWithConfig - builds a new QueryBuilder object with a table name, command type and a configuration DatabaseInfo
func NewQueryBuilderWithConfig(table string, commandType CommandType, config cfg.DatabaseInfo) *QueryBuilder {
	return &QueryBuilder{
		TableName:                   table,
		CommandType:                 commandType,
		StringEnclosingChar:         `'`,
		StringEscapeChar:            `\`,
		PreparedStatementChar:       config.ParameterPlaceholder,
		PreparedStatementInSequence: config.ParameterInSequence,
		ResultLimitPosition:         REAR,
		ReservedWordEscapeChar:      config.ReservedWordEscapeChar,
		ResultLimit:                 "",
		dbinfo:                      &config,
	}
}

// AddColumn - adds a column into the QueryBuilder
func (qb *QueryBuilder) AddColumn(ColumnName string) *QueryBuilder {

	if qb.CommandType != DELETE {
		ci := qb.addColumn(ColumnName, 255) //only allows non-DELETE statements
		qb.setColumnValue(ci, nil, true, nil, nil)
	}

	return qb
}

// AddColumnWithLength - adds a column with specified length into the QueryBuilder
func (qb *QueryBuilder) AddColumnWithLength(ColumnName string, Length int) *QueryBuilder {
	if qb.CommandType != DELETE {
		ci := qb.addColumn(ColumnName, Length) //only allows non-DELETE statements
		qb.setColumnValue(ci, nil, true, nil, nil)
	}

	return qb
}

//SetColumnValue - sets the column value
func (qb *QueryBuilder) SetColumnValue(ColumnName string, Value interface{}, IsSQLFunction bool) *QueryBuilder {
	//only allows non-DELETE statements
	if qb.CommandType != DELETE {
		idx := -1
		c := strings.ToLower(ColumnName)
		for i, v := range qb.Values {
			if c == strings.ToLower(v.ColumnName) {
				idx = i
				break
			}
		}

		if idx != -1 {
			qb.setColumnValue(idx, Value, true, nil, nil)
		}
	}

	return qb
}

//AddColumnValue - adds a column and a value. The value is enclosed with string quotes when the CommandType is INSERT or UPDATE
func (qb *QueryBuilder) AddColumnValue(ColumnName string, Value interface{}) *QueryBuilder {
	ci := qb.addColumn(ColumnName, 255)
	qb.setColumnValue(ci, Value, true, nil, nil)

	return qb
}

//AddColumnNonStringValue - adds a column and a value. The value is not enclosed in a string when the CommandType is INSERT or UPDATE
func (qb *QueryBuilder) AddColumnNonStringValue(ColumnName string, Value interface{}) *QueryBuilder {
	ci := qb.addColumn(ColumnName, 255)
	qb.setColumnValue(ci, Value, false, nil, nil)

	return qb
}

//AddColumnValueWithDefault - adds a column and a value with default value for BuildString() function. The value is enclosed with string quotes when the CommandType is INSERT or UPDATE
func (qb *QueryBuilder) AddColumnValueWithDefault(ColumnName string, Value interface{}, Default interface{}) *QueryBuilder {
	ci := qb.addColumn(ColumnName, 255)
	qb.setColumnValue(ci, Value, true, Default, nil)

	return qb
}

//AddColumnNonStringValueWithDefault - adds a column and a value with default value for BuildString() function. The value is not enclosed in a string when the CommandType is INSERT or UPDATE
func (qb *QueryBuilder) AddColumnNonStringValueWithDefault(ColumnName string, Value interface{}, Default interface{}) *QueryBuilder {
	ci := qb.addColumn(ColumnName, 255)
	qb.setColumnValue(ci, Value, false, Default, nil)

	return qb
}

//AddColumnValueNull - adds a column and a value with null detection. The value is enclosed with string quotes when the CommandType is INSERT or UPDATE
func (qb *QueryBuilder) AddColumnValueNull(ColumnName string, Value interface{}, NullDetectValue interface{}) *QueryBuilder {
	ci := qb.addColumn(ColumnName, 255)
	qb.setColumnValue(ci, Value, true, nil, NullDetectValue)

	return qb
}

//AddColumnNonStringValueNull - adds a column and a value with null detection for non-string value.  The value is enclosed with string quotes when the CommandType is INSERT or UPDATE
func (qb *QueryBuilder) AddColumnNonStringValueNull(ColumnName string, Value interface{}, NullDetectValue interface{}) *QueryBuilder {
	ci := qb.addColumn(ColumnName, 255)
	qb.setColumnValue(ci, Value, false, nil, NullDetectValue)

	return qb
}

//AddColumnValueWithDefaultNull - adds a column and a value with default value and null detection for BuildString() function. The value is enclosed with string quotes when the CommandType is INSERT or UPDATE
func (qb *QueryBuilder) AddColumnValueWithDefaultNull(ColumnName string, Value interface{}, Default interface{}, NullDetectValue interface{}) *QueryBuilder {
	ci := qb.addColumn(ColumnName, 255)
	qb.setColumnValue(ci, Value, true, Default, NullDetectValue)

	return qb
}

//AddColumnNonStringValueDefaultNull - adds a column and a value with default value and null detection for BuildString() function.
func (qb *QueryBuilder) AddColumnNonStringValueDefaultNull(ColumnName string, Value interface{}, Default interface{}, NullDetectValue interface{}) *QueryBuilder {
	ci := qb.addColumn(ColumnName, 255)
	qb.setColumnValue(ci, Value, false, Default, NullDetectValue)

	return qb
}

func (qb *QueryBuilder) addColumn(columnName string, length int) int {

	c := strings.ToLower(columnName)
	for i, v := range qb.Columns {
		if c == strings.ToLower(v.ColumnName) {
			return i
		}
	}

	qb.Columns = append(qb.Columns, queryColumn{ColumnName: columnName, Length: length})

	return len(qb.Columns) - 1
}

// CleanStringValue - clean a string value to prevent unescaped string errors for BuildString() function.
func (qb *QueryBuilder) CleanStringValue(Value string) string {

	if len(Value) > 0 {
		return strings.Replace(Value, qb.StringEnclosingChar, qb.StringEscapeChar+qb.StringEnclosingChar, -1)
	}

	return Value
}

func (qb *QueryBuilder) setColumnValue(ColumnIndex int, value interface{}, isDBString bool, defaultValue interface{}, nullDetectValue interface{}) *QueryBuilder {
	c := strings.ToLower(qb.Columns[ColumnIndex].ColumnName)
	idx := -1
	for i, v := range qb.Values {
		vc := strings.ToLower(v.ColumnName)
		if c == vc {
			idx = i
			break
		}
	}

	if idx == -1 {
		qb.Values = append(qb.Values, queryValue{
			ColumnName:      qb.Columns[ColumnIndex].ColumnName,
			IsDBString:      isDBString,
			DefaultValue:    defaultValue,
			NullDetectValue: nullDetectValue,
			Value:           value,
		})
	} else {
		qb.Values[idx].IsDBString = isDBString
		qb.Values[idx].DefaultValue = defaultValue
		qb.Values[idx].NullDetectValue = nullDetectValue
		qb.Values[idx].Value = value
	}

	return qb
}

//AddFilterWithValue - adds a filter with value into the QueryBuilder for BuildDataHelper() function.
func (qb *QueryBuilder) AddFilterWithValue(ColumnNameOrExpression string, Value interface{}) *QueryBuilder {

	qb.Filter = append(qb.Filter, queryFilter{
		ColumnNameOrExpression: ColumnNameOrExpression,
		Value:                  Value,
		IsDBString:             true,
	})

	return qb
}

//AddFilterWithNonStringValue - adds a filter with non-db string value into the QueryBuilder for BuildDataHelper() function.
func (qb *QueryBuilder) AddFilterWithNonStringValue(ColumnNameOrExpression string, Value interface{}) *QueryBuilder {

	qb.Filter = append(qb.Filter, queryFilter{
		ColumnNameOrExpression: ColumnNameOrExpression,
		Value:                  Value,
		IsDBString:             false,
	})

	return qb
}

//AddFilter - adds a filter into the QueryBuilder for both BuildString() and BuildDataHelper() function.
func (qb *QueryBuilder) AddFilter(FilterExpression string) *QueryBuilder {

	qb.Filter = append(qb.Filter, queryFilter{
		ColumnNameOrExpression: FilterExpression,
		Value:                  nil,
		FilterContainsValue:    true,
	})

	return qb
}

//AddOrder - adds a column to order by into the QueryBuilder for both BuildString() and BuildDataHelper() function.
func (qb *QueryBuilder) AddOrder(ColumnName string, Sorting QuerySortDirection) *QueryBuilder {
	qb.Order = append(qb.Order, querySort{ColumnName: ColumnName, Order: Sorting})

	return qb
}

//AddGroup - adds a group by into the QueryBuilder for both BuildString() and BuildDataHelper() function
func (qb *QueryBuilder) AddGroup(Group string) *QueryBuilder {
	qb.Group = append(qb.Group, Group)

	return qb
}

//BuildString - build an SQL string from QueryBuilder
func (qb *QueryBuilder) BuildString() (string, error) {
	retsql := ""

	valid, s := qb.basicValidation()
	if !valid {
		return "", errors.New(s)
	}

	// Auto attach schema if config is specified
	tbn := qb.TableName

	switch qb.CommandType {
	case SELECT:
		retsql = "SELECT "
		if len(qb.ResultLimit) > 0 && qb.ResultLimitPosition == FRONT {
			retsql += " TOP " + qb.ResultLimit + " "
		}
	case INSERT:
		retsql = "INSERT INTO " + tbn + " ("
	case UPDATE:
		retsql = "UPDATE " + tbn + " SET "
	case DELETE:
		retsql = "DELETE FROM " + tbn
	}

	//build columns (with placeholder for update )
	cma := ""
	if len(qb.Values) > 0 {
		for idx, v := range qb.Values {

			// Skip columns to render if the SkipNilWriteColumn is true and value is nil
			valueIsNil := false
			if v.Value == nil {
				valueIsNil = true
			} else {
				t := reflect.TypeOf(v.Value)
				if t == nil {
					valueIsNil = true
				} else {
					tv := reflect.ValueOf(v.Value)
					if tv.IsZero() {
						if tv.IsNil() {
							valueIsNil = true
						}
					}
				}
			}

			qb.Values[idx].skip = qb.SkipNilWriteColumn && valueIsNil

			switch qb.CommandType {
			case SELECT:
				retsql += cma + v.ColumnName
				cma = ", "
			case INSERT:
				if !qb.Values[idx].skip {
					retsql += cma + v.ColumnName
					cma = ", "
				}
			case UPDATE:
				if !qb.Values[idx].skip {
					if v.Value != nil {
						retsql += cma + v.ColumnName + " = " + qb.evaluateValue(v)
					} else {
						retsql += cma + v.ColumnName + " = NULL"
					}

					cma = ", "
				}
			}
		}
	} else {
		if qb.CommandType == SELECT {
			for _, v := range qb.Columns {
				retsql += cma + v.ColumnName
				cma = ", "
			}
		}
	}

	/* Append table name for SELECT*/
	if qb.CommandType == SELECT {
		retsql += " FROM " + tbn
	}

	//build value place holder for insert
	cma = ""
	if qb.CommandType == INSERT {
		tmpsql := ""
		for idx, v := range qb.Values {
			if !qb.Values[idx].skip {
				if v.Value != nil {
					tmpsql += cma + qb.evaluateValue(v)
				} else {
					tmpsql += cma + "NULL"
				}

				cma = ", "
			}
		}
		retsql += ") VALUES (" + tmpsql + ")"
	}

	//build filters
	cma = ""
	if len(qb.Filter) > 0 {
		tmpsql := ""
		for _, c := range qb.Filter {
			/* Only filters set with value will be rendered here */
			if qb.CommandType == SELECT || qb.CommandType == UPDATE || qb.CommandType == DELETE {
				if c.Value != nil {
					tmpsql += cma + c.ColumnNameOrExpression + " = " + qb.evaluateValue(queryValue{
						ColumnName: c.ColumnNameOrExpression,
						Value:      c.Value,
						IsDBString: c.IsDBString,
					})
				} else {
					if !c.FilterContainsValue {
						tmpsql += cma + c.ColumnNameOrExpression + " IS NULL"
					}
				}

				cma = " AND "
			}
		}

		if len(tmpsql) > 0 {
			retsql += " WHERE " + tmpsql
		}
	}

	//build sort orders
	cma = ""
	if len(qb.Order) > 0 {
		retsql += " ORDER BY "
		for _, v := range qb.Order {
			retsql += cma + v.ColumnName

			switch v.Order {
			case ASC:
				retsql += " ASC"
			case DESC:
				retsql += " DESC"
			}

			cma = ", "
		}
	}

	//build group by
	cma = ""
	if len(qb.Group) > 0 {
		retsql += " GROUP BY " + strings.Join(qb.Group, ", ")
	}

	if len(qb.ResultLimit) > 0 && qb.ResultLimitPosition == REAR {
		retsql += " LIMIT " + qb.ResultLimit
	}

	retsql += ";"

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
		retsql = replaceCustomPlaceHolder(retsql, sch)
	}

	return retsql, nil
}

// BuildDataHelper - build query for DataHelper (github.com/eaglebush/datahelper)
func (qb *QueryBuilder) BuildDataHelper() (query string, args []interface{}) {
	retsql := ""
	retargs := make([]interface{}, 0)

	valid, s := qb.basicValidation()
	if !valid {
		panic(s)
	}

	// Auto attach schema
	tbn := qb.TableName
	// if qb.dbinfo != nil {
	// 	pos := strings.LastIndex(tbn, `.`)
	// 	if pos == -1 && qb.dbinfo.Schema != "" {
	// 		// Get reserved word escape chars
	// 		rwe := parseReserveWordsChars(qb.dbinfo.ReservedWordEscapeChar)

	// 		if strings.Index(tbn, rwe[0]) != -1 && strings.Index(tbn, rwe[1]) != -1 {
	// 			tbn = rwe[0] + qb.dbinfo.Schema + rwe[1] + `.` + tbn
	// 		} else {
	// 			tbn = qb.dbinfo.Schema + `.` + tbn
	// 		}
	// 	}
	// }

	switch qb.CommandType {
	case SELECT:
		retsql = "SELECT "
		if len(qb.ResultLimit) > 0 && qb.ResultLimitPosition == FRONT {
			retsql += " TOP " + qb.ResultLimit + " "
		}
	case INSERT:
		retsql = "INSERT INTO " + tbn + " ("
	case UPDATE:
		retsql = "UPDATE " + tbn + " SET "
	case DELETE:
		retsql = "DELETE FROM " + tbn
	}

	//build columns (with placeholder for update )
	cma := ""
	pchar := ""
	paramcnt := 0
	columncnt := 0
	nullnow := false

	for idx, v := range qb.Values {

		// Skip columns to render if the SkipNilWriteColumn is true and value is nil
		valueIsNil := false
		if v.Value == nil {
			valueIsNil = true
		} else {
			t := reflect.TypeOf(v.Value)
			if t == nil {
				valueIsNil = true
			} else {
				tv := reflect.ValueOf(v.Value)
				if tv.IsZero() {
					k := t.Kind()
					if k == reflect.Map || k == reflect.Func || k == reflect.Ptr || k == reflect.Slice || k == reflect.Interface {
						if tv.IsNil() {
							valueIsNil = true
						}
					}
				}
			}
		}

		qb.Values[idx].skip = qb.SkipNilWriteColumn && valueIsNil
		nullnow = v.NullDetectValue == v.Value && v.NullDetectValue != nil

		switch qb.CommandType {
		case SELECT:
			retsql += cma + v.ColumnName
			cma = ", "
			columncnt++
		case INSERT:
			if !qb.Values[idx].skip {
				retsql += cma + v.ColumnName
				cma = ", "
				columncnt++
			}
		case UPDATE:
			if !qb.Values[idx].skip {
				retsql += cma + v.ColumnName
				pchar = " = "

				if v.IsDBString {
					pchar += qb.PreparedStatementChar
					if qb.PreparedStatementInSequence {
						paramcnt++
						pchar += strconv.Itoa(paramcnt)
					}
				} else {
					if nullnow {
						pchar += " NULL "
					} else {
						pchar += v.Value.(string)
					}
				}

				retsql += pchar
				cma = ", "
				columncnt++
			}
		}
	}

	/* Append table name for SELECT*/
	if qb.CommandType == SELECT {
		retsql += " FROM " + tbn
	}

	//build value place holder for insert
	cma = ""
	pchar = ""
	inscnt := 0

	if qb.CommandType == INSERT {

		q := make([]string, columncnt)
		for _, v := range qb.Values {
			if !v.skip {
				// On BuildDataHelper, the IsDBString property is interpreted as a literal string that may indicate SQL Functions
				nullnow = v.NullDetectValue == v.Value && v.NullDetectValue != nil
				pchar = qb.PreparedStatementChar

				if v.IsDBString {
					if qb.PreparedStatementInSequence {
						paramcnt++
						pchar += strconv.Itoa(paramcnt)
					}
				} else {
					if !nullnow {
						pchar = v.Value.(string)
					}
				}
				q[inscnt] = cma + pchar

				cma = ","
				inscnt++
			}
		}

		retsql += ") VALUES (" + strings.Join(q, "") + ")"

	}

	//build filters
	cma = ""
	if len(qb.Filter) > 0 {
		tmpsql := ""

		for _, c := range qb.Filter {
			/* Only filters set with value will be rendered here */
			if qb.CommandType == SELECT || qb.CommandType == UPDATE || qb.CommandType == DELETE {
				if c.Value != nil {
					pchar = qb.PreparedStatementChar
					if qb.PreparedStatementInSequence {
						paramcnt++
						pchar += strconv.Itoa(paramcnt)
					}
					tmpsql += cma + c.ColumnNameOrExpression + " = " + pchar
				} else {
					if !c.FilterContainsValue {
						tmpsql += cma + c.ColumnNameOrExpression + " IS NULL"
					}
				}

				cma = " AND "
			}
		}

		if len(tmpsql) > 0 {
			retsql += " WHERE " + tmpsql
		}
	}

	//build sort orders
	cma = ""
	if len(qb.Order) > 0 {
		retsql += " ORDER BY "
		for _, v := range qb.Order {
			retsql += cma + v.ColumnName

			switch v.Order {
			case ASC:
				retsql += " ASC"
			case DESC:
				retsql += " DESC"
			}

			cma = ", "
		}
	}

	//build group by
	cma = ""
	if len(qb.Group) > 0 {
		retsql += " GROUP BY " + strings.Join(qb.Group, ", ")
	}

	if len(qb.ResultLimit) > 0 && qb.ResultLimitPosition == REAR {
		retsql += " LIMIT " + qb.ResultLimit
	}

	retsql += ";"

	//build values
	for _, v := range qb.Values {
		if !v.skip {

			if qb.CommandType == INSERT || qb.CommandType == UPDATE {
				if v.NullDetectValue == v.Value && v.NullDetectValue != nil {
					if v.IsDBString {
						retargs = append(retargs, new(interface{}))
					}
				} else {
					if v.IsDBString {
						if v.Value != nil {
							retargs = append(retargs, v.Value)
						} else {
							retargs = append(retargs, new(interface{}))
						}
					}
				}
			}
		}
	}

	//build filter values
	if len(qb.Filter) > 0 {
		for _, v := range qb.Filter {
			if (qb.CommandType == SELECT || qb.CommandType == UPDATE || qb.CommandType == DELETE) && v.Value != nil {
				retargs = append(retargs, v.Value)
			}
		}
	}

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
		retsql = replaceCustomPlaceHolder(retsql, sch)

	}

	return retsql, retargs
}

func (qb *QueryBuilder) evaluateValue(value queryValue) string {
	s := ""
	var final interface{}

	if value.Value != nil {
		final = value.Value
	} else {
		if value.DefaultValue != nil {
			final = value.DefaultValue
		}
	}

	if value.NullDetectValue != nil {
		if final == value.NullDetectValue {
			s = "NULL"
		}
	}

	if final != nil && len(s) == 0 {
		//v := reflect.TypeOf(final)
		switch final.(type) {
		case int:
			s = strconv.FormatInt(int64(final.(int)), 10)
		case int8:
			s = strconv.FormatInt(int64(final.(int8)), 10)
		case int16:
			s = strconv.FormatInt(int64(final.(int16)), 10)
		case int32:
			s = strconv.FormatInt(int64(final.(int32)), 10)
		case int64:
			s = strconv.FormatInt(final.(int64), 10)
		case uint:
			s = strconv.FormatUint(uint64(final.(uint)), 10)
		case uint8:
			s = strconv.FormatUint(uint64(final.(uint8)), 10)
		case uint16:
			s = strconv.FormatUint(uint64(final.(uint16)), 10)
		case uint32:
			s = strconv.FormatUint(uint64(final.(uint32)), 10)
		case uint64:
			s = strconv.FormatUint(final.(uint64), 10)
		case float32:
			s = fmt.Sprintf("%f", final.(float32))
		case float64:
			s = fmt.Sprintf("%f", final.(float64))
		case bool:
			if final.(bool) {
				s = "1"
			} else {
				s = "0"
			}
		case string:
			// For BuildDataString(), the IsDBString is interpreted as a string that needs to be enclosed in quotes and escaped
			if value.IsDBString {
				s = "'" + qb.CleanStringValue(final.(string)) + "'"
			} else {
				s = final.(string)
			}
		case time.Time:
			s = "'" + final.(time.Time).Format(time.RFC3339) + "'"
		}
	}

	return s
}

func (qb *QueryBuilder) basicValidation() (bool, string) {
	if len(qb.TableName) == 0 {
		return false, "TableName was not specified"
	}

	if len(qb.Columns) == 0 && qb.CommandType != DELETE {
		return false, "No columns were specified"
	}

	/*
		if len(qb.Order) > 0 && (qb.CommandType == DELETE || qb.CommandType == INSERT || qb.CommandType == UPDATE) {
			return false, "Ordering (ORDER BY) is not supported when CommandType is DELETE, INSERT and UPDATE"
		}

		if len(qb.Group) > 0 && (qb.CommandType == DELETE || qb.CommandType == INSERT || qb.CommandType == UPDATE) {
			return false, "Grouping (GROUP BY) is not supported when CommandType is DELETE, INSERT and UPDATE"
		}
	*/

	return true, ""
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

func replaceCustomPlaceHolder(sql string, schema string) string {
	if schema != "" {
		schema = schema + `.`
	}

	re := regexp.MustCompile(`\{([a-zA-Z0-9\[\]\"]*)\}`)
	sql = re.ReplaceAllString(sql, schema+`$1`)

	return sql
}
