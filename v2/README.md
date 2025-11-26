# QueryBuilder v2

[![Go Reference](https://pkg.go.dev/badge/github.com/eaglebush/querybuilder/v2.svg)](https://pkg.go.dev/github.com/eaglebush/querybuilder/v2)
[![Go Report Card](https://goreportcard.com/badge/github.com/eaglebush/querybuilder)](https://goreportcard.com/report/github.com/eaglebush/querybuilder)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

Version 2.0 (2024.08.01)

QueryBuilder v2 is a redesigned SQL query builder with:

- Functional options for configuration
- EngineConstants abstraction for database engine behavior
- Tight integration with `di.DataInfo`
- A factory / spawn pattern for reusing configuration
- Reference mode support (prefixing objects for replicated / reference data)
- The same BuildWithCount convenience as v1, using subquery wrapping

---

## Installation

    go get github.com/eaglebush/querybuilder/v2

Import:

    import qb "github.com/eaglebush/querybuilder/v2"

---

## Design Overview

Key ideas:

- Create a `QueryBuilder` through `New` using a set of `Option` values.
- Use `EngineConstants` and `DataInfo` to configure engine-specific syntax (quotes, placeholders, limit style, etc.).
- Use `ValueOption` to configure how values behave (SQL string, default, match-to-null).
- Reuse settings by “spawning” new builders from a factory instance.
- Use `BuildWithCount()` to automatically generate a matching `COUNT(*)` query.

---

## Quick Start Example
```go
    import (
        "database/sql"

        qb "github.com/eaglebush/querybuilder/v2"
        di "github.com/eaglebush/datainfo"
    )

    func exampleSelect(db *sql.DB, info *di.DataInfo) error {
        builder := qb.NewSelect(
            "users",
            qb.DatabaseInfo(info),
            qb.Distinct(true),
            qb.ResultLimit("50"),
        ).
            AddColumn("Id").
            AddColumn("UserName").
            AddFilter("IsActive", true).
            AddOrder("UserName", qb.ASC)

        query, args, err := builder.Build()
        if err != nil {
            return err
        }

        rows, err := db.Query(query, args...)
        if err != nil {
            return err
        }
        defer rows.Close()

        return nil
    }
```
---

## Using BuildWithCount

`BuildWithCount` is available for SELECT commands and wraps your query as a subquery.
```go
    query, args, countQuery, err := builder.BuildWithCount()
    if err != nil {
        return err
    }

    var total int
    if err := db.QueryRow(countQuery, args...).Scan(&total); err != nil {
        return err
    }

    users := make([]User, 0, total)
    rows, err := db.Query(query, args...)
    if err != nil {
        return err
    }
    defer rows.Close()
```
Internally:

- Calls `Build()` a single time
- Strips trailing `;`
- Generates:
```sql
    SELECT COUNT(*) FROM (<your SELECT>) AS _qb_count;
```
Reuses `args` for both queries.

---

## Functional Options

Options are declared as:
```go
    type Option func(q *QueryBuilder) error
```
and include helpers like:

- `Source(name string)`
- `Schema(schema string)`
- `Command(ct CommandType)`
- `DatabaseInfo(di *di.DataInfo)`
- `Constants(ec EngineConstants)`
- `Distinct(yes bool)`
- `Interpolate(value bool)`
- `ReferenceMode(value bool)`
- `ReferenceModePrefix(prefix string)`
- `ResultLimit(value string)`
- `SkipNilWrite(skip bool)`

Example:
```go
    qb.New(
        qb.Source("users"),
        qb.Command(qb.SELECT),
        qb.DatabaseInfo(info),
        qb.Distinct(true),
        qb.ResultLimit("100"),
        qb.Interpolate(true),
        qb.SkipNilWrite(true),
    )
```
---

## Constructors

Convenience constructors for common use-cases:

- `NewSelect(dataObject string, opts ...Option) *QueryBuilder`
- `NewInsert(table string, opts ...Option) *QueryBuilder`
- `NewUpdate(table string, opts ...Option) *QueryBuilder`
- `NewDelete(table string, opts ...Option) *QueryBuilder`

Each of these:

- Sets `Source`
- Sets `CommandType`
- Applies additional options

Example:
```go
    sel := qb.NewSelect("users", qb.DatabaseInfo(info))
    ins := qb.NewInsert("users", qb.DatabaseInfo(info))
```
---

## Spawn Pattern

You can treat a `QueryBuilder` as a factory and “spawn” new builders that reuse its engine configuration.

Spawn helpers:

- `Spawn(builder QueryBuilder, opts ...Option) *QueryBuilder`
- `SpawnSelect(builder *QueryBuilder, dataObject string, opts ...Option) *QueryBuilder`
- `SpawnInsert(builder *QueryBuilder, table string, opts ...Option) *QueryBuilder`
- `SpawnUpdate(builder *QueryBuilder, table string, opts ...Option) *QueryBuilder`
- `SpawnDelete(builder *QueryBuilder, table string, opts ...Option) *QueryBuilder`

Example:
```go
    factory := qb.New(
        qb.DatabaseInfo(info),
        qb.Interpolate(true),
        qb.SkipNilWrite(true),
    )

    usersSel := qb.SpawnSelect(factory, "users").
        AddColumn("Id").
        AddColumn("UserName").
        AddFilter("IsActive", true)

    usersIns := qb.SpawnInsert(factory, "users").
        AddValue("UserName", "john.doe")
```
Both spawned builders share:

- Engine constants
- DataInfo configuration
- Interpolation behavior
- Skip-nil-write settings

But differ in:

- Source
- CommandType
- Columns, values, filters, etc.

---

## EngineConstants and DataInfo

`EngineConstants`:
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
The function:
```go
    func InitConstants(di *di.DataInfo) EngineConstants
```
Derives defaults from `DataInfo` if present, else uses fallbacks like:

- StringEnclosingChar: `'`
- StringEscapeChar: ``
- ParameterChar: `?`
- ReservedWordEscapeChar: `"`
- ParameterInSequence: false
- ResultLimitPosition: REAR

`DatabaseInfo(di *di.DataInfo)` sets:

- `qb.dbInfo = di`
- Rebuilds `dbEnConst` from it

---

## Value Options

`ValueOption` functions configure how a value is treated when calling `AddValue`.

Signature:
```go
    type ValueOption func(vo *ValueCompareOption) error
```
`ValueCompareOption`:
```go
    type ValueCompareOption struct {
        SQLString   bool
        Default     any
        MatchToNull any
    }
```
Helpers:

- `IsSqlString(indeed bool)`
- `Default(value any)`
- `MatchToNull(match any)`

Example:
```go
    builder.AddValue(
        "UserName",
        userPtr,
        qb.IsSqlString(true),
        qb.Default("N/A"),
        qb.MatchToNull(""),
    )
```
Meaning:

- Use a parameter for this value
- If nil → `"N/A"`
- If equals `""` → write `NULL`

---

## Filters, Ordering, Grouping

Similar to v1:

- `AddFilter(column string, value any)` – column with value
- `AddFilterExp(expr string)` – raw expression, no value
- `AddOrder(column string, order Sort)` – ORDER BY
- `AddGroup(group string)` – GROUP BY clause

`FilterFunc` can also be set, with signature:
```go
    FilterFunc func(offset int, char string, inSeq bool) ([]string, []any)
```
to add external filters and arguments.

---

## Interpolation and Reference Mode

If:

- `Interpolate(true)`
- `Schema("dbo")`
- `ReferenceMode(true)`
- `ReferenceModePrefix("ref")`

Then references like:
```sql
    FROM {Users}
```
become something like:
```sql
    FROM dbo.ref_Users
```
Priority:

- If `Schema` is set in `DataInfo`, it overrides `ReferenceModePrefix`.
- If `ReferenceMode` is off, prefix is not used.
- If `Interpolate` is false, `{...}` tokens are left as-is.

---

## More Details

For a complete listing of all fields, methods, and helper functions, see:

`API_REFERENCE.md` in the v2 directory.

---

## License

MIT License

Copyright (c) 2024 Eaglebush

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the “Software”), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED “AS IS”, WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
