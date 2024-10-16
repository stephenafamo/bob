---
sidebar_position: 1.1
description: How to use queries built with Bob
---

# Using the Query

## Building queries

The `Query` object is an interface that has a single method:

```go
type Query interface {
	// start is the index of the args, usually 1.
	// it is present to allow re-indexing in cases of a subquery
	// The method returns the value of any args placed
	// An `io.Writer` is used for efficiency when building the query.
	WriteQuery(ctx context.Context, w io.Writer, start int) (args []any, err error)
}
```

The `WriteQuery` method is useful when we want to write to an existing `io.Writer`. However, we often just want the query string and arguments. So the Query objects have the following methods:

- `Build(ctx context.Context) (query string, args []any, err error)`
- `BuildN(ctx context.Context, start int) (query string, args []any, err error)`
- `MustBuild(ctx context.Context) (query string, args []any) // panics on error`
- `MustBuildN(ctx context.Context, start int) (query string, args []any) // panics on error`

```go
queryString, args, err := psql.Select(...).Build(ctx)
```

Since the query is built from scratch every time the `WriteQuery()` method is called, it can be useful to initialize the query one time and reuse where necessary.

For that, the `MustBuild()` function can be used. This panics on error.

```go
myquery, myargs := psql.Insert(...).MustBuild(ctx)
```

## Executing queries

The returned `query` and `args` can then be passed to your querier (e.g. `*sql.DB` or `*sql.Tx`) to execute

```go
ctx := context.Background()

// Build the query
myquery, myargs := psql.Insert(...).MustBuild(ctx)

// Execute the query
err := db.ExecContext(ctx, myquery, myargs...)
```

In addition to these, `Bob` also has a [sql executor](../sql-executor/intro) which can build and run queries in a single step.
