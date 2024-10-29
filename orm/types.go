package orm

import (
	"context"
	"io"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/expr"
)

type Model interface {
	// PrimaryKeyVals returns the values of the primary key columns
	// If a single column, expr.Arg(col) is expected
	// If multiple columns, expr.ArgGroup(col1, col2, ...) is expected
	PrimaryKeyVals() bob.Expression
}

type Setter[T, InsertQ, UpdateQ any] interface {
	// SetColumns should return the column names that are set
	SetColumns() []string
	// Act as a mod for the insert query
	// the BeforeInsertHooks MUST be run here
	bob.Mod[InsertQ]
	// Return a mod for the update query
	// this should add "SET col1 = val1, col2 = val2, ..."
	UpdateMod() bob.Mod[UpdateQ]
}

type SchemaTable string

func (s SchemaTable) WriteSQL(ctx context.Context, w io.Writer, d bob.Dialect, start int) ([]any, error) {
	schema, _ := ctx.Value(CtxUseSchema).(string)
	return expr.Quote(schema, string(s)).WriteSQL(ctx, w, d, start)
}
