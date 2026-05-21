package dialect

import (
	"context"
	"io"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/clause"
)

// needsFromItemParens reports whether a from_item must be parenthesized when
// written in a comma-separated FROM/USING list. JOIN binds tighter than comma
// in PostgreSQL, so an item with joins needs parens when other items follow.
func needsFromItemParens(items []clause.TableRef, item clause.TableRef) bool {
	return len(items) > 1 && len(item.Joins) > 0
}

func writeFromItemList(
	ctx context.Context, w io.StringWriter, d bob.Dialect, start int,
	args []any, items []clause.TableRef,
) ([]any, error) {
	for i, item := range items {
		if i > 0 {
			w.WriteString(", ")
		}

		if needsFromItemParens(items, item) {
			w.WriteString("(")
		}

		itemArgs, err := bob.Express(ctx, w, d, start+len(args), item)
		if err != nil {
			return nil, err
		}
		args = append(args, itemArgs...)

		if needsFromItemParens(items, item) {
			w.WriteString(")")
		}
	}

	return args, nil
}
