---
sidebar_position: 1.1
description: How to use queries built with Bob
---

# Using the Query

## Building queries

The `Query` interface embeds `Expression` and adds:

```go
type Query interface {
	Expression
	Type() QueryType
	// start is the index of the args, usually 1.
	// it is present to allow re-indexing in cases of a subquery
	// The method returns the value of any args placed
	// An `io.StringWriter` is used for efficiency when building the query.
	WriteQuery(ctx context.Context, w io.StringWriter, start int) (args []any, err error)
}
```

`WriteQuery` is the low-level renderer. When writing a custom expression that
contains a subquery, call `bob.WriteQuery(ctx, w, start, subquery)` to run the
subquery's hooks with the enclosing build's executor. Use the build methods
when you want a SQL string and arguments:

- `Build(ctx context.Context, exec bob.Executor) (query string, args []any, err error)`
- `BuildN(ctx context.Context, exec bob.Executor, start int) (query string, args []any, err error)`
- `MustBuild(ctx context.Context, exec bob.Executor) (query string, args []any) // panics on error`
- `MustBuildN(ctx context.Context, exec bob.Executor, start int) (query string, args []any) // panics on error`

```go
queryString, args, err := psql.Select(...).Build(ctx, nil)
```

Since the query is built from scratch every time the `WriteQuery()` method is called, it can be useful to initialize the query one time and reuse where necessary.

For that, the `MustBuild()` function can be used. This panics on error.

```go
myquery, myargs := psql.Insert(...).MustBuild(ctx, nil)
```

## Executing queries

The returned `query` and `args` can then be passed to your querier (e.g. `*sql.DB` or `*sql.Tx`) to execute

```go
ctx := context.Background()

// Build and execute with the same executor.
myquery, myargs := psql.Insert(...).MustBuild(ctx, db)

// Execute the query
err := db.ExecContext(ctx, myquery, myargs...)
```

In addition to these, `Bob` also has a [sql executor](../sql-executor/intro) which can build and run queries in a single step.
