package orm

import (
	"context"
	"io"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/expr"
)

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

func (s SchemaTable) WriteSQL(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
	schema, _ := ctx.Value(CtxUseSchema).(string)
	return expr.Quote(schema, string(s)).WriteSQL(ctx, w, d, start)
}

type ArgWithPosition struct {
	Name       string
	Start      int
	Stop       int
	Expression bob.Expression
}
