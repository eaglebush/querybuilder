package querybuilder

// ValueCompareOption options for adding values
type (
	ValueCompareOption struct {
		SQLString   bool // Sets if the value is an SQL string. When true, this value is enclosed by the database client in single quotes to represent as string
		Default     any  // When set to non-nil, this is the default value when the value encounters a nil
		MatchToNull any  // When the primary value matches with this value, the resulting value will be set to NULL
	}

	ValueOption func(vo *ValueCompareOption) error
)

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
