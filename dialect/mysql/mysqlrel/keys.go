// Package mysqlrel builds batch joins for generated MySQL relationships.
package mysqlrel

import (
	"slices"
	"strconv"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/clause"
	"github.com/stephenafamo/bob/dialect/mysql"
	"github.com/stephenafamo/bob/dialect/mysql/dialect"
	"github.com/stephenafamo/bob/dialect/mysql/sm"
	"github.com/stephenafamo/bob/dialect/mysql/vm"
	"github.com/stephenafamo/bob/expr"
	"github.com/stephenafamo/bob/internal/relbatch"
)

// KeyBinding selects how a generated batch compares one key component.
type KeyBinding int

const (
	Direct KeyBinding = iota // compare the original parameter to the physical column
	Signed
	Unsigned
	Character
)

// MatchParents groups typed VALUES before joining to the target. If a key
// cannot be represented losslessly in VALUES, probe the target index with
// the original column = ? predicates instead.
func MatchParents[P any](
	parents []*P,
	key func(*P) []any,
	target clause.TableRef,
	columns []string,
	bindings []KeyBinding,
	aliases ...string,
) (bob.Mod[dialect.SelectEditor], mysql.Expression) {
	names := relbatch.KeyNames(len(columns))
	var values []bob.Mod[dialect.ValuesEditor]
	var probes []bob.Mod[dialect.SelectEditor]
	var first mysql.SelectQuery
	direct := slices.Contains(bindings, Direct)
	for i, p := range parents {
		if p == nil {
			continue
		}
		pos := expr.Raw(strconv.Itoa(i))
		args := key(p)
		if direct {
			probe := probeKeys(target, columns, names, args, pos)
			if first.Type() == bob.QueryTypeUnknown {
				first = probe
			} else {
				probes = append(probes, sm.UnionAll(probe))
			}
			continue
		}
		row := make([]bob.Expression, 0, len(columns)+1)
		for j, v := range args {
			switch bindings[j] {
			case Signed:
				row = append(row, mysql.Cast(mysql.Arg(v), "SIGNED"))
			case Unsigned:
				row = append(row, unsignedArg(v))
			case Character:
				row = append(row, columnArg(v, target, columns[j]))
			}
		}
		values = append(values, vm.RowValue(append(row, pos)...))
	}
	var input clause.TableRef
	switch {
	case first.Type() != bob.QueryTypeUnknown:
		input = clause.TableRef{Expression: first.WithMods(probes...), Alias: "p"}
	case len(values) != 0:
		input = clause.TableRef{Expression: mysql.Values(values...), Alias: "p", Columns: append(names, "pos")}
	default:
		return sm.Where(mysql.Raw("0")), mysql.Raw("'[]'")
	}
	groups := mysql.Select(sm.From(input), relbatch.Group[dialect.SelectEditor]("p", names, mysql.Raw("CAST(JSON_ARRAYAGG(p.pos) AS CHAR)")))
	join, owners := relbatch.Join[dialect.SelectEditor](target, columns, names, aliases, groups)
	return join, mysql.Expression{}.New(owners)
}

// A parameter can match several distinct physical keys. Keep all those keys,
// but don't repeat one because multiple children carry it.
func probeKeys(target clause.TableRef, columns, names []string, args []any, pos bob.Expression) mysql.SelectQuery {
	return mysql.Select(sm.Distinct(), sm.From(target.Expression).As("t"), sm.Columns(expr.OP("AS", pos, expr.Quote("pos"))),
		bob.ModFunc[mysql.SelectEditor](func(e mysql.SelectEditor) {
			for j, v := range args {
				col := mysql.Quote("t", columns[j])
				e.AppendSelect(col.As(names[j]))
				e.AppendWhere(col.EQ(mysql.Arg(v)))
			}
		}))
}

// unsignedArg keeps a string-valued BIGINT UNSIGNED exact through binding.
// A prepared CAST(? AS UNSIGNED) can clamp the parameter to MaxInt64 before
// evaluating the cast. Bind it as text first, then convert the full digits.
func unsignedArg(value any) bob.Expression {
	return mysql.Cast(mysql.Cast(mysql.Arg(value), "CHAR"), "UNSIGNED")
}

// columnArg supplies the physical character column's collation. The scalar
// subquery reads no rows, so COALESCE returns the bound value, including NULL.
func columnArg(value any, target clause.TableRef, column string) bob.Expression {
	empty := mysql.Select(sm.Columns(mysql.Quote(column)), sm.From(target.Expression), sm.Where(mysql.Raw("FALSE")))
	return mysql.F("COALESCE", mysql.Arg(value), empty)
}
