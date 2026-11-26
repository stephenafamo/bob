package orm

import (
	"context"
	"io"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/expr"
)

// RowMapping describes the schema-specific parts of refreshing a model.
// KeyColumns and KeyValues follow primary-key constraint order.
type RowMapping[T any, S ~[]T] struct {
	KeyColumns   expr.ColumnsExpr
	KeyValues    func(T) bob.Expression
	CopyColumns  func(T, T)
	CopyMatching func(S, ...T)
}

// In builds a primary-key predicate for a batch, matching nothing when empty.
func (m RowMapping[T, S]) In(rows S) bob.Expression {
	if len(rows) == 0 {
		return expr.Raw("NULL")
	}
	return expr.OP(
		"IN",
		m.KeyColumns.RowValue(),
		bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
			keys := make([]bob.Expression, len(rows))
			for i, row := range rows {
				keys[i] = m.KeyValues(row)
			}
			return bob.ExpressSlice(ctx, w, d, start, keys, "(", ", ", ")")
		}),
	)
}
