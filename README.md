# Bob (the builder): A spec compliant SQL query builder

[![Go Reference](https://pkg.go.dev/badge/github.com/stephenafamo/bob.svg)](https://pkg.go.dev/github.com/stephenafamo/bob)

## Features

* Faster than comparable packages. [See Benchmarks](https://github.com/stephenafamo/go-sql-builder-benchmarks).
* Build any query. Supports the specification as closely as possible.

## Examples

Examples are in the [examples folder](examples):

* [Postgres](examples/psql)
* [SQLite](examples/sqlite)

## Principles

### Custom Crafting

In `bob`, each dialect, and the applicable query mods are custom crafted
to be as close to the specification as possible.
This is unlike most other query builders that use a common structure and attempt to adapt it to every dialect.

### Progressive enhancement

Most query mods will accept a literal string that will be printed as is.

However, many functions and methods are provided to express even the most complex queries in readable Go code.

```go
// Assuming we're building the following query
/*
SELECT status,
    LEAD(created_date, 1, NOW())
    OVER(PARTITION BY presale_id ORDER BY created_date) -
    created_date AS "difference"
FROM presales_presalestatus
*/

// different ways to express "SELECT status"
qm.Select("status") // SELECT status
qm.Select(expr.Quote("status")) // SELECT "status"

// Ways to express LEAD(created_date, 1, NOW())
"LEAD(created_date, 1, NOW()")
expr.Func("LEAD", "created_date", 1, "NOW()"))
expr.Func("LEAD", "created_date", 1, expr.Func("NOW"))

// Ways to express PARTITION BY presale_id ORDER BY created_date
"PARTITION BY presale_id ORDER BY created_date"
expr.Window("").PartitionBy("presale_id").OrderBy("created_date")

// Expressing LEAD(...) OVER(...)
"LEAD(created_date, 1, NOW()) OVER(PARTITION BY presale_id ORDER BY created_date)"
expr.OVER(
    expr.Func("LEAD", "created_date", 1, expr.Func("NOW")),
    expr.Window("").PartitionBy("presale_id").OrderBy("created_date"),
)

// The full query
psql.Select(
    qm.Select(
        "status",
        expr.C(expr.MINUS(expr.OVER(
            expr.Func("LEAD", "created_date", 1, expr.Func("NOW")),
            expr.Window("").PartitionBy("presale_id").OrderBy("created_date"),
        ), "created_date"), "difference"),
    ),
    qm.From("presales_presalestatus"),
)
```

## QueryMods

QueryMods are options applied to a query. Each query type of each dialect defines what mods can be applied to it.  
This way, the possible options can be built to match the spec as closely as possible.

Despite this custom configuration, the mods are designed to match each other as closely
so that switching dialects can be achieved by simply switching imports.  
However, if using an unspported mod, the error will be displayed at compile time.

As an example, both `SELECT` and `INSERT` can use CTEs(Common Table Expressions), but while `INSERT` can take an `INTO` expression, `SELECT` instead needs a `FROM`

```go
var selMod = psql.SelectQM{}
cte := psql.Select(
    selMod.From("users"),
    selMod.Where(expr.GTE("age", 21)),
)

var cte query.Query
psql.Select(
    selMod.With("adults").As(cte), // works
    selMod.From("projects"),
)

var inMod = psql.InsertQM{}
psql.Insert(
    inMod.With("adults").As(cte), // works as well
    inMod.From("projects"), // ERROR!
    inMod.Into("projects"), // works
)
```

Using this query mod system, the mods closely match the allowed syntax for each specific query type.

## Quotes

It is often required to quote identifiers in SQL queries. With `bob`  use the `expr.Quote()` where necessary.  
When building the query, the quotes are added correctly by the dialect.

It can take multiple strings that need to be quoted and joined with `.`

```go
// Postgres: "schema_name"."table_name"
// MySQL: `schema_name`.`table_name`
expr.Quote("schema_name", "table_name")
```

## Placeholders

To prevent SQL injection, it is necessary to use placeholders in our queries. With `bob` use `expr.Arg()` where necessary.  
This will write the placeholder correctly in the generated sql, and return the value in the argument slice.

```go
// args: 100, "Stephen"
// Postgres: SELECT * from users WHERE id = $1 AND name = $2
// MySQL: SELECT * from users WHERE id = ? AND name = ?
// SQL Server: SELECT * from users WHERE id = @p1 AND name = @p2
psql.Select(
    qm.From("users"),
    qm.Where(expr.EQ("id", expr.Arg(100))),
    qm.Where(expr.EQ("name", expr.Arg("Stephen"))),
).WriteQuery(w, 1)
```

Another option is to use `expr.Statement()` which takes a clause and args. The placeholder in the clauses are question marks `?`.

```go
// SELECT * from users WHERE id = $1 AND name = $2
// args: 100, "Stephen"
psql.Select(
    qm.From("users"),
    qm.Where(expr.Statement("id = ? and name = ?", 100, "Stephen")),
).WriteQuery(w, 1)
```

## Using the Query

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

For times when you only want the strings and args,
and don't want to create an io.Writer, the `Build` function can be used:

```go
queryString, args, err := query.Build(psql.Select(...))
```

Since the query is built from scratch every time the `WriteQuery()` method is called,
it can be useful to initialize the query one time and reuse where necessary.

For that, the `MustBuild()` function can be used. This panics on error.

```go
var myquery, myargs = query.MustBuild(psql.Insert(...))
```

## Roadmap

* [x] Postgres
  * [x] Select
  * [x] Insert
  * [x] Update
  * [x] Delete
  * [ ] Postgres Specific Operators
    * [ ] Is [Not] True
    * [ ] Is [Not] False
    * [ ] Is [Not] Unknown
    * [ ] [Not] Between Symmetric
    * [ ] Is [Not] [NFC|NFD|NFKC|NFKD] Normalized
* [ ] MySQL
  * [ ] Select
  * [ ] Insert
  * [ ] Update
  * [ ] Delete
* [x] SQLite
  * [x] Select
  * [x] Insert
  * [x] Update
  * [x] Delete
  * [ ] SQLite Specific Operators
    * [ ] GLOB
* [ ] SQL Server
  * [ ] Select
  * [ ] Insert
  * [ ] Update
  * [ ] Delete
* [ ] Common Operators
  * [x] [Not] Equal
  * [x] Not Equal
  * [x] Less than
  * [x] Less than or equal to
  * [x] Greater than
  * [x] Greater than or equal to
  * [x] And
  * [x] Or
  * [x] [Not] In
  * [x] [Not] Null
  * [x] Is [not] distinct from
  * [x] Concatenation: ||
  * [ ] Between
