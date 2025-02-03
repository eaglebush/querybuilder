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
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	dhl "github.com/NarsilWorks-Inc/datahelperlite"
	cfg "github.com/eaglebush/config"
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
	SQLString   bool        // Sets if the value is an SQL string. When true, this value is enclosed by the database client in single quotes to represent as string
	Default     interface{} // When set to non-nil, this is the default value when the value encounters a nil
	MatchToNull interface{} // When the primary value matches with this value, the resulting value will be set to NULL
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
	column      string      // Name of the column
	value       interface{} // value of the column
	defvalue    interface{} // default value
	matchtonull interface{} // when primary value is matched by this value, it will set the value to NULL
	sqlstring   bool        // indicates if the value is an SQL string
	skip        bool        // skip this query value
	forcenull   bool        // forced to null
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

// QueryBuilder is a structure to build SQL queries
type QueryBuilder struct {
	Source              string // Table or view name of the query
	CommandType         CommandType
	Filter              []queryFilter                                                       // Query filter
	ResultLimit         string                                                              // The value of the row limit
	ParameterOffset     int                                                                 // The parameter sequence offset
	FilterFunc          func(offset int, char string, inSeq bool) ([]string, []interface{}) // returns filter from outside functions like filterbuilder
	referenceMode       bool
	referenceModePrefix string
	schema              string
	skipNilWriteColumn  bool // Sets the condition that the Nil columns in an INSERT or UPDATE command would be skipped, instead of being set.
	dbEngineConstants   EngineConstants
	interpolateTables   bool          // When true, all table name with {} around it will be prepended with schema
	order               []querySort   // Order by columns
	group               []string      // Group by columns
	columns             []QueryColumn // Columns of the query
	values              []queryValue  // Values of the columns
	dbInfo              *cfg.DatabaseInfo
}

// New builds a new QueryBuilder
//
// The following are the default values when calling this method to create a new QueryBuilder:
//
//	Command: SELECT
//	StringEnclosingChar:    `'`
//	StringEscapeChar:       `\`
//	ParameterChar:          `?`
//	ReservedWordEscapeChar: `"`
//	ResultLimitPosition:    REAR
//	ResultLimit:            ""
//	interpolateTables:      true
//	skipNilWriteColumn:     false
func New(options ...Option) *QueryBuilder {
	n := QueryBuilder{
		dbEngineConstants:   InitConstants(nil),
		ResultLimit:         "",
		interpolateTables:   true,
		skipNilWriteColumn:  true,
		referenceMode:       false,
		referenceModePrefix: `ref`,
	}
	for _, o := range options {
		if o == nil {
			continue
		}
		o(&n)
	}
	return &n
}

// Spawn creates a copy of a builder and resets the non-constant values
func Spawn(builder QueryBuilder, options ...Option) *QueryBuilder {
	n := QueryBuilder{
		dbEngineConstants:   builder.dbEngineConstants,
		referenceMode:       builder.referenceMode,
		referenceModePrefix: builder.referenceModePrefix,
		skipNilWriteColumn:  builder.skipNilWriteColumn,
		interpolateTables:   builder.interpolateTables,
		schema:              builder.schema,
		ResultLimit:         "",
	}
	for _, o := range options {
		if o == nil {
			continue
		}
		o(&n)
	}
	return &n
}

