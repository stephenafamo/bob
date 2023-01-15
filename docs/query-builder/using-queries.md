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
	WriteQuery(w io.Writer, start int) (args []any, err error)
}
```

The `WriteQuery` method is useful when we want to write to an exisiting `io.Writer`. However we often just want the query string and arguments. So the Query objects have the following methods:

* `Build() (query string, args []any, err error)`
* `BuildN(start int) (query string, args []any, err error)`
* `MustBuild() (query string, args []any) // panics on error`
* `MustBuildN(start int) (query string, args []any) // panics on error`

```go
queryString, args, err := psql.Select(...).Build()
```

Since the query is built from scratch every time the `WriteQuery()` method is called, it can be useful to initialize the query one time and reuse where necessary.

For that, the `MustBuild()` function can be used. This panics on error.

```go
myquery, myargs := psql.Insert(...).MustBuild()
```

## Executing queries

The returned `query` and `args` can then be passed to your querier (e.g. `*sql.DB` or `*sql.Tx`) to execute

```go
ctx := context.Background()
myquery, myargs := psql.Insert(...).MustBuild()
err := sql.ExecContext(ctx, myquery, myargs...)
```

In addition to these, `Bob` also has a [sql executor](../sql-executor) which can build and run queries in a single step.
