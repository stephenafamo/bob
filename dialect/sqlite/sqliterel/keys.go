// Package sqliterel matches generated batch relationships using SQLite's
// comparison rules, including collations supplied by the application.
package sqliterel

import (
	"strconv"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/clause"
	"github.com/stephenafamo/bob/dialect/sqlite"
	"github.com/stephenafamo/bob/dialect/sqlite/dialect"
	"github.com/stephenafamo/bob/dialect/sqlite/sm"
	"github.com/stephenafamo/bob/expr"
	"github.com/stephenafamo/bob/internal/relbatch"
)

// MatchParents returns a join and a column containing the JSON array of
// matching input positions. Group on the physical key, with its collation,
// so multiple parents never multiply the outer rows (including for LIMIT).
func MatchParents[P any](
	parents []*P,
	key func(*P) []any,
	target clause.TableRef,
	columns []string,
	aliases ...string,
) (bob.Mod[*dialect.SelectQuery], sqlite.Expression) {
	values := sqlite.Values()
	for i, p := range parents {
		if p != nil {
			// Only the ordinal is a literal; all parent values are bound.
			values.Expression.AppendValues(expr.Raw(strconv.Itoa(i)), expr.Arg(key(p)...))
		}
	}
	if len(values.Expression.RowVals) == 0 {
		return sm.Where(sqlite.Raw("0")), sqlite.Raw("'[]'")
	}

	names := relbatch.KeyNames(len(columns))
	input := relbatch.Alias("__bob_input", aliases)
	// VALUES has no target-column affinity or collation. First select only the
	// distinct matching physical keys. Then match positions to those keys,
	// not to all of the children that carry them.
	inputKeys := sqlite.Select(sm.From(sqlite.Quote(input)))
	matched := sqlite.Select(sm.Distinct(), sm.From(target.Expression).As("t"))
	physical := make([]bob.Expression, len(columns))
	on := make([]bob.Expression, len(columns))
	for i, col := range columns {
		k := sqlite.Quote("t", col)
		physical[i] = k
		matched.Apply(sm.Columns(k.As(names[i])))
		inputKeys.Apply(sm.Columns(sqlite.Quote(names[i])))
		on[i] = sqlite.Quote("m", names[i]).EQ(sqlite.Quote("p", names[i]))
	}
	matched.Apply(sm.Where(sqlite.Group(physical...).OP("IN", inputKeys)))
	groups := sqlite.Select(sm.With(input, append([]string{"pos"}, names...)...).As(values),
		sm.From(matched).As("m"), sm.InnerJoin(sqlite.Quote(input)).As("p").On(on...))
	relbatch.Group(groups.Expression, "m", names, sqlite.Raw("'[' || group_concat(p.pos) || ']'"))
	join, owners := relbatch.Join[*dialect.SelectQuery](target, columns, names, aliases, groups)
	return join, sqlite.Expression{}.New(owners)
}
