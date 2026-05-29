---
sidebar_position: 3
description: How to use quotes with Bob
---

# Quotes

Identifiers in SQL often need dialect-specific quoting. Use `dialect.Quote("part1", "part2")` (e.g. `psql.Quote`, `mysql.Quote`, `sqlite.Quote`) to build a qualified quoted name: `"schema"."table"` on PostgreSQL/SQLite, `` `schema`.`table` `` on MySQL.

Bob uses two patterns:

1. **`string` parameters that mean a single SQL identifier** — the dialect quotes them when rendering (you pass `"kind"`, SQL gets `"kind"`).
2. **`any` parameters** — rendered with `bob.Express`. Pass `dialect.Quote(...)` for identifiers; a bare `string` is written **verbatim** (table names without quotes, SQL fragments, function names like `generate_series`).

For qualified column names in `SET` / `ON CONFLICT DO UPDATE`, use `SetExpr` (or `im.UpdateCol` on MySQL), not `SetCol`.

## When the API quotes for you

| API / parameter | Dialects | Notes |
|-----------------|----------|-------|
| `sm/im/um/dm/mm.With(name, columns...)` | psql, mysql, sqlite | CTE name and column list |
| `sm.Window(name)` | psql, mysql, sqlite | Named window |
| `wm.BasedOn(name)` | psql, mysql, sqlite | Reference to existing window |
| `im.OnConflictOnConstraint(name)` | psql, sqlite | Constraint name |
| `um/dm.WhereCurrentOf(cursor)` | psql | Cursor name |
| `um/im/mm.SetCol("col")` | psql; um/im on sqlite; um + `im.UpdateCol` on mysql | LHS of single-column `SET` |
| `um/im/mm.SetCols("a", "b")` | psql only | Tuple assignment `(a, b) = ...` |
| `mm.Columns("a", "b")` | psql only | `MERGE ... INSERT (a, b)` column list |
| `im.SetExcluded("a", "b")` | psql, sqlite | `"a" = EXCLUDED."a"`, ... |
| `im.Excluded("col")` | psql, sqlite | `EXCLUDED."col"` |
| `im.Into(table, "c1", "c2")` | all | Insert **column** names (`table` is `any`, see below) |
| `FromChain.As(alias, "c1", ...)` | all | Table alias and optional rename columns |
| `JoinChain.Using("c1", "c2")` | all | `USING` column names |
| `OrderBy(...).Collate(name)` | all | Collation name |
| `fm.Columns("col", datatype)` | psql | Column name in table-function definition (`datatype` is literal SQL) |

## When you pass `any` and use `Quote()` yourself

| API / parameter | Typical use |
|-----------------|-------------|
| `sm.From(table)` | `From("users")` → unquoted; `From(psql.Quote("users"))` or `From(psql.Quote("public", "users"))` when quoted |
| `im.Into(name, ...)` | `Into(psql.Quote("distributors"), "did", "dname")` — table `any`, columns still `string` and auto-quoted |
| `um.Table(name)` / `TableAs(name, alias)` | `Table(psql.Quote("employees"))`; `alias` is a `string` and quoted via `WriteQuoted` |
| `dm.From` / `mm.Into` / `mm.Using` | Same as `From` / `Into` |
| `sm.Columns(...)` | `Columns(psql.Quote("id"))` — each item is `any` |
| `im.OnConflict(...)` | `OnConflict(psql.Quote("id"))` |
| `Returning(...)` | `Returning(psql.Quote("id"))` |
| `sm.ForUpdate` / `ForShare` / `ForNoKeyUpdate` / `ForKeyShare` | `ForUpdate(psql.Quote("my_table"))` |
| `sm.OrderBy` / `GroupBy` / `Having` | Expression or `Quote` as needed |
| `psql.F(name, args...)` | `F("generate_series", 1, 3)` — literal name; `F(psql.Quote("pg_catalog", "array_agg"), col)` for qualified |

`Cast(exp, typname string)` appends the type name as literal text (not an identifier slot).

## Examples

```go
// Qualified quoted name
psql.Quote("schema_name", "table_name") // "schema_name"."table_name"

// Column in SET (quoted automatically)
um.SetCol("kind").ToArg("Drama")

// Qualified column in SET (SetExpr = SetCol with an expression LHS)
um.SetExpr(psql.Quote("employees", "id")).ToArg(1)

// Several assignments in um.Set (do not use EQ — it wraps in parentheses)
um.Set(
  psql.Quote("employees", "dept_id").Assign(psql.Quote("accounts", "dept_id")),
)

// MERGE update
mm.SetCol("price").To(psql.Quote("u", "price"))

// Table in FROM (pass Quote when you want quoting)
sm.From(psql.Quote("films"))

// Function name
psql.F(psql.Quote("pg_catalog", "array_agg"), psql.Quote("x"))
```
