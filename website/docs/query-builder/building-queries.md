---
sidebar_position: 1
description: How to build queries with Bob
---

# Building Queries

Query building is done with the use of QueryMods.

## Query Mods

QueryMods are options applied to a query. Each query type of each dialect defines what mods can be applied to it. This way, the possible options can be built to match the spec as closely as possible.

Despite this custom configuration, the mods are designed to match each other as closely so that switching dialects can be achieved by simply switching imports. However, if using an unsupported mod, the error will be displayed at compile time.

As an example, both `SELECT` and `INSERT` can use CTEs(Common Table Expressions), but while `INSERT` can take an `INTO` expression, `SELECT` instead needs a `FROM`

```go
import "github.com/stephenafamo/bob/dialect/psql/sm"
cte := psql.Select(
    sm.From("users"),
    sm.Where(psql.Quote("age").GTE(psql.Arg(21))),
)

var cte query.Query
psql.Select(
    sm.With("adults").As(cte), // works
    sm.From("projects"),
)

import "github.com/stephenafamo/bob/dialect/psql/insert/im"
psql.Insert(
    im.With("adults").As(cte), // works as well
    im.From("projects"), // ERROR: Does not compile!!!
    im.Into("projects"), // works
)
```

Using this query mod system, the mods closely match the allowed syntax for each specific query type.

Use `WithMods` to add mods conditionally.

```go
q := psql.Select(
	sm.From("projects"),
) // SELECT * FROM projects

if !user.IsAdmin {
	q = q.WithMods(
		sm.Where(psql.Quote("user_id").EQ(psql.Arg(user.ID))),
	) // SELECT * FROM projects WHERE "user_id" = $1
}
```

For PostgreSQL, SQLite, and MySQL builders, including table writes and generated
SQL queries, `WithMods` returns a new query. Two callers can
derive from the same base: `left := base.WithMods(leftFilter)` and
`right := base.WithMods(rightFilter)`. To run the same query again, reuse `base`.
Use the result: a bare `q.WithMods(filter)` leaves `q` unchanged. Run `bobcheck`
after upgrading to find these calls.

For a bulk insert, collect the rows first and use
`psql.Insert(im.Into("jobs", "id"), im.Rows(rows...))`. A separate `WithMods`
for each row has to preserve every intermediate query and copies the growing
row list repeatedly.

Custom mods can replace or append entries in the clause lists they
receive. For example, changing an insert column or replacing a cell in a
`VALUES` row affects only that query, even when another query uses the same
`im.Into` or `im.Rows` mod. The same rule applies to nested clause lists such
as `GROUPING SETS`. Do not retain the editor after the callback.
Treat a reusable mod's inputs as read-only. Replace an expression in a clause
instead of editing the expression, its bound values, or its callback state.

Table and view query hooks run in order on one scoped editor per build.
Each hook sees the previous hook's edits and context. Apply a mod with
`Edit` and return the context and any error.

```go
models.Projects.SelectQueryHooks.AppendHooks(func(ctx context.Context, _ bob.Executor, e psql.SelectEditor) (context.Context, error) {
	sm.Where(psql.Quote("archived").EQ(psql.Arg(false))).Edit(e)
	return ctx, nil
})
```

Register hooks before constructing a table or view query. Later registrations
apply to new queries. Existing queries and their `WithMods` branches keep their
original hook list. Both `bob.Build(ctx, tx, q)` and `bob.One(ctx, tx, q, mapper)`
run query hooks once per statement, including nested statements, with `tx`.
Pass `nil` explicitly to build without an executor. A hook that needs one can
return an error. Standalone builds do not run model lifecycle hooks.

`bob.RunQueryHooks(ctx, exec, q)` accepts any `bob.Query`, including a table,
view, bound query, or custom wrapper. It returns `(ctx, stmt, err)`. After
checking `err`, build or execute `stmt` with the returned context. Its hooks
have already run.

A custom query can implement `bob.HookableQuery` to choose the statement:

```go
var _ bob.HookableQuery = filteredQuery{}

func (q filteredQuery) RunHooks(ctx context.Context, exec bob.Executor) (context.Context, bob.Query, error) {
	return ctx, q.selectQuery.WithMods(sm.Where(q.filter)), nil
}
```

Execution, caching, and preparation use the returned statement's SQL, mapper
mods, and loaders. For a wrapper that also changes how SQL is written, return
the wrapper around the resolved inner statement. A hook that only changes the
context must still return the query to execute.

## Expressions

Every dialect contain starter functions to fluently build complex expressions. It starts with one of several functions which then return a chain that has methods for various operators.

For example:

```go
// Query: ($1 >= 50) AND ("name" IS NOT NULL)
// Args: 'Stephen'
psql.Arg("Stephen").GTE(psql.Raw(50)).
	And(psql.Quote("name").IsNotNull())

// OR

psql.And(
	psql.Arg("Stephen").GTE(psql.Raw(50)),
	psql.Quote("name").IsNotNull(),
)
```

### Starters

These functions are included in every dialect and can be used to create a chainable expression.

See the [starters page](./starters) for the list of common starters.

### Operators

The expression type returned by the starter functions have methods to build queries using operators.

See the [operators page](./operators) for the list of common operators.

## Raw Queries

As any good query builder, you are allowed to use your own raw SQL queries. Either at the top level with `psql.RawQuery()` or inside any clause with `psql.Raw()`.

These functions take a query and args. The placeholder in the clauses are question marks `?`.

```go
// SELECT * from users WHERE id = $1 AND name = $2
// args: 100, "Stephen"

psql.RawQuery(`SELECT * FROM USERS WHERE id = ? and name = ?`, 100, "Stephen")
// -----
// OR
// -----
psql.Select(
	sm.From("users"),
	sm.Where(psql.Raw("id = ? and name = ?", 100, "Stephen")),
)
```
