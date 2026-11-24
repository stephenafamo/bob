package genruntime

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/stephenafamo/bob/types"
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

// ParentIndexes associates the owner list with the actual scanned model.
// Loaders and after-select hooks may reorder the slice before it is linked.
type ParentIndexes[T any] map[*T][]int

func (p ParentIndexes[T]) Mapper(m scan.Mapper[*T], name string) scan.Mapper[*T] {
	return func(ctx context.Context, cols []string) (scan.BeforeFunc, func(any) (*T, error)) {
		before, after := m(ctx, cols)
		index := slices.Index(cols, name)
		var owners types.JSON[[]int]
		return func(row *scan.Row) (any, error) {
				if index < 0 {
					return nil, fmt.Errorf("missing relationship owner column %q", name)
				}
				owners.Val = nil // each child keeps its own decoded slice
				row.ScheduleScanByIndex(index, &owners)
				return before(row)
			}, func(link any) (*T, error) {
				child, err := after(link)
				if err == nil {
					p[child] = owners.Val
				}
				return child, err
			}
	}
}
