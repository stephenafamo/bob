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

For conditional queries, the query object have an `Apply()` method which can be used to add more query mods.

```go
q := psql.Select(
	sm.From("projects"),
) // SELECT * FROM projects

if !user.IsAdmin {
	q.Apply(
		sm.Where(psql.Quote("user_id").EQ(psql.Arg(user.ID))),
	) // SELECT * FROM projects WHERE "user_id" = $1
}
```

> Since the mods modify the main query object any new mods added with `Apply()` will affect all instances of the query.
>
> To reuse the base of a query and add new mods each time, first use the `Clone()` method.

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
