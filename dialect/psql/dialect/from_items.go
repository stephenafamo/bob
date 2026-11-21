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

// expressFromItems writes a comma-separated list of from_items and returns
// only the newly generated arguments, following the contract of
// [bob.ExpressIf] and [bob.ExpressSlice]. start must be the placeholder
// offset the first item begins at.
func expressFromItems(
	ctx context.Context, w io.StringWriter, d bob.Dialect, start int,
	items []clause.TableRef,
) ([]any, error) {
	var args []any
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