// InitConstants return defaults of database engine constants
func InitConstants(di *cfg.DatabaseInfo) EngineConstants {
	ec := EngineConstants{
		StringEnclosingChar:    `'`,
		StringEscapeChar:       `\`,
		ParameterChar:          `?`,
		ReservedWordEscapeChar: `"`,
		ParameterInSequence:    false,
		ResultLimitPosition:    REAR,
	}
	if di != nil {
		if di.StringEnclosingChar != nil {
			ec.StringEnclosingChar = *di.StringEnclosingChar
		}
		if di.StringEscapeChar != nil {
			ec.StringEscapeChar = *di.StringEscapeChar
		}
		if di.ParameterPlaceholder != "" {
			ec.ParameterChar = di.ParameterPlaceholder
		}
		ec.ParameterInSequence = di.ParameterInSequence
		if di.ReservedWordEscapeChar != nil {
			ec.ReservedWordEscapeChar = *di.ReservedWordEscapeChar
		}
	}
	return ec
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
		q.schema = sch
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
func Config(di *cfg.DatabaseInfo) Option {
	return func(q *QueryBuilder) error {
		q.dbInfo = di
		InitConstants(di)
		return nil
	}
}

// Constants are builder settings that follows the database engine settings.
func Constants(ec EngineConstants) Option {
	return func(q *QueryBuilder) error {
		q.dbEngineConstants = ec
		return nil
	}
}

// Interpolate converts all table name with {} around it will be prepended with schema and reference code prefix
func Interpolate(value bool) Option {
	return func(q *QueryBuilder) error {
		q.interpolateTables = value
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
		q.referenceMode = value
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
		q.referenceModePrefix = prefix
		return nil
	}
}

// SkipNilWrite sets the condition to skip nil columns when writing to table
func SkipNilWrite(skip bool) Option {
	return func(q *QueryBuilder) error {
		q.skipNilWriteColumn = skip
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
func Default(value interface{}) ValueOption {
	return func(vco *ValueCompareOption) error {
		vco.Default = value
		return nil
	}
}

// MatchToNull is the condition the primary value matches with this value, the resulting value will be set to NULL
func MatchToNull(match interface{}) ValueOption {
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
func (qb *QueryBuilder) AddValue(name string, value interface{}, vcOpts ...ValueOption) *QueryBuilder {
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
func (qb *QueryBuilder) SetColumnValue(name string, value interface{}) *QueryBuilder {
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
			qb.dbEngineConstants.StringEnclosingChar,
			qb.dbEngineConstants.StringEscapeChar+qb.dbEngineConstants.StringEnclosingChar)
	}
	return value
}

// AddFilter adds a filter with value.
func (qb *QueryBuilder) AddFilter(column string, value interface{}) *QueryBuilder {
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
		containsvalue: true,
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
func (qb *QueryBuilder) Build() (query string, args []interface{}, err error) {
	if qb.Source == "" {
		return "", nil, ErrNoTableSpecified
	}
	if len(qb.columns) == 0 && qb.CommandType != DELETE {
		return "", nil, ErrNoColumnSpecified
	}
	// get real values of qb.Values and set them back
	for i := range qb.values {
		qb.values[i].value = realValue(qb.values[i].value)
		qb.values[i].defvalue = realValue(qb.values[i].defvalue)
		qb.values[i].matchtonull = realValue(qb.values[i].matchtonull)
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
		if len(qb.ResultLimit) > 0 && qb.dbEngineConstants.ResultLimitPosition == FRONT {
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
		qb.values[idx].forcenull = false
		isnl := isNil(v.value)
		// If value is nil, get defvalue
		if isnl && !isNil(v.defvalue) {
			v.value = v.defvalue
			isnl = false
		}
		// If matchtonull is true, column value is nil
		if !isnl && !isNil(v.matchtonull) && v.matchtonull == v.value {
			isnl = true
			qb.values[idx].forcenull = true
			qb.values[idx].sqlstring = true
		}
		// Skip columns to render if the SkipNilWriteColumn is true and value is nil
		qb.values[idx].skip = qb.skipNilWriteColumn && isnl
		switch qb.CommandType {
		case SELECT:
			sb.WriteString(cma + v.column)
			cma = ", "
			columncnt++
		case INSERT:
			if qb.values[idx].skip && !qb.values[idx].forcenull {
				break
			}
			sb.WriteString(cma + v.column)
			cma = ", "
			columncnt++
		case UPDATE:
			if qb.values[idx].skip && !qb.values[idx].forcenull {
				break
			}
			sb.WriteString(cma + v.column)
			pchar = " = "
			if isnl {
				pchar += "NULL"
			} else {
				if v.sqlstring {
					pchar += qb.dbEngineConstants.ParameterChar
					if qb.dbEngineConstants.ParameterInSequence {
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
			if v.skip && !v.forcenull {
				continue
			}
			pchar = "NULL"
			if !isNil(v.value) && !v.forcenull {
				if !v.sqlstring {
					pchar, _ = v.value.(string)
				} else {
					pchar = qb.dbEngineConstants.ParameterChar
					if qb.dbEngineConstants.ParameterInSequence {
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
				pchar = qb.dbEngineConstants.ParameterChar
				if qb.dbEngineConstants.ParameterInSequence {
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
			fbs, _ := qb.FilterFunc(paramcnt, qb.dbEngineConstants.ParameterChar, qb.dbEngineConstants.ParameterInSequence)
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
	if len(qb.ResultLimit) > 0 && qb.dbEngineConstants.ResultLimitPosition == REAR {
		sb.WriteString(" LIMIT " + qb.ResultLimit)
	}
	sb.WriteString(";")

	// build values
	args = make([]interface{}, 0, 15)
	for _, v := range qb.values {
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
		fbs, fbargs := qb.FilterFunc(paramcnt, qb.dbEngineConstants.ParameterChar, qb.dbEngineConstants.ParameterInSequence)
		if len(fbs) > 0 {
			args = append(args, fbargs...)
		}
	}

	query = sb.String()
	if qb.interpolateTables {
		sch := ``
		if qb.referenceMode {
			sch = qb.referenceModePrefix
			if !strings.HasSuffix(sch, "_") {
				sch += "_"
			}
		}
		// If there is a schema defined, it will prevail
		if qb.schema != "" {
			sch = qb.schema
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

func (qb *QueryBuilder) setColumnValue(index int, value interface{}, sqlString bool, defValue interface{}, matchToNull interface{}) *QueryBuilder {
	for i, v := range qb.values {
		if !strings.EqualFold(qb.columns[index].Name, v.column) {
			continue
		}
		qb.values[i].sqlstring = sqlString
		qb.values[i].defvalue = defValue
		qb.values[i].matchtonull = matchToNull
		qb.values[i].value = value
		return qb
	}
	qb.values = append(qb.values, queryValue{
		column:      qb.columns[index].Name,
		sqlstring:   sqlString,
		defvalue:    defValue,
		matchtonull: matchToNull,
		value:       value,
	})
	return qb
}

func isNil(value interface{}) bool {
	if value == nil {
		return true
	}
	if t := reflect.TypeOf(value); t == nil {
		return true
	}
	if v := reflect.ValueOf(value); v.IsZero() {
		if k := v.Kind(); k == reflect.Map ||
			k == reflect.Func ||
			k == reflect.Ptr ||
			k == reflect.Slice ||
			k == reflect.Interface {
			return v.IsNil()
		}
	}
	return false
}

// converts the value to a basic interface as nil or non-nil
func realValue(value interface{}) interface{} {
	if isNil(value) {
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
	case string, int, int8, int16, int32,
		int64, float32, float64, time.Time, bool,
		byte, []byte, ssd.Decimal:
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
	case *ssd.Decimal:
		ret = *t
	case dhl.VarChar, dhl.VarCharMax, dhl.NVarCharMax:
		ret = t
	}
	return
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
