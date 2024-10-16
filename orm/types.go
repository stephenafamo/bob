package orm

import (
	"context"
	"io"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/expr"
)

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

type SchemaTable string

func (s SchemaTable) WriteSQL(ctx context.Context, w io.Writer, d bob.Dialect, start int) ([]any, error) {
	schema, _ := ctx.Value(CtxUseSchema).(string)
	return expr.Quote(schema, string(s)).WriteSQL(ctx, w, d, start)
}
