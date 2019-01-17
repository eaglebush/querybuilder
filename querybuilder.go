/**
QueryBuilder package

Builds SQL query for both prepared and literal SQL string.
- BuildString() function - builds literal SQL string together with the supplied values. The functions tagged for BuildString() should be used for automatic formatting
- BuildDataHelper() function - builds a prepared command query and outputs an array of interface objects for arguments. The functions tagged for BuildDataHelper should be used.
*/

package querybuilder

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

//CommandType - the type of command to perform
type CommandType int

//QuerySortDirection - sort type
type QuerySortDirection int

//ResultLimitPosition -gets/sets the position of the row limiting command
type ResultLimitPosition int

//CommandType enum
const (
	SELECT CommandType = 0
	INSERT CommandType = 1
	UPDATE CommandType = 2
	DELETE CommandType = 3
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
	IsDBStringType  bool
}

type queryFilter struct {
	ColumnNameOrExpression string
	Value                  interface{}
	IsDBStringType         bool
}

//QuerySort - sort information
type querySort struct {
	ColumnName string
	Order      QuerySortDirection
}

//QueryBuilder - a class to build SQL queries
type QueryBuilder struct {
	TableName           string
	CommandType         CommandType
	Columns             []queryColumn
	Values              []queryValue
	Order               []querySort
	Group               []string
	Filter              []queryFilter
	StringEnclosingChar string
	StringEscapeChar    string
	ResultLimitPosition ResultLimitPosition
	ResultLimit         string
}

//NewQueryBuilder - builds a new QueryBuilder object
func NewQueryBuilder(table string) *QueryBuilder {
	return &QueryBuilder{TableName: table, StringEnclosingChar: `'`, StringEscapeChar: `\`, ResultLimitPosition: REAR, ResultLimit: ""}
}

//NewQueryBuilderBare - builds a new QueryBuilder object without a table name
func NewQueryBuilderBare() *QueryBuilder {
	return &QueryBuilder{StringEnclosingChar: `'`, StringEscapeChar: `\`, ResultLimitPosition: REAR, ResultLimit: ""}
}

//AddColumn - adds a column into the QueryBuilder
func (qb *QueryBuilder) AddColumn(ColumnName string) *QueryBuilder {
	if qb.CommandType != DELETE {
		qb.addColumn(ColumnName, 255) //only allows non-DELETE statements
	}

	return qb
}

//AddColumnWithLength - adds a column with specified length into the QueryBuilder
func (qb *QueryBuilder) AddColumnWithLength(ColumnName string, Length int) *QueryBuilder {
	if qb.CommandType != DELETE {
		qb.addColumn(ColumnName, Length) //only allows non-DELETE statements
	}

	return qb
}

//SetColumnValue - sets the column value
func (qb *QueryBuilder) SetColumnValue(ColumnName string, Value interface{}) *QueryBuilder {
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

//AddColumnNonStringValue - adds a column and a value for BuildString() function. The value is not enclosed in a string when the CommandType is INSERT or UPDATE
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
	idx := -1
	c := strings.ToLower(columnName)
	for i, v := range qb.Columns {
		if c == strings.ToLower(v.ColumnName) {
			idx = i
			break
		}
	}

	if idx == -1 {
		qb.Columns = append(qb.Columns, queryColumn{ColumnName: columnName, Length: length})
		idx = len(qb.Columns) - 1
	}

	return idx
}

//CleanStringValue - clean a string value to prevent unescaped string errors for BuildString() function.
func (qb *QueryBuilder) CleanStringValue(Value string) string {
	s := Value

	if len(s) > 0 {
		s = strings.Replace(s, qb.StringEnclosingChar, qb.StringEscapeChar+qb.StringEnclosingChar, -1)
	}

	return s
}

func (qb *QueryBuilder) setColumnValue(ColumnIndex int, value interface{}, isDBStringType bool, defaultValue interface{}, nullDetectValue interface{}) *QueryBuilder {
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
			IsDBStringType:  isDBStringType,
			DefaultValue:    defaultValue,
			NullDetectValue: nullDetectValue,
			Value:           value,
		})
	} else {
		qb.Values[idx].IsDBStringType = isDBStringType
		qb.Values[idx].DefaultValue = defaultValue
		qb.Values[idx].NullDetectValue = nullDetectValue
		qb.Values[idx].Value = value
	}

	return qb
}

