# QueryBuilder

[![Go Reference](https://pkg.go.dev/badge/github.com/eaglebush/querybuilder/.svg)](https://pkg.go.dev/github.com/eaglebush/querybuilder/)
[![Go Report Card](https://goreportcard.com/badge/github.com/eaglebush/querybuilder)](https://goreportcard.com/report/github.com/eaglebush/querybuilder)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

QueryBuilder  is a lightweight SQL query builder for Go.

It focuses on:

- Building SQL strings for SELECT, INSERT, UPDATE, DELETE
- Generating parameter lists compatible with database drivers
- Supporting multiple database engines via configuration (cfg.DatabaseInfo)
- Schema interpolation for tables using {TableName} tokens
- Convenience method BuildWithCount for SELECT COUNT(*) wrapping

---

## Overview

QueryBuilder  is structured around a central `QueryBuilder` struct.
You configure:

- CommandType (SELECT / INSERT / UPDATE / DELETE)
- Table or view name
- Columns and values
- Filters (WHERE)
- ORDER BY, GROUP BY
- Result limiting (TOP, LIMIT)
- Database-specific settings via cfg.DatabaseInfo

Once configured, you call:

- `Build()` – to get the SQL string and its parameter slice
- `BuildWithCount()` – for SELECT, to also get a `COUNT(*)` wrapper query using the same parameters

---

## Installation

Use Go modules:

    go get github.com/eaglebush/querybuilder/

Then import:

    import qb "github.com/eaglebush/querybuilder/"

---

## Basic Usage

### Creating a SELECT

```go
    import (
        "database/sql"

        qb  "github.com/eaglebush/querybuilder/"
        cfg "github.com/eaglebush/config"
    )

    func exampleSelect(db *sql.DB, info cfg.DatabaseInfo) error {
        builder := qb.NewSelect("users", info).
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

        // scan rows ...
        return nil
    }
```

`AddColumn` and `AddFilter` modify the `QueryBuilder` and return it, allowing simple chaining.

---

## INSERT / UPDATE / DELETE

### INSERT
```go
    builder := qb.NewInsert("users", info).
        AddValue("UserName", "john.doe", nil).
        AddValue("IsActive", true, nil)

    query, args, err := builder.Build()
```

The values are passed as parameters (depending on SQLString), and the generated SQL looks like:
```go
    INSERT INTO users (UserName, IsActive) VALUES (?, ?);
```
### UPDATE
```go
    builder := qb.NewUpdate("users", info, true). // skip nil write
        AddValue("UserName", "john.doe", nil).
        AddValue("IsActive", true, nil).
        AddFilter("Id", 123)

    query, args, err := builder.Build()
```
Generates something like:
```go
    UPDATE users SET UserName = ?, IsActive = ? WHERE Id = ?;
```
### DELETE
```go
    builder := qb.NewDelete("users", info).
        AddFilter("Id", 123)

    query, args, err := builder.Build()
```
Generates:
```sql
    DELETE
        FROM users
        WHERE Id = ?;
```
---

## Using ValueOption

`ValueOption` controls how INSERT/UPDATE values are evaluated:

Fields:

- `SQLString bool` – when true, the value is treated as a parameter (placeholder)
- `Default any` – if the original value is nil, this default is used
- `MatchToNull any` – when the primary value equals this, the column is forced to NULL

Example:
```go
    vo := &qb.ValueOption{
        SQLString:   true,
        Default:     "N/A",
        MatchToNull: "",
    }

    builder.AddValue("UserName", userNamePtr, vo)
```
Behavior:

- If `userNamePtr` is nil → value becomes `"N/A"`
- If `*userNamePtr` is `""` → the column is written as `NULL`
- Otherwise `*userNamePtr` is passed as a parameter value

---

## Filters and FilterFunc

### AddFilter
```go
    builder.AddFilter("IsActive", true)
```
This results in a condition similar to:
```sql
    WHERE IsActive = ?
```
and the value `true` is added to the args slice.

### AddFilterExp
```go
    builder.AddFilterExp("CreatedAt >= NOW() - INTERVAL 7 DAY")
```
Used when you want a raw expression that does not take a parameter:
```sql
    WHERE CreatedAt >= NOW() - INTERVAL 7 DAY
```
### FilterFunc

`FilterFunc` is a callback:
```go
    FilterFunc func(offset int, char string, inSeq bool) ([]string, []any)
```
It lets you append additional filter expressions and their parameter values from outside the builder.

Example pattern:
```go
    builder.FilterFunc = func(offset int, char string, inSeq bool) ([]string, []any) {
        return []string{"Status = " + char}, []any{"ACTIVE"}
    }
```
Those expressions and args are appended after normal filters.

---

## ORDER BY, GROUP BY, LIMIT

### ORDER BY
```go
    builder.
        AddOrder("UserName", qb.ASC).
        AddOrder("CreatedAt", qb.DESC)
```
Generates:
```sql
    ORDER BY UserName ASC, CreatedAt DESC
```
### GROUP BY
```go
    builder.AddGroup("Role").AddGroup("Department")
```
Generates:
```sql
    GROUP BY Role, Department
```
### LIMIT / TOP

Control:

- `ResultLimitPosition` – FRONT or REAR
   - FRONT → used for dialects like `SELECT TOP n`
   - REAR → used for dialects like `LIMIT n`
- `ResultLimit` – the string value (e.g. `"10"`)

Example (using REAR):
```go
    builder.ResultLimitPosition = qb.REAR
    builder.ResultLimit = "10"
```
Generates:
```sql
    SELECT ...
    FROM ...
    LIMIT 10;
```
---

## Interpolation of Table Names

If `InterpolateTables` is true, any table token written as `{TableName}` will be expanded with a schema prefix.

Example:

- `TableName = "{Users}"`
- `Schema = "dbo"`

Query:
```sql
    SELECT Id, UserName
    FROM {Users};
```
Becomes:
```sql
    SELECT Id, UserName
    FROM dbo.Users;
```
The function `InterpolateTable(sql, schema)` performs the replacement.

---

## BuildWithCount

`BuildWithCount()` is a convenience method for SELECT statements.

Signature:
```go
    func (qb *QueryBuilder) BuildWithCount() (
        query string,
        args []any,
        countQuery string,
        err error,
    )
```
Usage:
```go
    builder := qb.NewSelect("users", info).
        AddColumn("Id").
        AddColumn("UserName").
        AddFilter("IsActive", true)

    query, args, countQuery, err := builder.BuildWithCount()
    if err != nil {
        // handle
    }

    // Get total count
    var total int
    if err := db.QueryRow(countQuery, args...).Scan(&total); err != nil {
        // handle
    }

    // Preallocate based on count
    users := make([]User, 0, total)

    // Run main query
    rows, err := db.Query(query, args...)
```
How it works:

- First calls `Build()` once
- Strips any trailing `;` and whitespace
- Wraps your SELECT as a subquery:
```sql
    SELECT COUNT(*) FROM (<your SELECT>) AS _qb_count;
```
- Reuses the same args slice

This ensures:

- No duplication of builder logic
- No parameter re-numbering issues
- All DISTINCT, GROUP BY, expressions, etc. are preserved

---

## More Details

For a complete listing of all fields, methods, and helper functions, see:

`API_REFERENCE.md` in the  directory.

---

## License

MIT License

Copyright (c) 2024-2025 eaglebush

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
