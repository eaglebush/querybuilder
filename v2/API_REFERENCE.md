# QueryBuilder v2 – API Reference

Package
-------

    package querybuilder

Import path:

    github.com/eaglebush/querybuilder/v2

---

## Top-Level Types

### type CommandType
```go
    type CommandType uint8
```
Represents SQL command type.

Constants:
```go
    const (
        SELECT CommandType = 0
        INSERT CommandType = 1
        UPDATE CommandType = 2
        DELETE CommandType = 3
    )
```
---

### type Sort
```go
    type Sort uint8
```
Sort order for ORDER BY.

Constants:
```go
    const (
        ASC  Sort = 0
        DESC Sort = 1
    )
```
---

### type Limit
```go
    type Limit uint8
```
Represents where the limit clause goes.

Constants:
```go
    const (
        FRONT Limit = 0
        REAR  Limit = 1
    )
```
---

### errors
```go
    var (
        ErrNoTableSpecified  = errors.New("table or view was not specified")
        ErrNoColumnSpecified = errors.New("no columns were specified")
    )
```
---

### type Option
```go
    type Option func(q *QueryBuilder) error
```
Configures a new or spawned `QueryBuilder` with a functional option.

---

### type ValueOption
```go
    type ValueOption func(vo *ValueCompareOption) error
```
Configures behavior for a specific value comparison/addition with `AddValue`.

---

### type ValueCompareOption
```go
    type ValueCompareOption struct {
        SQLString   bool
        Default     any
        MatchToNull any
    }
```
Fields:

- `SQLString` – treat value as SQL parameter
- `Default` – default when primary value is nil
- `MatchToNull` – when primary value equals this, write NULL

---

### type QueryColumn
```go
    type QueryColumn struct {
        Name   string
        Length int
    }
```
Represents a column in the builder.

---

### type EngineConstants
```go
    type EngineConstants struct {
        StringEnclosingChar    string
        StringEscapeChar       string
        ReservedWordEscapeChar string
        ParameterChar          string
        ParameterInSequence    bool
        ResultLimitPosition    Limit
    }
```
Fields:

- `StringEnclosingChar` – e.g. `'`
- `StringEscapeChar` – escape char, e.g. ``
- `ReservedWordEscapeChar` – e.g. `"`
- `ParameterChar` – placeholder char, e.g. `?` or `@p`
- `ParameterInSequence` – whether to number parameters
- `ResultLimitPosition` – FRONT or REAR

---

### type queryValue

Internal representation of a column value:
```go
    type queryValue struct {
        column      string
        value       any
        defValue    any
        matchToNull any
        sqlstring   bool
        skip        bool
        forceNull   bool
    }
```
---

### type queryFilter

Internal WHERE filter:
```go
    type queryFilter struct {
        expression    string
        value         any
        containsValue bool
    }
```
---

### type querySort

Internal ORDER BY component:
```go
    type querySort struct {
        column string
        order  Sort
    }
```
---

### type QueryBuilder
```go
    type QueryBuilder struct {
        // Public fields
        Source          string
        CommandType     CommandType
        Filter          []queryFilter
        ResultLimit     string
        ParameterOffset int
        FilterFunc      func(offset int, char string, inSeq bool) ([]string, []any)

        // Private fields
        skpNilWrCol bool
        dbEnConst   EngineConstants
        intTbls     bool
        order       []querySort
        group       []string
        columns     []QueryColumn
        values      []queryValue
        dbInfo      *di.DataInfo
        distinct    bool
    }
```
Important public fields:

- `Source` – table, view, or query name
- `CommandType` – SELECT, INSERT, UPDATE, DELETE
- `Filter` – WHERE filters
- `ResultLimit` – textual limit value
- `ParameterOffset` – starting parameter index
- `FilterFunc` – optional extra filters callback

Private fields are managed internally and not meant to be mutated directly.

---

## Constructors

### func New(options ...Option) *QueryBuilder

Creates a new builder with defaults:

- `dbEnConst = InitConstants(nil)`
- `intTbls = true`
- `skpNilWrCol = true`
- `ResultLimit = ""`

Applies all given options. If `dbInfo` is still nil at the end, it allocates a default `DataInfo` and copies engine constants into it.

---

### func Spawn(builder QueryBuilder, options ...Option) *QueryBuilder

Creates a new builder using an existing builder as a template.

Copies:

- `dbEnConst`
- `skpNilWrCol`
- `intTbls`
- Resets `ResultLimit` to empty

Applies options. If `dbInfo` is nil, a new default one is created.

---

### func InitConstants(di *di.DataInfo) EngineConstants

Builds `EngineConstants` from `DataInfo`:

- If `di` is non-nil and its fields are set, they override defaults.
- Otherwise, defaults are used.

---

### NewSelect / NewInsert / NewUpdate / NewDelete

Convenience constructors:
```go
    func NewSelect(dataObject string, opts ...Option) *QueryBuilder
    func NewInsert(table string, opts ...Option) *QueryBuilder
    func NewUpdate(table string, opts ...Option) *QueryBuilder
    func NewDelete(table string, opts ...Option) *QueryBuilder
```
They append `Source(...)` and `Command(...)` to the provided options internally.

---

