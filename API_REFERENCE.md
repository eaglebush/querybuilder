# QueryBuilder – API Reference

This document describes all public types, fields, and methods in of the QueryBuilder package, plus the most important internal helpers.

Package
-------

    package querybuilder

Import path:

    github.com/eaglebush/querybuilder

---

## Types

### type Command
```go
    type Command uint8
```
Represents the kind of SQL command being built.

Constants:
```go
    const (
        SELECT Command = 0
        INSERT Command = 1
        UPDATE Command = 2
        DELETE Command = 3
    )
```
Used in:

- `QueryBuilder.CommandType`
- Constructors such as `NewQueryBuilderWithCommandType`

---

### type Sort
```go
    type Sort uint8
```
Represents sort order in ORDER BY clauses.

Constants:
```go
    const (
        ASC  Sort = 0
        DESC Sort = 1
    )
```
Used by:

- `AddOrder(column string, order Sort)`

---

### type Limit
```go
    type Limit uint8
```
Represents which part of the query contains the limit clause.

Constants:
```go
    const (
        FRONT Limit = 0
        REAR  Limit = 1
    )
```
Front is intended for dialects like:
```sql
    SELECT TOP 10 ...
```
Rear is intended for dialects like:
```sql
    ... LIMIT 10
```
Used in:

- `QueryBuilder.ResultLimitPosition`

---

### errors
```go
    var (
        ErrNoTableSpecified  = errors.New("table or view was not specified")
        ErrNoColumnSpecified = errors.New("no columns were specified")
    )
```
Returned by `Build()` when:

- No table or view name was set
- No columns were specified for non-DELETE commands

---

### type ValueOption
```go
    type ValueOption struct {
        SQLString   bool
        Default     any
        MatchToNull any
    }
```
Used by:

- `AddValue(Name string, Value any, vo *ValueOption)`

Fields:

- `SQLString` – if true, the value is treated as an SQL string parameter; a placeholder is used and the value is added to `args`.
- `Default` – if the primary value is nil, this default is used instead.
- `MatchToNull` – if the primary value equals this value, the column is forced to `NULL`.

---

### type QueryColumn
```go
    type QueryColumn struct {
        Name   string
        Length int
    }
```
Represents a column metadata entry configured by:

- `addColumn(name string, length int)` (internal)
- `AddColumn` and `AddColumnFixed`

Fields:

- `Name` – column name or expression used in the query.
- `Length` – optional length hint; not used in SQL generation directly, but kept for consistency with original design.

---

### type queryValue
```go
    type queryValue struct {
        column      string
        value       any
        defvalue    any
        matchtonull any
        sqlstring   bool
        skip        bool
        forcenull   bool
    }
```
Internal struct representing a column’s value.

Fields:

- `column` – column name (must match an entry in `Columns`).
- `value` – actual value to be written or compared.
- `defvalue` – default value when primary value is nil.
- `matchtonull` – condition to force NULL when values match.
- `sqlstring` – when true, uses a parameter placeholder in SQL.
- `skip` – indicates whether the column should be skipped when writing.
- `forcenull` – indicates that the column should be forced to `NULL`.

---

### type queryFilter
```go
    type queryFilter struct {
        expression    string
        value         any
        containsvalue bool
    }
```
Internal representation of a `WHERE` filter.

Fields:

- `expression` – either a column name or a raw SQL expression.
- `value` – parameter value if `expression` is a column name.
- `containsvalue` – when true, the expression is assumed to contain its own operator/value and will not be appended with `= ?` or `IS NULL`.

Created by:

- `AddFilter(column, value)` – `containsvalue` is false.
- `AddFilterExp(expr)` – `containsvalue` is true, `value` is nil.

---

### type querySort
```go
    type querySort struct {
        column string
        order  Sort
    }
```
Internal representation of ORDER BY elements.

Fields:

- `column` – column or expression to sort by.
- `order` – ascending or descending (ASC or DESC).

Created by:

- `AddOrder(column string, order Sort)`

---

### type QueryBuilder
```go
    type QueryBuilder struct {
        TableName              string
        CommandType            Command
        Distinct               bool
        Columns                []QueryColumn
        Values                 []queryValue
        Order                  []querySort
        Group                  []string
        Filter                 []queryFilter
        StringEnclosingChar    string
        StringEscapeChar       string
        ReservedWordEscapeChar string
        ParameterChar          string
        ParameterInSequence    bool
        SkipNilWriteColumn     bool
        ResultLimitPosition    Limit
        ResultLimit            string
        InterpolateTables      bool
        Schema                 string
        ParameterOffset        int
        FilterFunc             func(offset int, char string, inSeq bool) ([]string, []any)
        dbInfo                 *cfg.DatabaseInfo
    }
```
Fields:

