package bob

import (
	"bytes"
	"context"
)

// MustBuild builds a query and panics on error
// useful for initializing queries that need to be reused
func MustBuild(ctx context.Context, q Query) (string, []any) {
	return MustBuildN(ctx, q, 1)
}

func MustBuildN(ctx context.Context, q Query, start int) (string, []any) {
	sql, args, err := BuildN(ctx, q, start)
	if err != nil {
		panic(err)
	}

	return sql, args
}

// Convinient function to build query from start
func Build(ctx context.Context, q Query) (string, []any, error) {
	return BuildN(ctx, q, 1)
}

// Convinient function to build query from a point
func BuildN(ctx context.Context, q Query, start int) (string, []any, error) {
	b := &bytes.Buffer{}
	args, err := q.WriteQuery(ctx, b, start)

	return b.String(), args, err
}
