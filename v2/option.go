package querybuilder

import (
	"log"

	di "github.com/eaglebush/datainfo"
)

type OptionID uint8

const (
	OPTID_UNSET                 OptionID = 0
	OPTID_CONSTANTS             OptionID = 1
	OPTID_COMMAND               OptionID = 2
	OPTID_DATABASE_INFO         OptionID = 3
	OPTID_DISTINCT              OptionID = 4
	OPTID_INSERT_RETURN         OptionID = 5
	OPTID_INTERPOLATE           OptionID = 6
	OPTID_REFERENCE_MODE        OptionID = 7
	OPTID_REFERENCE_MODE_PREFIX OptionID = 8
	OPTID_SCHEMA                OptionID = 9
	OPTID_SOURCE                OptionID = 10
	OPTID_VALUE                 OptionID = 11
	OPTID_COLUMN                OptionID = 12
	OPTID_RESULT_LIMIT          OptionID = 13
	OPTID_SKIP_NIL_WRITE        OptionID = 14
)

// Option function for QueryBuilder
type Option func(q *QueryBuilder) OptionID

// Constants are builder settings that follows the database engine settings.
func Constants(ec EngineConstants) Option {
	return func(q *QueryBuilder) OptionID {
		q.dbEnConst = ec
		return OPTID_CONSTANTS
	}
}

// Command sets the command of a query builder
func Command(ct CommandType) Option {
	return func(q *QueryBuilder) OptionID {
		q.CommandType = ct
		return OPTID_COMMAND
	}
}

// Config sets the database info
func DatabaseInfo(dnf *di.DataInfo) Option {
	return func(q *QueryBuilder) OptionID {
		q.dbInfo = dnf
		q.dbEnConst = InitConstants(dnf)
		return OPTID_DATABASE_INFO
	}
}

// Distinct sets the option to return distinct values
func Distinct(yes bool) Option {
	return func(q *QueryBuilder) OptionID {
		q.distinct = yes
		return OPTID_DISTINCT
	}
}

// InsertReturn sets the last insert id query for Insert command.
// Query might include column name.
// Inline means that this query appends to the query without semi-colon.
func InsertReturn(sql string, inline bool) Option {
	return func(q *QueryBuilder) OptionID {
		q.insertRetn = len(sql) > 0
		q.insertRetnSql = sql
		q.insertRetnInline = inline
		return OPTID_INSERT_RETURN
	}
}

// Interpolate converts all table name with {} around it will be prepended with schema and reference code prefix
func Interpolate(value bool) Option {
	return func(q *QueryBuilder) OptionID {
		q.intTbls = value
		return OPTID_INTERPOLATE
	}
}

// ReferenceMode enables the builder to generate query that adds a `ref` prefix to table names after the schema.
//
// This can be used in instances that the database object is just a reference populated by event source rather than user interaction.
//
// Warning: If the interpolation is set to off, this property is ignored.
func ReferenceMode(value bool) Option {
	return func(q *QueryBuilder) OptionID {
		if q.dbInfo == nil {
			q.dbInfo, _ = di.New()
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
		return OPTID_REFERENCE_MODE
	}
}

// ReferenceModePrefix changes the reference prefix to add to database object names when set in ReferenceMode
//
// Warning: If the interpolation is set to off, this property is ignored.
func ReferenceModePrefix(prefix string) Option {
	return func(q *QueryBuilder) OptionID {
		if prefix == "" {
			return OPTID_REFERENCE_MODE_PREFIX
		}
		if q.dbInfo == nil {
			q.dbInfo, _ = di.New()
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
		return OPTID_REFERENCE_MODE_PREFIX
	}
}

// Schema sets the schema of a query builder
func Schema(sch string) Option {
	return func(q *QueryBuilder) OptionID {
		if q.dbInfo == nil {
			q.dbInfo, _ = di.New()
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
		return OPTID_SCHEMA
	}
}

// Source sets the table, view or stored procedure name
func Source(name string) Option {
	return func(q *QueryBuilder) OptionID {
		q.Source = name
		return OPTID_SOURCE
	}
}

// Value adds a column-value data to the initialization process.
//
// It will be added to the data but it will be ignored when the command is SELECT and DELETE.
func Value(name string, value any, vcOpts ...ValueOption) Option {
	return func(q *QueryBuilder) OptionID {
		q.AddValue(name, value, vcOpts...)
		return OPTID_VALUE
	}
}

// Column adds a column data to the initialization process.
//
// It will be added to the data but it will be ignored when the command is INSERT, UPDATE and DELETE.
func Column(name string) Option {
	return func(q *QueryBuilder) OptionID {
		q.AddColumn(name)
		return OPTID_COLUMN
	}
}

// ResultLimit sets the result limit at initialization. ResultLimit can also be set at QueryBuilder ResultLimit field.
func ResultLimit(value string) Option {
	return func(q *QueryBuilder) OptionID {
		q.ResultLimit = value
		return OPTID_RESULT_LIMIT
	}
}

// SkipNilWrite sets the condition to skip nil columns when writing to table
func SkipNilWrite(skip bool) Option {
	return func(q *QueryBuilder) OptionID {
		q.skpNilWrCol = skip
		return OPTID_SKIP_NIL_WRITE
	}
}