- `TableName` – table or view name
- `CommandType` – SELECT, INSERT, UPDATE, DELETE
- `Distinct` – when true, SELECT uses `DISTINCT`
- `Columns` – configured by AddColumn / AddColumnFixed / AddValue
- `Values` – internal per-column values and their options
- `Order` – ORDER BY clauses
- `Group` – GROUP BY columns
- `Filter` – WHERE filters
- `StringEnclosingChar` – usually `'`
- `StringEscapeChar` – escape char for strings
- `ReservedWordEscapeChar` – for escaping reserved identifiers
- `ParameterChar` – placeholder prefix, e.g. `?`, `@p`
- `ParameterInSequence` – when true, generates `@p0`, `@p1`, etc.
- `SkipNilWriteColumn` – when true, nil columns are not written in INSERT/UPDATE
- `ResultLimitPosition` – FRONT or REAR
- `ResultLimit` – textual limit value, e.g. `"10"`
- `InterpolateTables` – if true, `{Table}` tokens are interpolated with schema
- `Schema` – schema override; if set, takes precedence over `dbInfo.Schema`
- `ParameterOffset` – starting offset for parameter numbering
- `FilterFunc` – optional callback for additional filter expressions and args
- `dbInfo` – optional `cfg.DatabaseInfo` providing database engine settings

---

## Constructors and Factory Functions

### func NewQueryBuilder(table string) *QueryBuilder

Creates a new `QueryBuilder` with:

- Given table name
- Default string and parameter configuration (quote char, escape char, etc.)
- `CommandType` defaulting to `SELECT`

    qb := querybuilder.NewQueryBuilder("users")

---

### func NewQueryBuilderWithCommandType(table string, commandType Command) *QueryBuilder

Creates a new `QueryBuilder` with:

- Given table name
- Specified command type

Example:
```go
    qb := querybuilder.NewQueryBuilderWithCommandType("users", querybuilder.UPDATE)
```
---

### func NewQueryBuilderBare() *QueryBuilder

Creates a new `QueryBuilder` with no table name, default constants and empty fields.

You typically use this when you want to set everything manually.

---

### func NewQueryBuilderWithConfig(table string, commandType Command, config cfg.DatabaseInfo) *QueryBuilder

Creates a `QueryBuilder` that uses database info from `cfg.DatabaseInfo`.

This config defines:

- `StringEnclosingChar`
- `StringEscapeChar`
- `ParameterPlaceholder`
- `ParameterInSequence`
- `ReservedWordEscapeChar`
- `InterpolateTables`

And sets `TableName`, `CommandType`, and default `ResultLimitPosition` (REAR).

---

### func NewSelect(table string, config cfg.DatabaseInfo) *QueryBuilder

Shortcut for:

- `NewQueryBuilderWithConfig(table, SELECT, config)`

---

### func NewInsert(table string, config cfg.DatabaseInfo) *QueryBuilder

Shortcut for:

- `NewQueryBuilderWithConfig(table, INSERT, config)`

---

### func NewUpdate(table string, config cfg.DatabaseInfo, skipnull bool) *QueryBuilder

Shortcut for:

- `NewQueryBuilderWithConfig(table, UPDATE, config)`
- Sets `SkipNilWriteColumn` to `skipnull`

This is typically used to avoid writing NULL values to the table when the column is nil.

---

### func NewDelete(table string, config cfg.DatabaseInfo) *QueryBuilder

Shortcut for:

- `NewQueryBuilderWithConfig(table, DELETE, config)`

---

## Column and Value Methods

### func (*QueryBuilder) AddColumn(name string) *QueryBuilder

Adds a column for SELECT / INSERT / UPDATE.

- For DELETE commands, does nothing and returns the builder.
- Uses `addColumn(name, 255)` internally and sets a default value (nil) with `sqlstring = true`.

Usage:

    qb.AddColumn("Id").AddColumn("UserName")

---

### func (*QueryBuilder) AddColumnFixed(name string, length int) *QueryBuilder

Same as `AddColumn`, but allows specifying the column length.

---

### func (*QueryBuilder) AddValue(name string, value any, vo *ValueOption) *QueryBuilder

Adds or updates a column and associates a value with it.

Evaluates `ValueOption`:

- `SQLString` – treat as parameter or inline string
- `Default` – default value when primary is nil
- `MatchToNull` – forces `NULL` when matched

