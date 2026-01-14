package querybuilder

import (
	"log"

	di "github.com/eaglebush/datainfo"
)

// Option function for QueryBuilder
type Option func(q *QueryBuilder) error

// Constants are builder settings that follows the database engine settings.
func Constants(ec EngineConstants) Option {
	return func(q *QueryBuilder) error {
		q.dbEnConst = ec
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

// Distinct sets the option to return distinct values
func Distinct(yes bool) Option {
	return func(q *QueryBuilder) error {
		q.distinct = yes
		return nil
	}
}

// InsertReturn sets the last insert id query for Insert command. Query might include column name. Inline means that this query appends to the query without semi-colon.
func InsertReturn(sql string, inline bool) Option {
	return func(q *QueryBuilder) error {
		q.insertRetn = len(sql) > 0
		q.insertRetnSql = sql
		q.insertRetnInline = inline
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

// Source sets the table, view or stored procedure name
func Source(name string) Option {
	return func(q *QueryBuilder) error {
		q.Source = name
		return nil
	}
}
