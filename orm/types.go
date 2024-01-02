package orm

import "github.com/stephenafamo/bob"

type Table interface {
	// PrimaryKeyVals returns the values of the primary key columns
	// If a single column, expr.Arg(col) is expected
	// If multiple columns, expr.ArgGroup(col1, col2, ...) is expected
	PrimaryKeyVals() bob.Expression
}

type Setter[T any, InsertQ any, UpdateQ any] interface {
	// SetColumns should return the column names that are set
	SetColumns() []string
	// Overwrite the values in T with the set values in the setter
	Overwrite(T)
	// Act as a mod for the update query
	bob.Mod[UpdateQ]
	// Return a mod for the insert query
	InsertMod() bob.Mod[InsertQ]
}
