---
sidebar_position: 2
description: Easily query a database table
---

# View

A View model makes it easy to map an entity to a database table and query it.

To create a View model, use the `NewView()` function.

```go
import (
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/expr"
	"github.com/stephenafamo/bob/dialect/psql"
)

type User struct {
	ID	int
	Name  string
	Email string
}

var userView = psql.NewView[*User, bob.Expression]("public", "users", expr.ColsForStruct[User]("users"))
```

:::tip

The `NewViewx()` function takes an extra type parameter to determine how slices of the corresponding table struct are returned.

:::

A View model provides the following methods:

## Name()

Returns the bare table or view name as a `string` (no schema prefix). For example, `"users"`.

Use this when you need the identifier as data (logging, map keys, etc.), not when building SQL.

## Schema()

PostgreSQL and SQLite only. Returns the schema name the model was constructed with, or `""` if none was set at codegen time.

When the schema is empty, pass a schema at runtime with `psql.UseSchema` or `sqlite.UseSchema` so `NameExpr()` can qualify the table in SQL.

## Alias()

Returns the alias used for generated columns and for `NameAsExpr()`. When a schema is set at construction, this is usually `schema.table` (e.g. `"public.users"`). Otherwise it matches the table name.

## NameExpr()

Returns the table or view as a bob [expression](../query-builder/building-queries#expressions) for query builders (`sm.From`, `im.Into`, joins, etc.).

- PostgreSQL/SQLite with schema at construction: qualified name, e.g. `"public"."users"`.
- PostgreSQL/SQLite with empty schema: uses `UseSchema` from context when present; otherwise the unqualified table name.
- MySQL: the quoted table name (MySQL models do not use schema/database qualification).

```go
import (
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/sm"
)

query := psql.Select(sm.From(userView.NameExpr()))
```

## NameAsExpr()

Like `NameExpr()`, but adds `AS alias` when the alias differs from the table name (for example when a schema is set at construction, so the alias is `schema.table`). When schema is empty and the alias matches the table name, the redundant `FROM "users" AS "users"` form is omitted; `Alias()` is unchanged for joins and column qualification. Use in `FROM`, `JOIN`, and other clauses that need a named range variable—for example, PostgreSQL/SQLite `INSERT INTO ... AS alias` or subqueries that reference `Alias()` in `WHERE`.

```go
import (
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/sm"
)

query := psql.Select(sm.From(userView.NameAsExpr()))
```

:::note

MySQL does not support a table alias on the `INSERT` target (`INSERT INTO t AS alias`). Generated MySQL `Table.Insert()` uses `NameExpr()` only; use row aliases in `ON DUPLICATE KEY UPDATE` when needed.

:::

## Columns

A [`expr.Columns`](https://pkg.go.dev/github.com/stephenafamo/bob/expr.Columns) field.
This is also a bob [expression](../query-builder/building-queries#expressions). Which by default, the expression evaluates to:

```sql
-- table_alias.column_name
"public.users"."id" AS "id",
"public.users"."name" AS "name",
"public.users"."email" AS "email"
```

Learn about how to manipulate a columns list in the [columns documentation](./columns).

## Query()

The `Query()` method on a View model starts a SELECT query on the model's database view/table. It accepts [query mods](../query-builder/building-queries#query-mods) to modify the final query.

```go
import (
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/sm"
)

query := userView.Query(
	sm.Where(
		psql.Quote("name").In(
			psql.Arg("Ayan", "Rudra", "Ila")
		),
		sm.Limit(10), // LIMIT 10
	)
)
```

The query can then be executed with `One()`, `All()`, `Cursor()`, `Count()` or `Exists()`.

```go
// SELECT * FROM "users" LIMIT 1
userView.Query().One(ctx, db)

// SELECT * FROM "users"
userView.Query().All(ctx, db)

// Like All, but returns a cursor for moving through large results
userView.Query().Cursor(ctx, db)

// SELECT count(1) FROM "users"
userView.Query().Count(ctx, db)

// Like One(), but only returns a boolean indicating if the model was found
userView.Query().Exists(ctx, db)
```

:::tip

The `Count()` function clones the current query which can be an expensive operation.

:::

