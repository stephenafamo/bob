# TypeSQL: A typesafe SQL query builder

[![Go Reference](https://pkg.go.dev/badge/github.com/stephenafamo/typesql.svg)](https://pkg.go.dev/github.com/stephenafamo/typesql)

## Principles

### Custom Crafting

In `typesql`, each dialect, and the applicable query mods are custom crafted
to be as close to the specification as possible.
This is unlike most other query builders that use a common structure and attempt to adapt it to every dialect.

### Progressive enhancement

Most query mods will accept aa literal string that will be printed as is.

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

psql.Select(
    selMod.With(cte, "adults"), // works
    selMod.From("projects"),
)

var inMod = psql.InsertQM{}
psql.Insert(
    inMod.With(cte, "adults"), // works as well
    inMod.From("projects"), // ERROR!
    inMod.Into("projects"), // works
)
```

Using this query mod system, the mods closely match the allowed syntax for each specific query type.

## Quotes

It is often required to quote identifiers in SQL queries. With `typesql`  use the `expr.Quote()` where necessary.  
When building the query, the quotes are added correctly by the dialect.

It can take multiple strings that need to be quoted and joined with `.`

```go
// Postgres: "schema_name"."table_name"
// MySQL: `schema_name`.`table_name`
expr.Quote("schema_name", "table_name")
```

## Placeholders

To prevent SQL injection, it is necessary to use placeholders in our queries. With `typesql` use `expr.Arg()` where necessary.  
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

## Using the returned Query

The Query object is an interface that has a single method:

```go
type Query interface {
    // start is the index of the args, usually 1.
    // it is present to allow re-indexing in cases of a subquery
    // The method returns the value of any args placed
    WriteQuery(w io.Writer, start int) (args []any, err error)
}
```

An `io.Writer` is used for efficiency when building the query.

## Roadmap

* [ ] Postgres
  * [x] Select
  * [x] Insert
  * [ ] Update
  * [ ] Delete
* [ ] MySQL
  * [ ] Select
  * [ ] Insert
  * [ ] Update
  * [ ] Delete
* [ ] SQLite
  * [ ] Select
  * [ ] Insert
  * [ ] Update
  * [ ] Delete
* [ ] SQL Server
  * [ ] Select
  * [ ] Insert
  * [ ] Update
  * [ ] Delete
* [ ] Common Operators
