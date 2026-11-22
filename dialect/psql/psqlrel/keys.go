// Package psqlrel builds PostgreSQL batch filters for generated relationships.
package psqlrel

import (
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/dialect"
	"github.com/stephenafamo/bob/dialect/psql/sm"
	"github.com/stephenafamo/bob/types/pgtypes"
)

// RelationshipKey is one column of a batch relationship filter. Columns of
// a composite key must have the same number of values in the same order.
type RelationshipKey struct {
	Column string
	DBType string
	Values any
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

// FilterRelationshipKeys uses ANY for a single column, or a join to distinct
// unnested tuples for composite keys. Neither form multiplies matching rows.
func FilterRelationshipKeys(table string, keys ...RelationshipKey) bob.Mod[*dialect.SelectQuery] {
	if len(keys) == 1 {
		k := keys[0]
		return sm.Where(psql.Quote(table, k.Column).EQ(psql.Any(psql.Cast(psql.Arg(k.Values), k.DBType+"[]"))))
	}
	const src, batch = "bob_rel_keys_src", "bob_rel_keys"
	names := make([]string, len(keys))
	cols := make([]any, len(keys))
	arrays := make([]any, len(keys))
	on := make([]bob.Expression, len(keys))
	for i, k := range keys {
		names[i] = k.Column
		cols[i] = psql.Quote(src, k.Column)
		arrays[i] = psql.Cast(psql.Arg(k.Values), k.DBType+"[]")
		on[i] = psql.Quote(table, k.Column).EQ(psql.Quote(batch, k.Column))
	}
	return sm.InnerJoin(psql.Select(
		sm.Distinct(), sm.Columns(cols...),
		sm.From(psql.F("unnest", arrays...)).As(src, names...),
	)).As(batch).On(on...)
}
