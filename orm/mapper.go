package orm

import (
	"context"
	"slices"
	"strings"

	"github.com/stephenafamo/scan"
)

type scanTarget[T any] struct {
	idx int
	dst func(*T) any
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

// CollectColumn scans an extra column beside each mapped row, e.g. the
// parent key of a many-to-many join. Resolve its index once per result set.
func CollectColumn[T any](name string, dst *[]T) scan.MapperMod {
	return func(_ context.Context, cols []string) (scan.BeforeFunc, scan.AfterMod) {
		index := slices.Index(cols, name)
		return func(row *scan.Row) (any, error) {
			var zero T
			*dst = append(*dst, zero)
			if index >= 0 {
				row.ScheduleScanByIndex(index, &(*dst)[len(*dst)-1])
			}
			return nil, nil
		}, func(any, any) error { return nil }
	}
}
