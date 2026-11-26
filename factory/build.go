// Package factory provides the runtime helpers used by generated Bob factories.
package factory

import (
	"context"
	"testing"

	"github.com/stephenafamo/bob"
)

// BuildMany evaluates build separately for each element.
func BuildMany[S ~[]T, T any](n int, build func() T) S {
	rows := make(S, n)
	for i := range rows {
		rows[i] = build()
	}
	return rows
}

// CreateMany stops on the first failed insert and returns no partial slice.
func CreateMany[S ~[]T, T any](ctx context.Context, exec bob.Executor, n int, create func(context.Context, bob.Executor) (T, error)) (S, error) {
	rows := make(S, n)
	for i := range rows {
		row, err := create(ctx, exec)
		if err != nil {
			return nil, err
		}
		rows[i] = row
	}
	return rows, nil
}

// Must returns the factory result or panics on failure.
func Must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}

// OrFail checks a result on behalf of a generated CreateOrFail method.
func OrFail[T any](tb testing.TB, v T, err error) T {
	tb.Helper()
	if err != nil {
		tb.Fatal(err)
	}
	return v
}