Used primarily in INSERT and UPDATE.

---

### func (*QueryBuilder) SetColumnValue(name string, value any) *QueryBuilder

Updates the value for an existing column in `Values`:

- Looks for a `queryValue` whose `column` name matches
- Updates its `value`, `sqlstring`, `defvalue`, and `matchtonull` using internal logic

If the column is not present, the builder is returned unchanged.

---

### internal func (qb *QueryBuilder) addColumn(name string, length int) int

- Returns the index of the column in `qb.Columns`
- Adds a new `QueryColumn` if not present

---

### internal func (qb *QueryBuilder) setColumnValue(index int, value any, sqlString bool, defValue, matchToNull any) *QueryBuilder

Handles adding or updating a `queryValue` corresponding to `qb.Columns[index]`.

---

## Filter and Sorting Methods

### func (*QueryBuilder) AddFilter(column string, value any) *QueryBuilder

Appends a `queryFilter` where:

- `expression = column`
- `value = value`
- `containsvalue = false`

Generated SQL will use `expression = ?` or `expression IS NULL`.

---

### func (*QueryBuilder) AddFilterExp(expr string) *QueryBuilder

Appends a `queryFilter` where:

- `expression = expr`
- `value = nil`
- `containsvalue = true`

The expression is inserted as-is in the WHERE clause.

---

### func (*QueryBuilder) AddOrder(column string, order Sort) *QueryBuilder

Appends a `querySort` entry for ORDER BY.

Usage:

    qb.AddOrder("UserName", ASC)

---

### func (*QueryBuilder) AddGroup(group string) *QueryBuilder

Adds a column or expression to group by.

---

## Build Methods

### func (*QueryBuilder) Build() (query string, args []any, err error)

Builds the SQL string and the corresponding parameter slice.

Validation:

- Fails with `ErrNoTableSpecified` when `TableName` is empty
- Fails with `ErrNoColumnSpecified` when no columns and not a DELETE command

Process:

1. Normalize `Values` and `Filter` using `realvalue`
2. Build command header (SELECT/INSERT/UPDATE/DELETE)
3. Build column list or SET clause
4. Build INSERT VALUES placeholders
5. Build WHERE clause from `Filter` and `FilterFunc`
6. Build ORDER BY and GROUP BY
7. Add LIMIT or TOP based on `ResultLimitPosition`
8. Collect args from column values and filters
9. Interpolate tables if `InterpolateTables` is true

Returns:

- `query` – the SQL string including `;`
- `args` – ordered parameter values
- `err` – any validation error

---

### func (*QueryBuilder) BuildWithCount() (query string, args []any, countQuery string, err error)

Builds:

- The main `SELECT` query and args
- A wrapper `COUNT(*)` query using the same args

Behavior:

- Only valid for SELECT; otherwise returns an error
- Internally calls `Build()` once
- Trims trailing `;` and whitespace
- Generates:

    SELECT COUNT(*) FROM (<main select>) AS _qb_count;

`args` is reused for both queries.

---

## String and Table Utility Methods

### func (*QueryBuilder) Escape(value string) string

Escapes the string enclosing character in `value` using `StringEscapeChar`.

---

### func ParseReserveWordsChars(ec string) []string

Parses the reserved word escape characters string into a pair of opening and closing characters.

Behavior:

- If length is 1 → returns `[ec, ec]`
- If length >= 2 → returns `[ec[0:1], ec[1:2]]`
- Otherwise → defaults to `[""", """]`

---

### func InterpolateTable(sql string, schema string) string

Replaces tokens of the form `{TableName}` with `schema.TableName` if `schema` is not empty.

Uses a regular expression to match the brace-wrapped identifiers.

---

## Helper Functions

### func isNil(value any) bool

Determines if a value is logically nil, handling pointers, interfaces, maps, slices, etc.

---

### func realvalue(value any) any

Normalizes values to basic Go types or nil.

- If the value is nil or logically nil → returns nil
- If the value is a `*any` pointer, dereferences once and uses `getv`
- Otherwise passes the value to `getv`

---

### func getv(input any) any

Returns the underlying value for basic scalar types and pointers to those, including:

- `string, int, int8, int16, int32, int64`
- `float32, float64`
- `time.Time, bool`
- `byte, []byte`
- `ssd.Decimal`
- `dhl.VarChar, dhl.VarCharMax, dhl.NVarCharMax`

For pointer types, returns the dereferenced value if non-nil, otherwise nil.

---

This concludes the API reference.
