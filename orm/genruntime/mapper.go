package genruntime

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/stephenafamo/scan"
)

type scanTarget[T any] struct {
	idx int
	dst func(*T) any
}

// PositionalMapper scans one value per row. The generated schedule binds
// fields by position, so even columns with identical labels remain distinct.
func PositionalMapper[T any](schedule func(*scan.Row, *T)) scan.Mapper[T] {
	return func(context.Context, []string) (scan.BeforeFunc, func(any) (T, error)) {
		return func(row *scan.Row) (any, error) {
			v := new(T)
			schedule(row, v)
			return v, nil
		}, func(v any) (T, error) { return *v.(*T), nil }
	}
}

func scanTargets[T any](cols []string, prefix string, field func(string) func(*T) any) []scanTarget[T] {
	targets := make([]scanTarget[T], 0, len(cols))
	for i, col := range cols {
		if name, ok := strings.CutPrefix(col, prefix); ok {
			if dst := field(name); dst != nil {
				targets = append(targets, scanTarget[T]{i, dst})
			}
		}
	}
	return targets
}

// ModelMapper resolves columns once, then scans directly into a new model
// for each row. The generated field selector returns nil for unknown columns.
func ModelMapper[T any](field func(string) func(*T) any) scan.Mapper[*T] {
	return func(_ context.Context, cols []string) (scan.BeforeFunc, func(any) (*T, error)) {
		targets := scanTargets(cols, "", field)
		return func(row *scan.Row) (any, error) {
			model := new(T)
			for _, target := range targets {
				row.ScheduleScanByIndex(target.idx, target.dst(model))
			}
			return model, nil
		}, func(v any) (*T, error) { return v.(*T), nil }
	}
}

// JoinedModelMapper scans prefixed columns into a nullable scratch row.
// finish must return nil on a LEFT JOIN miss, and a fresh model on a hit.
func JoinedModelMapper[B, T any](prefix string, field func(string) func(*B) any, finish func(*B) *T) scan.Mapper[*T] {
	return func(_ context.Context, cols []string) (scan.BeforeFunc, func(any) (*T, error)) {
		targets := scanTargets(cols, prefix, field)
		buf := new(B)
		return func(row *scan.Row) (any, error) {
			for _, target := range targets {
				row.ScheduleScanByIndex(target.idx, target.dst(buf))
			}
			return buf, nil
		}, func(any) (*T, error) { return finish(buf), nil }
	}
}

// CollectRelationshipKey scans hidden key columns beside the model. Associate
// keys by pointer so after-select hooks may safely reorder the result slice.
func CollectRelationshipKey[T, K any](keys map[*T]K, destinations func(*K) []any) scan.MapperMod {
	return func(_ context.Context, cols []string) (scan.BeforeFunc, scan.AfterMod) {
		var key K
		dst := destinations(&key)
		indexes := make([]int, len(dst))
		for i := range indexes {
			name := "related_.key" + strconv.Itoa(i)
			indexes[i] = slices.Index(cols, name)
			if indexes[i] < 0 {
				return func(*scan.Row) (any, error) { return nil, fmt.Errorf("missing relationship key column %q", name) },
					func(any, any) error { return nil }
			}
		}
		before := func(row *scan.Row) (any, error) {
			var zero K
			key = zero
			for i, index := range indexes {
				row.ScheduleScanByIndex(index, dst[i])
			}
			return nil, nil
		}
		after := func(_, model any) error {
			keys[model.(*T)] = key
			return nil
		}
		return before, after
	}
}
