// Package QueryBuilder
// v2.0
// 2024.08.01
// Builds SQL query based on the inputs
//
// TODO:
//	- figure out how to cache queries to save iteration of columns and values

package querybuilder

import (
	"errors"
	"log"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	dhl "github.com/NarsilWorks-Inc/datahelperlite"
	di "github.com/eaglebush/datainfo"
	ssd "github.com/shopspring/decimal"
)

type CommandType uint8
type Sort uint8
type Limit uint8

// CommandType enum
const (
	SELECT CommandType = 0 // Select record type
	INSERT CommandType = 1 // Insert record type
	UPDATE CommandType = 2 // Update record type
	DELETE CommandType = 3 // Delete record type
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

// Option function for QueryBuilder
type Option func(q *QueryBuilder) error
type ValueOption func(vo *ValueCompareOption) error

// ValueCompareOption options for adding values
type ValueCompareOption struct {
	SQLString   bool // Sets if the value is an SQL string. When true, this value is enclosed by the database client in single quotes to represent as string
	Default     any  // When set to non-nil, this is the default value when the value encounters a nil
	MatchToNull any  // When the primary value matches with this value, the resulting value will be set to NULL
}

type QueryColumn struct {
	Name   string // name of the column
	Length int    // length of the column
}

type EngineConstants struct {
	StringEnclosingChar    string // Gets or sets the character that encloses a string in the query
	StringEscapeChar       string // Gets or Sets the character that escapes a reserved character such as the character that encloses a s string
	ReservedWordEscapeChar string // Reserved word escape chars. For escaping with different opening and closing characters, just set to both. Example. `[]` for SQL server
	ParameterChar          string // Gets or sets the character placeholder for prepared statements
	ParameterInSequence    bool   // Sets of the placeholders will be generated as a sequence of placeholder. Example, for SQL Server, @p0, @p1 @p2
	ResultLimitPosition    Limit  // The position of the row limiting statement in a query. For SQL Server, the limiting is set at the SELECT clause such as TOP 1. Later versions of SQL server supports OFFSET and FETCH.
}

type queryValue struct {
	column      string // Name of the column
	value       any    // value of the column
	defValue    any    // default value
	matchToNull any    // when primary value is matched by this value, it will set the value to NULL
	sqlstring   bool   // indicates if the value is an SQL string
	skip        bool   // skip this query value
	forceNull   bool   // forced to null
}

type queryFilter struct {
	expression    string // Column name or expression of the filter
	value         any    // Value of the filter if the expression is a column name
	containsValue bool   // indicates that the filter has a separate value, not a filter expression
}

type querySort struct {
	column string
	order  Sort
}

// QueryBuilder is a structure to build SQL queries
type QueryBuilder struct {
	// Public fields
	Source          string                                                      // Table or view name of the query
	CommandType     CommandType                                                 // Command type
	Filter          []queryFilter                                               // Query filter
	ResultLimit     string                                                      // The value of the row limit
	ParameterOffset int                                                         // The parameter sequence offset
	FilterFunc      func(offset int, char string, inSeq bool) ([]string, []any) // returns filter from outside functions like filterbuilder

	// Private fields
	skpNilWrCol bool            // Sets the condition that the Nil columns in an INSERT or UPDATE command would be skipped, instead of being set.
	dbEnConst   EngineConstants // These constants are specific for database vendors
	intTbls     bool            // When true, all table name with {} around it will be prepended with schema
	order       []querySort     // Order by columns
	group       []string        // Group by columns
	columns     []QueryColumn   // Columns of the query
	values      []queryValue    // Values of the columns
	dbInfo      *di.DataInfo    // The database information from the configuration
	distinct    bool            // The output should return distinct values
}

// New builds a new QueryBuilder
func New(options ...Option) *QueryBuilder {
	n := QueryBuilder{
		dbEnConst:   InitConstants(nil),
		intTbls:     true,
		skpNilWrCol: true,
		ResultLimit: "",
	}
	for _, o := range options {
		if o == nil {
			continue
		}
		o(&n)
	}
	if n.dbInfo == nil {
		n.dbInfo = di.New()
		n.dbInfo.StringEnclosingChar = &n.dbEnConst.StringEnclosingChar
		n.dbInfo.StringEscapeChar = &n.dbEnConst.StringEscapeChar
		n.dbInfo.ParameterPlaceHolder = &n.dbEnConst.ParameterChar
		n.dbInfo.ReservedWordEscapeChar = &n.dbEnConst.ReservedWordEscapeChar
		n.dbInfo.ParameterInSequence = &n.dbEnConst.ParameterInSequence
		n.dbInfo.ResultLimitPosition = di.LimitPosition(n.dbEnConst.ResultLimitPosition)
		log.Println("[QueryBuilder] Warning: DataInfo was not explicitly set. Using default with DBEngineConstants default values.")
	}

	return &n
}

// Spawn creates a copy of a builder and resets the non-constant values
func Spawn(builder QueryBuilder, options ...Option) *QueryBuilder {
	n := QueryBuilder{
		dbEnConst:   builder.dbEnConst,
		skpNilWrCol: builder.skpNilWrCol,
		intTbls:     builder.intTbls,
		ResultLimit: "",
	}
	for _, o := range options {
		if o == nil {
			continue
		}
		o(&n)
	}
	if n.dbInfo == nil {
		n.dbInfo = di.New()
		n.dbInfo.StringEnclosingChar = &n.dbEnConst.StringEnclosingChar
		n.dbInfo.StringEscapeChar = &n.dbEnConst.StringEscapeChar
		n.dbInfo.ParameterPlaceHolder = &n.dbEnConst.ParameterChar
		n.dbInfo.ReservedWordEscapeChar = &n.dbEnConst.ReservedWordEscapeChar
		n.dbInfo.ParameterInSequence = &n.dbEnConst.ParameterInSequence
		n.dbInfo.ResultLimitPosition = di.LimitPosition(n.dbEnConst.ResultLimitPosition)
		log.Println("[QueryBuilder] Warning: DataInfo was not explicitly set. Using default with DBEngineConstants default values.")
	}
	return &n
}

// InitConstants return defaults of database engine constants, with database configuration if present
func InitConstants(di *di.DataInfo) EngineConstants {
	ec := EngineConstants{
		StringEnclosingChar:    `'`,
		StringEscapeChar:       `\`,
		ParameterChar:          `?`,
		ReservedWordEscapeChar: `"`,
		ParameterInSequence:    false,
		ResultLimitPosition:    REAR,
	}
	if di != nil {
		if di.StringEnclosingChar != nil && *di.StringEnclosingChar != "" {
			ec.StringEnclosingChar = *di.StringEnclosingChar
		}
		if di.StringEscapeChar != nil && *di.StringEscapeChar != "" {
			ec.StringEscapeChar = *di.StringEscapeChar
		}
		if di.ParameterPlaceHolder != nil && *di.ParameterPlaceHolder != "" {
			ec.ParameterChar = *di.ParameterPlaceHolder
		}
		if di.ReservedWordEscapeChar != nil && *di.ReservedWordEscapeChar != "" {
			ec.ReservedWordEscapeChar = *di.ReservedWordEscapeChar
		}
		if di.ParameterInSequence != nil {
			ec.ParameterInSequence = *di.ParameterInSequence
		}
		ec.ResultLimitPosition = Limit(di.ResultLimitPosition)
	}
	return ec
}

// Distinct sets the option to return distinct values
func Distinct(yes bool) Option {
	return func(q *QueryBuilder) error {
		q.distinct = yes
		return nil
	}
}

// Source sets the table, view or stored procedure name
func Source(name string) Option {
	return func(q *QueryBuilder) error {
		q.Source = name
		return nil
	}
}

// Schema sets the schema of a query builder
func Schema(sch string) Option {
	return func(q *QueryBuilder) error {
		if q.dbInfo == nil {
			q.dbInfo = di.New()
			q.dbInfo.StringEnclosingChar = &q.dbEnConst.StringEnclosingChar
			q.dbInfo.StringEscapeChar = &q.dbEnConst.StringEscapeChar
			q.dbInfo.ParameterPlaceHolder = &q.dbEnConst.ParameterChar
			q.dbInfo.ReservedWordEscapeChar = &q.dbEnConst.ReservedWordEscapeChar
			q.dbInfo.ParameterInSequence = &q.dbEnConst.ParameterInSequence
			q.dbInfo.ResultLimitPosition = di.LimitPosition(q.dbEnConst.ResultLimitPosition)
			log.Println("[QueryBuilder] Warning: DataInfo was not explicitly set. Using default with DBEngineConstants default values.")
		}
		q.dbInfo.Schema = new(string)
		*q.dbInfo.Schema = sch
		return nil
	}
}

// Command sets the command of a query builder
func Command(ct CommandType) Option {
	return func(q *QueryBuilder) error {
		q.CommandType = ct
		return nil
	}
}

// Config sets the database info
func DatabaseInfo(dnf *di.DataInfo) Option {
	return func(q *QueryBuilder) error {
		q.dbInfo = dnf
		q.dbEnConst = InitConstants(dnf)
		return nil
	}
}

// Constants are builder settings that follows the database engine settings.
func Constants(ec EngineConstants) Option {
	return func(q *QueryBuilder) error {
		q.dbEnConst = ec
		return nil
	}
}

// Interpolate converts all table name with {} around it will be prepended with schema and reference code prefix
func Interpolate(value bool) Option {
	return func(q *QueryBuilder) error {
		q.intTbls = value
		return nil
	}
}

// ReferenceMode enables the builder to generate query that adds a `ref` prefix to table names after the schema.
//
// This can be used in instances that the database object is just a reference populated by event source rather than user interaction.
//
// Warning: If the interpolation is set to off, this property is ignored.
func ReferenceMode(value bool) Option {
	return func(q *QueryBuilder) error {
		if q.dbInfo == nil {
			q.dbInfo = di.New()
			q.dbInfo.StringEnclosingChar = &q.dbEnConst.StringEnclosingChar
			q.dbInfo.StringEscapeChar = &q.dbEnConst.StringEscapeChar
			q.dbInfo.ParameterPlaceHolder = &q.dbEnConst.ParameterChar
			q.dbInfo.ReservedWordEscapeChar = &q.dbEnConst.ReservedWordEscapeChar
			q.dbInfo.ParameterInSequence = &q.dbEnConst.ParameterInSequence
			q.dbInfo.ResultLimitPosition = di.LimitPosition(q.dbEnConst.ResultLimitPosition)
			log.Println("[QueryBuilder] Warning: DataInfo was not explicitly set. Using default with DBEngineConstants default values.")
		}
		q.dbInfo.ReferenceMode = new(bool)
		*q.dbInfo.ReferenceMode = value
		return nil
	}
}

// ReferenceModePrefix changes the reference prefix to add to database object names when set in ReferenceMode
//
// Warning: If the interpolation is set to off, this property is ignored.
func ReferenceModePrefix(prefix string) Option {
	return func(q *QueryBuilder) error {
		if prefix == "" {
			return nil
		}
		if q.dbInfo == nil {
			q.dbInfo = di.New()
			q.dbInfo.StringEnclosingChar = &q.dbEnConst.StringEnclosingChar
			q.dbInfo.StringEscapeChar = &q.dbEnConst.StringEscapeChar
			q.dbInfo.ParameterPlaceHolder = &q.dbEnConst.ParameterChar
			q.dbInfo.ReservedWordEscapeChar = &q.dbEnConst.ReservedWordEscapeChar
			q.dbInfo.ParameterInSequence = &q.dbEnConst.ParameterInSequence
			q.dbInfo.ResultLimitPosition = di.LimitPosition(q.dbEnConst.ResultLimitPosition)
			log.Println("[QueryBuilder] Warning: DataInfo was not explicitly set. Using default with DBEngineConstants default values.")
		}
		q.dbInfo.ReferenceModePrefix = new(string)
		*q.dbInfo.ReferenceModePrefix = prefix
		return nil
	}
}

// ResultLimit sets the result limit at initialization. ResultLimit can also be set at QueryBuilder ResultLimit field.
func ResultLimit(value string) Option {
	return func(q *QueryBuilder) error {
		q.ResultLimit = value
		return nil
	}
}

// SkipNilWrite sets the condition to skip nil columns when writing to table
func SkipNilWrite(skip bool) Option {
	return func(q *QueryBuilder) error {
		q.skpNilWrCol = skip
		return nil
	}
}

// IsSqlString sets if the value is an SQL string. When true, this value is enclosed by the database client in single quotes to represent as string
func IsSqlString(indeed bool) ValueOption {
	return func(vco *ValueCompareOption) error {
		vco.SQLString = indeed
		return nil
	}
}

// Default is the default value of the column when the value encounters a nil
func Default(value any) ValueOption {
	return func(vco *ValueCompareOption) error {
		vco.Default = value
		return nil
	}
}

// MatchToNull is the condition the primary value matches with this value, the resulting value will be set to NULL
func MatchToNull(match any) ValueOption {
	return func(vco *ValueCompareOption) error {
		vco.MatchToNull = match
		return nil
	}
}

// NewSelect is a shortcut builder for Select queries
//
// dataObject can be a table, view or a joined query name
func NewSelect(dataObject string, opts ...Option) *QueryBuilder {
	opts = append(opts, Source(dataObject), Command(SELECT))
	return New(opts...)
}

// NewInsert is a shortcut builder for Insert queries
func NewInsert(table string, opts ...Option) *QueryBuilder {
	opts = append(opts, Source(table), Command(INSERT))
	return New(opts...)
}

// NewUpdate is a shortcut builder for Update queries
func NewUpdate(table string, opts ...Option) *QueryBuilder {
	opts = append(opts, Source(table), Command(UPDATE))
	return New(opts...)
}

// NewDelete is a shortcut builder for Delete queries
func NewDelete(table string, opts ...Option) *QueryBuilder {
	opts = append(opts, Source(table), Command(DELETE))
	return New(opts...)
}

// SpawnSelect creates a new builder out of factory and initializes for query
//
// dataObject can be a table, view or a joined query name
func SpawnSelect(builder *QueryBuilder, dataObject string, opts ...Option) *QueryBuilder {
	opts = append(opts, Source(dataObject), Command(SELECT))
	return Spawn(*builder, opts...)
}

// SpawnInsert creates a new builder out of factory and initializes for insert command
func SpawnInsert(builder *QueryBuilder, table string, opts ...Option) *QueryBuilder {
	opts = append(opts, Source(table), Command(INSERT))
	return Spawn(*builder, opts...)
}

// SpawnUpdate creates a new builder out of factory and initializes for update command
func SpawnUpdate(builder *QueryBuilder, table string, opts ...Option) *QueryBuilder {
	opts = append(opts, Source(table), Command(UPDATE))
	return Spawn(*builder, opts...)
}

// SpawnDelete creates a new builder out of factory and initializes for delete command
func SpawnDelete(builder *QueryBuilder, table string, opts ...Option) *QueryBuilder {
	opts = append(opts, Source(table), Command(DELETE))
	return Spawn(*builder, opts...)
}

// AddColumn adds a column to the builder
func (qb *QueryBuilder) AddColumn(name string) *QueryBuilder {
	if qb.CommandType == DELETE {
		return qb
	}
	return qb.setColumnValue(qb.addColumn(name, 255), nil, true, nil, nil)
}

// AddColumnFixed adds a column with specified length
func (qb *QueryBuilder) AddColumnFixed(name string, length int) *QueryBuilder {
	if qb.CommandType == DELETE {
		return qb
	}
	return qb.setColumnValue(qb.addColumn(name, length), nil, true, nil, nil)
}

// AddValue adds a value. The value options sets certain conditions to evaluate the supplied value
func (qb *QueryBuilder) AddValue(name string, value any, vcOpts ...ValueOption) *QueryBuilder {
	vo := ValueCompareOption{
		SQLString:   true,
		Default:     nil,
		MatchToNull: nil,
	}
	for _, o := range vcOpts {
		if o == nil {
			continue
		}
		o(&vo)
	}
	return qb.setColumnValue(qb.addColumn(name, 8000), value, vo.SQLString, vo.Default, vo.MatchToNull)
}

// SetColumnValue - sets the column value
func (qb *QueryBuilder) SetColumnValue(name string, value any) *QueryBuilder {
	if qb.CommandType == DELETE {
		return qb
	}
	for i, v := range qb.values {
		if strings.EqualFold(name, v.column) {
			continue
		}
		return qb.setColumnValue(i, value, true, nil, nil)
	}
	return qb
}

// Escape a string value to prevent unescaped errors
func (qb *QueryBuilder) Escape(value string) string {
	if len(value) > 0 {
		return strings.ReplaceAll(
			value,
			qb.dbEnConst.StringEnclosingChar,
			qb.dbEnConst.StringEscapeChar+qb.dbEnConst.StringEnclosingChar)
	}
	return value
}

// AddFilter adds a filter with value.
func (qb *QueryBuilder) AddFilter(column string, value any) *QueryBuilder {
	qb.Filter = append(
		qb.Filter,
		queryFilter{
			expression: column,
			value:      value,
		})
	return qb
}

// AddFilterExp adds a specific filter expression that could not be done with AddFilter
func (qb *QueryBuilder) AddFilterExp(expr string) *QueryBuilder {
	qb.Filter = append(qb.Filter, queryFilter{
		expression:    expr,
		value:         nil,
		containsValue: true,
	})
	return qb
}

// AddOrder - adds a column to order by into the QueryBuilder for both BuildString() and BuildDataHelper() function.
func (qb *QueryBuilder) AddOrder(column string, order Sort) *QueryBuilder {
	qb.order = append(qb.order, querySort{column: column, order: order})
	return qb
}

// AddGroup - adds a group by clause
func (qb *QueryBuilder) AddGroup(group string) *QueryBuilder {
	qb.group = append(qb.group, group)
	return qb
}

// Build an SQL string with corresponding values
func (qb *QueryBuilder) Build() (query string, args []any, err error) {
	if qb.Source == "" {
		return "", nil, ErrNoTableSpecified
	}
	if len(qb.columns) == 0 && qb.CommandType != DELETE {
		return "", nil, ErrNoColumnSpecified
	}
	// get real values of qb.Values and set them back
	for i := range qb.values {
		qb.values[i].value = realValue(qb.values[i].value)
		qb.values[i].defValue = realValue(qb.values[i].defValue)
		qb.values[i].matchToNull = realValue(qb.values[i].matchToNull)
	}

	// get real values of filter values and set them back
	for i := range qb.Filter {
		qb.Filter[i].value = realValue(qb.Filter[i].value)
	}

	// Auto attach schema
	var sb strings.Builder
	tbn := qb.Source
	switch qb.CommandType {
	case SELECT:
		sb.WriteString("SELECT ")
		if qb.distinct {
			sb.WriteString("DISTINCT ")
		}
		if len(qb.ResultLimit) > 0 && qb.dbEnConst.ResultLimitPosition == FRONT {
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

	for idx, v := range qb.values {
		qb.values[idx].forceNull = false
		isnl := isNil(v.value)
		// If value is nil, get defvalue
		if isnl && !isNil(v.defValue) {
			v.value = v.defValue
			isnl = false
		}
		// If matchtonull is true, column value is nil
		if !isnl && !isNil(v.matchToNull) && v.matchToNull == v.value {
			isnl = true
			qb.values[idx].forceNull = true
			qb.values[idx].sqlstring = true
		}
		// Skip columns to render if the SkipNilWriteColumn is true and value is nil
		qb.values[idx].skip = qb.skpNilWrCol && isnl
		switch qb.CommandType {
		case SELECT:
			sb.WriteString(cma + v.column)
			cma = ", "
			columncnt++
		case INSERT:
			if qb.values[idx].skip && !qb.values[idx].forceNull {
				break
			}
			sb.WriteString(cma + v.column)
			cma = ", "
			columncnt++
		case UPDATE:
			if qb.values[idx].skip && !qb.values[idx].forceNull {
				break
			}
			sb.WriteString(cma + v.column)
			pchar = " = "
			if isnl {
				pchar += "NULL"
			} else {
				if v.sqlstring {
					pchar += qb.dbEnConst.ParameterChar
					if qb.dbEnConst.ParameterInSequence {
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
		for _, v := range qb.values {
			if v.skip && !v.forceNull {
				continue
			}
			pchar = "NULL"
			if !isNil(v.value) && !v.forceNull {
				if !v.sqlstring {
					pchar, _ = v.value.(string)
				} else {
					pchar = qb.dbEnConst.ParameterChar
					if qb.dbEnConst.ParameterInSequence {
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
				pchar = qb.dbEnConst.ParameterChar
				if qb.dbEnConst.ParameterInSequence {
					paramcnt++
					pchar += strconv.Itoa(paramcnt)
				}
				tsb.WriteString(cma + c.expression + " = " + pchar)
			} else {
				tsb.WriteString(cma + c.expression)
				if !c.containsValue {
					tsb.WriteString(" IS NULL")
				}
			}
			cma = "\r\t\t AND "
		}
		if qb.FilterFunc != nil {
			fbs, _ := qb.FilterFunc(paramcnt, qb.dbEnConst.ParameterChar, qb.dbEnConst.ParameterInSequence)
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
	if len(qb.order) > 0 {
		sb.WriteString(" ORDER BY ")
		cma = ""
		for _, v := range qb.order {
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
	if len(qb.group) > 0 {
		sb.WriteString(" GROUP BY " + strings.Join(qb.group, ", "))
	}
	if len(qb.ResultLimit) > 0 && qb.dbEnConst.ResultLimitPosition == REAR {
		sb.WriteString(" LIMIT " + qb.ResultLimit)
	}
	sb.WriteString(";")

	// build values
	args = make([]any, 0, 15)
	for _, v := range qb.values {
		if v.skip ||
			!v.sqlstring ||
			!(qb.CommandType == INSERT || qb.CommandType == UPDATE) ||
			isNil(v.value) ||
			v.forceNull {
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
		fbs, fbargs := qb.FilterFunc(paramcnt, qb.dbEnConst.ParameterChar, qb.dbEnConst.ParameterInSequence)
		if len(fbs) > 0 {
			args = append(args, fbargs...)
		}
	}

	query = sb.String()
	if qb.intTbls {
		sch := ""
		if qb.dbInfo.ReferenceMode != nil && *qb.dbInfo.ReferenceMode {
			sch = *qb.dbInfo.ReferenceModePrefix
			if !strings.HasSuffix(sch, "_") {
				sch += "_"
			}
		}
		// If there is a schema defined, it will prevail
		if qb.dbInfo.Schema != nil && *qb.dbInfo.Schema != "" {
			sch = *qb.dbInfo.Schema
		}
		// replace table names marked with {table}
		query = InterpolateTable(query, sch)
	}
	qb.ParameterOffset = paramcnt
	return
}

func (qb *QueryBuilder) addColumn(name string, length int) int {
	for i, v := range qb.columns {
		if !strings.EqualFold(name, v.Name) {
			continue
		}
		return i
	}
	qb.columns = append(qb.columns, QueryColumn{Name: name, Length: length})
	return len(qb.columns) - 1
}

func (qb *QueryBuilder) setColumnValue(index int, value any, sqlString bool, defValue any, matchToNull any) *QueryBuilder {
	for i, v := range qb.values {
		if !strings.EqualFold(qb.columns[index].Name, v.column) {
			continue
		}
		qb.values[i].sqlstring = sqlString
		qb.values[i].defValue = defValue
		qb.values[i].matchToNull = matchToNull
		qb.values[i].value = value
		return qb
	}
	qb.values = append(qb.values, queryValue{
		column:      qb.columns[index].Name,
		sqlstring:   sqlString,
		defValue:    defValue,
		matchToNull: matchToNull,
		value:       value,
	})
	return qb
}

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

// // converts the value to a basic interface as nil or non-nil
// func realValue(value any) any {
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

func realValue(value any) any {
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