### SpawnSelect / SpawnInsert / SpawnUpdate / SpawnDelete

Spawn variants that take an existing builder as a factory:
```go
    func SpawnSelect(builder *QueryBuilder, dataObject string, opts ...Option) *QueryBuilder
    func SpawnInsert(builder *QueryBuilder, table string, opts ...Option) *QueryBuilder
    func SpawnUpdate(builder *QueryBuilder, table string, opts ...Option) *QueryBuilder
    func SpawnDelete(builder *QueryBuilder, table string, opts ...Option) *QueryBuilder
```
---

## Options

### Distinct(yes bool) Option

Sets `q.distinct = yes`.

---

### Source(name string) Option

Sets `q.Source = name`.

---

### Schema(sch string) Option

Ensures `dbInfo` is not nil, then sets `dbInfo.Schema` to `sch`.

---

### Command(ct CommandType) Option

Sets `q.CommandType = ct`.

---

### DatabaseInfo(dnf *di.DataInfo) Option

Sets:

- `q.dbInfo = dnf`
- `q.dbEnConst = InitConstants(dnf)`

---

### Constants(ec EngineConstants) Option

Sets `q.dbEnConst = ec`.

---

### Interpolate(value bool) Option

Enables or disables table interpolation: `q.intTbls = value`.

---

### ReferenceMode(value bool) Option

Ensures `dbInfo` is non-nil, then sets `dbInfo.ReferenceMode`.

---

### ReferenceModePrefix(prefix string) Option

Ensures `dbInfo` is non-nil, then sets `dbInfo.ReferenceModePrefix`.

---

### ResultLimit(value string) Option

Sets `q.ResultLimit = value`.

---

### SkipNilWrite(skip bool) Option

Sets `q.skpNilWrCol = skip`.

---

## ValueOption Helpers

### IsSqlString(indeed bool) ValueOption

Sets `vco.SQLString = indeed`.

---

### Default(value any) ValueOption

Sets `vco.Default = value`.

---

### MatchToNull(match any) ValueOption

Sets `vco.MatchToNull = match`.

---

## Column and Value Methods

### func (*QueryBuilder) AddColumn(name string) *QueryBuilder

Adds a column with default length 255, unless `CommandType` is DELETE.

---

### func (*QueryBuilder) AddColumnFixed(name string, length int) *QueryBuilder

Adds a column with a specific length.

---

### func (*QueryBuilder) AddValue(name string, value any, vcOpts ...ValueOption) *QueryBuilder

Applies `ValueOption` to a `ValueCompareOption`, then calls `setColumnValue` for the given name, with:

- `SQLString`
- `Default`
- `MatchToNull`

---

### func (*QueryBuilder) SetColumnValue(name string, value any) *QueryBuilder

Updates the value for an existing column entry if found in `qb.values`.

---

## Filter / Order / Group

### func (*QueryBuilder) AddFilter(column string, value any) *QueryBuilder

Adds a filter with column and value, `containsValue = false`.

---

### func (*QueryBuilder) AddFilterExp(expr string) *QueryBuilder

Adds a filter expression with `containsValue = true` and `value = nil`.

---

### func (*QueryBuilder) AddOrder(column string, order Sort) *QueryBuilder

Adds an ORDER BY component.

---

### func (*QueryBuilder) AddGroup(group string) *QueryBuilder

Adds a GROUP BY component.

---

## Build and BuildWithCount

### func (*QueryBuilder) Build() (query string, args []any, err error)

Process:

1. Validate `Source` and `columns`.
2. Normalize `values` and `Filter` with `realValue`.
3. Build SQL for SELECT/INSERT/UPDATE/DELETE using `dbEnConst`.
4. Build WHERE from `Filter` and `FilterFunc`.
5. Build ORDER BY, GROUP BY, LIMIT.
6. Populate `args` from values and filter values.
7. Apply interpolation if `intTbls` is true.
8. Update `ParameterOffset`.

---

### func (*QueryBuilder) BuildWithCount() (query string, args []any, countQuery string, err error)

Only valid for SELECT.

Calls `Build()` once, then:

- Trims whitespace and trailing `;`
- Wraps:

    SELECT COUNT(*) FROM (<query>) AS _qb_count;

Returns the resulting `countQuery` sharing the same `args`.

---

## Utility and Helper Functions

### func realValue(value any) any

Normalizes values similarly to v1:

- Returns nil if logically nil
- Dereferences `*any` once and passes to `getv`
- Otherwise passes directly to `getv`

---

### func isNil(value any) bool

Checks if a value is nil, including pointer, interface, map, slice, func, or chan.

---

### func getv(input any) any

Dereferences common scalar pointers and returns basic scalar values:

- Strings, integers, floats, bool, time.Time, byte, []byte, ssd.Decimal, dhl.VarChar, dhl.VarCharMax, dhl.NVarCharMax.

---

### func ParseReserveWordsChars(ec string) []string

Same as v1: returns a 2-element slice of escape characters.

---

### func InterpolateTable(sql string, schema string) string

Replaces `{TableName}` tokens with `schema.TableName`, or optionally `refPrefix_TableName` when using reference mode, depending on how it’s called from `Build`.

---

This concludes the v2 API reference.
