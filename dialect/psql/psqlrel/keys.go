// Package psqlrel builds PostgreSQL batch filters for generated relationships.
package psqlrel

import (
	"slices"
	"strconv"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/clause"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/dialect"
	"github.com/stephenafamo/bob/dialect/psql/sm"
	"github.com/stephenafamo/bob/expr"
	"github.com/stephenafamo/bob/internal/relbatch"
	"github.com/stephenafamo/bob/types/pgtypes"
)

// BatchKey holds a KeyValues array in non-nil parent order, including duplicates.
type BatchKey struct {
	Column string
	Values interface{ Index(int) any }
}

// MatchParents returns the non-nil parents and a join whose owner indexes
// refer to that slice. WITH ORDINALITY numbers the scalar key tuples in order.
func MatchParents[P any](
	parents []*P,
	target clause.TableRef,
	keys []BatchKey,
	aliases ...string,
) ([]*P, bob.Mod[*dialect.SelectQuery], psql.Expression) {
	return matchParents(parents, target, keys, unnestInput, aliases)
}

// MatchParentsWithArrayKey uses VALUES so that each array-valued key stays in
// one cell, even when the input arrays have different lengths.
func MatchParentsWithArrayKey[P any](
	parents []*P, target clause.TableRef, keys []BatchKey, aliases ...string,
) ([]*P, bob.Mod[*dialect.SelectQuery], psql.Expression) {
	return matchParents(parents, target, keys, valuesInput, aliases)
}

func matchParents[P any](
	parents []*P, target clause.TableRef, keys []BatchKey,
	inputFor func(clause.TableRef, []BatchKey, int) clause.TableRef, aliases []string,
) ([]*P, bob.Mod[*dialect.SelectQuery], psql.Expression) {
	// KeyValues skips nils. Keep the same order without modifying the
	// caller's slice, and reuse it when there are no nils.
	if gap := slices.Index(parents, nil); gap >= 0 {
		dense := make([]*P, gap, len(parents)-1)
		copy(dense, parents[:gap])
		for _, p := range parents[gap+1:] {
			if p != nil {
				dense = append(dense, p)
			}
		}
		parents = dense
	}
	if len(parents) == 0 {
		return parents, sm.Where(psql.Raw("FALSE")), psql.Raw("'[]'::text")
	}
	columns, names := make([]string, len(keys)), relbatch.KeyNames(len(keys))
	for i, k := range keys {
		columns[i] = k.Column
	}
	input := inputFor(target, keys, len(parents))
	groups := psql.Select(sm.From(input.As("p", append(names, "pos")...)))
	relbatch.Group(groups.Expression, "p", names, psql.Raw("json_agg(p.pos - 1 ORDER BY p.pos)::text"))
	// An outer FOR UPDATE locks the physical relationship rows. PostgreSQL
	// does not propagate it into a CTE (grouped subqueries cannot be locked).
	ownersTable := psql.Select(sm.With("owners").As(groups), sm.From("owners"))
	join, owners := relbatch.Join[*dialect.SelectQuery](target, columns, names, aliases, ownersTable)
	return parents, join, psql.Expression{}.New(owners)
}

// Infer the parameter type from the column without reading any rows.
// An explicit CHARACTER[] cast, for example, would truncate its elements.
func emptyColumn(target clause.TableRef, col string) bob.Query {
	return psql.Select(sm.Columns(psql.Quote(col)), sm.From(target.Expression), sm.Where(psql.Raw("FALSE")))
}

func unnestInput(target clause.TableRef, keys []BatchKey, _ int) clause.TableRef {
	return clause.TableRef{Expression: unnestKeys(target, keys), WithOrdinality: true}
}

func unnestKeys(target clause.TableRef, keys []BatchKey) bob.Expression {
	arrays := make([]any, len(keys))
	for i, k := range keys {
		arrays[i] = keyArray(target, k)
	}
	return psql.F("unnest", arrays...)
}

func keyArray(target clause.TableRef, k BatchKey) psql.Expression {
	return psql.Raw("COALESCE(?, ARRAY?)", psql.Arg(k.Values), emptyColumn(target, k.Column))
}

func valuesInput(target clause.TableRef, keys []BatchKey, size int) clause.TableRef {
	input := psql.Values()
	for i := range size {
		row := make([]bob.Expression, len(keys), len(keys)+1)
		for j, k := range keys {
			row[j] = psql.Raw("COALESCE(?, ?)", psql.Arg(k.Values.Index(i)), emptyColumn(target, k.Column))
		}
		input.Expression.AppendValues(append(row, expr.Raw(strconv.Itoa(i+1)))...)
	}
	return clause.TableRef{Expression: input}
}

// KeyValues collects one key component from each non-nil parent.
func KeyValues[P, K any](rows []*P, key func(*P) K) pgtypes.Array[K] {
	values := make(pgtypes.Array[K], 0, len(rows))
	for _, row := range rows {
		if row != nil {
			values = append(values, key(row))
		}
	}
	return values
}

// UniqueKeyValues deduplicates a single non-unique, comparable join column.
func UniqueKeyValues[P any, K comparable](rows []*P, key func(*P) K) pgtypes.Array[K] {
	values := make(pgtypes.Array[K], 0, len(rows))
	seen := make(map[K]struct{}, len(rows))
	for _, row := range rows {
		if row == nil {
			continue
		}
		k := key(row)
		if _, ok := seen[k]; !ok {
			values = append(values, k)
			seen[k] = struct{}{}
		}
	}
	return values
}

// FilterRelationshipKeys tests membership of scalar keys without multiplying
// rows. Use the target columns' types and collations, as in MatchParents.
func FilterRelationshipKeys(target clause.TableRef, keys ...BatchKey) bob.Mod[*dialect.SelectQuery] {
	if len(keys) == 1 {
		k := keys[0]
		return sm.Where(psql.Quote(target.Alias, k.Column).EQ(psql.Any(keyArray(target, k))))
	}
	names := relbatch.KeyNames(len(keys))
	cols := make([]bob.Expression, len(keys))
	inputCols := make([]any, len(keys))
	for i, k := range keys {
		cols[i] = psql.Quote(target.Alias, k.Column)
		inputCols[i] = psql.Quote("p", names[i])
	}
	filter := psql.Select(sm.Columns(inputCols...), sm.From(unnestKeys(target, keys)).As("p", names...))
	return sm.Where(psql.Group(cols...).OP("IN", filter))
}
