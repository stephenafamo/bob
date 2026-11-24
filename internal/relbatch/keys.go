// Package relbatch builds the ownership join used by generated batch loaders.
package relbatch

import (
	"slices"
	"strconv"
	"strings"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/clause"
	"github.com/stephenafamo/bob/expr"
	"github.com/stephenafamo/bob/mods"
)

func KeyNames(n int) []string {
	names := make([]string, n)
	for i := range names {
		names[i] = "k" + strconv.Itoa(i)
	}
	return names
}

// Group collects the input positions for each SQL key. source supplies keys
// with the same comparison semantics as the target columns.
func Group(q interface {
	AppendSelect(...any)
	AppendGroup(any)
}, source string, names []string, owners bob.Expression,
) {
	q.AppendSelect(expr.OP("AS", owners, expr.Quote("owners")))
	for _, name := range names {
		k := expr.Quote(source, name)
		q.AppendSelect(k)
		q.AppendGroup(k)
	}
}

func Alias(name string, reserved []string) string {
	for slices.ContainsFunc(reserved, func(s string) bool { return strings.EqualFold(name, s) }) {
		name += "_"
	}
	return name
}

// Join adds one owner list per matched key without multiplying outer rows.
func Join[Q interface{ AppendJoin(clause.Join) }](
	target clause.TableRef,
	columns, names, aliases []string,
	groups bob.Query,
) (bob.Mod[Q], bob.Expression) {
	alias := Alias("__bob_parents", aliases)
	on := make([]bob.Expression, len(columns))
	for i, col := range columns {
		on[i] = expr.OP("=", expr.Quote(target.Alias, col), expr.Quote(alias, names[i]))
	}
	return mods.Join[Q](clause.Join{Type: clause.InnerJoin, To: clause.TableRef{Expression: groups, Alias: alias}, On: on}),
		expr.Quote(alias, "owners")
}