//AddFilterWithValue - adds a filter with value into the QueryBuilder for BuildDataHelper() function.
func (qb *QueryBuilder) AddFilterWithValue(ColumnNameOrExpression string, Value interface{}) *QueryBuilder {
	qb.Filter = append(qb.Filter, queryFilter{ColumnNameOrExpression: ColumnNameOrExpression, Value: Value, IsDBStringType: true})

	return qb
}

//AddFilterWithNonStringValue - adds a filter with non-db string value into the QueryBuilder for BuildDataHelper() function.
func (qb *QueryBuilder) AddFilterWithNonStringValue(ColumnNameOrExpression string, Value interface{}) *QueryBuilder {
	qb.Filter = append(qb.Filter, queryFilter{ColumnNameOrExpression: ColumnNameOrExpression, Value: Value, IsDBStringType: false})

	return qb
}

//AddFilter - adds a filter into the QueryBuilder for both BuildString() and BuildDataHelper() function.
func (qb *QueryBuilder) AddFilter(FilterExpression string) *QueryBuilder {
	qb.Filter = append(qb.Filter, queryFilter{ColumnNameOrExpression: FilterExpression, Value: nil})

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

	switch qb.CommandType {
	case SELECT:
		retsql = "SELECT "
		if len(qb.ResultLimit) > 0 && qb.ResultLimitPosition == FRONT {
			retsql += " TOP " + qb.ResultLimit + " "
		}
	case INSERT:
		retsql = "INSERT INTO " + qb.TableName + " ("
	case UPDATE:
		retsql = "UPDATE " + qb.TableName + " SET "
	case DELETE:
		retsql = "DELETE FROM " + qb.TableName
	}

	//build columns (with placeholder for update )
	cma := ""
	if len(qb.Values) > 0 {
		for _, v := range qb.Values {
			switch qb.CommandType {
			case SELECT, INSERT:
				retsql += cma + v.ColumnName
				cma = ", "
			case UPDATE:
				retsql += cma + v.ColumnName + " = " + qb.evaluateValue(v)
				cma = ", "
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
		retsql += " FROM " + qb.TableName
	}

	//build value place holder for insert
	cma = ""
	if qb.CommandType == INSERT {
		tmpsql := ""
		for _, v := range qb.Values {
			tmpsql += cma + qb.evaluateValue(v)
			cma = ", "
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
					tmpsql += cma + c.ColumnNameOrExpression + " = " + qb.evaluateValue(queryValue{ColumnName: c.ColumnNameOrExpression, Value: c.Value, IsDBStringType: c.IsDBStringType})
				} else {
					tmpsql += cma + c.ColumnNameOrExpression
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

	return retsql, nil
}

//BuildDataHelper - build query for DataHelper (github.com/eaglebush/datahelper)
func (qb *QueryBuilder) BuildDataHelper() (query string, args []interface{}) {
	retsql := ""
	retargs := make([]interface{}, 0)

	valid, s := qb.basicValidation()
	if !valid {
		panic(s)
	}

	switch qb.CommandType {
	case SELECT:
		retsql = "SELECT "
		if len(qb.ResultLimit) > 0 && qb.ResultLimitPosition == FRONT {
			retsql += " TOP " + qb.ResultLimit + " "
		}
	case INSERT:
		retsql = "INSERT INTO " + qb.TableName + " ("
	case UPDATE:
		retsql = "UPDATE " + qb.TableName + " SET "
	case DELETE:
		retsql = "DELETE FROM " + qb.TableName
	}

	//build columns (with placeholder for update )
	cma := ""
	for _, v := range qb.Values {
		switch qb.CommandType {
		case SELECT, INSERT:
			retsql += cma + v.ColumnName
			cma = ", "
		case UPDATE:
			retsql += cma + v.ColumnName + " = ?"
			cma = ", "
		}
	}

	/* Append table name for SELECT*/
	if qb.CommandType == SELECT {
		retsql += " FROM " + qb.TableName
	}

	//build value place holder for insert
	cma = ""
	if qb.CommandType == INSERT {
		q := make([]string, len(qb.Columns))
		for i := 0; i < cap(q); i++ {
			q[i] = cma + "?"
			cma = ","
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
					tmpsql += cma + c.ColumnNameOrExpression + " = ?"
				} else {
					tmpsql += cma + c.ColumnNameOrExpression
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
		if qb.CommandType == INSERT || qb.CommandType == UPDATE {
			retargs = append(retargs, v.Value)
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
			if value.IsDBStringType {
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
