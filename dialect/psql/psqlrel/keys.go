// Package psqlrel builds PostgreSQL batch filters for generated relationships.
package psqlrel

import (
	"strconv"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/clause"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/dialect"
	"github.com/stephenafamo/bob/dialect/psql/sm"
	"github.com/stephenafamo/bob/types/pgtypes"
)

// BatchKey holds the values for one column of a relationship filter.
type BatchKey struct {
	Column string
	Values any
}

// Infer the parameter type from the column without reading any rows.
// An explicit CHARACTER[] cast, for example, would truncate its elements.
func emptyColumn(target clause.TableRef, col string) bob.Query {
	return psql.Select(sm.Columns(psql.Quote(col)), sm.From(target.Expression), sm.Where(psql.Raw("FALSE")))
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
// rows. Use the target columns' types and collations for the filter.
func FilterRelationshipKeys(target clause.TableRef, keys ...BatchKey) bob.Mod[dialect.SelectEditor] {
	if len(keys) == 1 {
		k := keys[0]
		return sm.Where(psql.Quote(target.Alias, k.Column).EQ(psql.Any(keyArray(target, k))))
	}
	names := make([]string, len(keys))
	cols := make([]bob.Expression, len(keys))
	inputCols := make([]any, len(keys))
	for i, k := range keys {
		names[i] = "k" + strconv.Itoa(i)
		cols[i] = psql.Quote(target.Alias, k.Column)
		inputCols[i] = psql.Quote("p", names[i])
	}
	filter := psql.Select(sm.Columns(inputCols...), sm.From(unnestKeys(target, keys)).As("p", names...))
	return sm.Where(psql.Group(cols...).OP("IN", filter))
}
