package orm

import (
	"context"

	"github.com/stephenafamo/bob"
)

// CopyByKey updates each destination in place from the first source with the
// same primary key. The generated copy function copies column fields only.
func CopyByKey[T any, K comparable](dst, src []T, key func(T) K, copyColumns func(T, T)) {
	first := make(map[K]T, len(src))
	for _, row := range src {
		k := key(row)
		if _, ok := first[k]; !ok {
			first[k] = row
		}
	}
	for _, row := range dst {
		if match, ok := first[key(row)]; ok {
			copyColumns(row, match)
		}
	}
}

// ApplyModelHooks connects a bulk mutation to the models being changed.
// RETURNING supplies the refreshed rows (whose own hooks run in Bob); Exec
// supplies no rows, so the loader runs the after hook on the original models.
func ApplyModelHooks[T any, S ~[]T](q interface {
	AppendHooks(...func(context.Context, bob.Executor) (context.Context, error))
	AppendLoader(...bob.Loader)
}, rows S, copyRows func(...T), before, after *bob.Hooks[S, bob.SkipModelHooksKey],
) {
	q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
		return before.RunHooks(ctx, exec, rows)
	})
	q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, result any) error {
		switch v := result.(type) {
		case T:
			copyRows(v)
		case []T:
			copyRows(v...)
		case S:
			copyRows(v...)
		default:
			_, err := after.RunHooks(ctx, exec, rows)
			return err
		}
		return nil
	}))
}

// StitchByKey links children to all parents with a matching non-NULL key.
// The child index lets through-table loaders use the key scanned beside a
// child instead of a field on the child. Both input orders are preserved.
func StitchByKey[P, C any, K comparable](
	parents []*P,
	children []C,
	parentKey func(*P) (K, bool),
	childKey func(int, C) (K, bool),
	link func(*P, C),
) {
	index := make(map[K][]*P, len(parents))
	for _, p := range parents {
		if p == nil {
			continue
		}
		if k, ok := parentKey(p); ok {
			index[k] = append(index[k], p)
		}
	}
	for i, child := range children {
		if k, ok := childKey(i, child); ok {
			for _, p := range index[k] {
				link(p, child)
			}
		}
	}
}
