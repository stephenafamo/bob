# Bob (the builder): A spec compliant SQL query builder

[![Go Reference](https://pkg.go.dev/badge/github.com/stephenafamo/bob.svg)](https://pkg.go.dev/github.com/stephenafamo/bob)

## Features

* Faster than comparable packages. [See Benchmarks](https://github.com/stephenafamo/go-sql-builder-benchmarks).
* Build any query. Supports the specification as closely as possible.

## Examples

Examples are in the [examples folder](examples):

* [Postgres](examples/psql)
* [SQLite](examples/sqlite)

## QuickLinks

* [Building A Query](#query-building)
* [Using A Query](#using-the-query)
* [Raw Queries](#raw-queries)

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
qm.Select(qm.Quote("status")) // SELECT "status"

// Ways to express LEAD(created_date, 1, NOW())
"LEAD(created_date, 1, NOW()"
qm.F("LEAD", "created_date", 1, "NOW()")
qm.F("LEAD", "created_date", 1, qm.F("NOW"))

// Ways to express PARTITION BY presale_id ORDER BY created_date
"PARTITION BY presale_id ORDER BY created_date"
qm.Window("").PartitionBy("presale_id").OrderBy("created_date")

// Expressing LEAD(...) OVER(...)
"LEAD(created_date, 1, NOW()) OVER(PARTITION BY presale_id ORDER BY created_date)"
qm.F("LEAD", "created_date", 1, qm.F("NOW")).
    Over("").
    PartitionBy("presale_id").
    OrderBy("created_date")

// The full query
psql.Select(
    qm.Select(
        "status",
        qm.F("LEAD", "created_date", 1, qm.F("NOW")).
            Over("").
            PartitionBy("presale_id").
            OrderBy("created_date").
            Minus("created_date").
            As("difference")),
    qm.From("presales_presalestatus")),
)
```

## Query Building

Query building is done with the use of QueryMods.

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
    selMod.Where(qm.X("age").GTE(21)),
)

var cte query.Query
psql.Select(
    selMod.With("adults").As(cte), // works
    selMod.From("projects"),
)

var inMod = psql.InsertQM{}
psql.Insert(
    inMod.With("adults").As(cte), // works as well
    inMod.From("projects"), // ERROR: Does not compile!!!
    inMod.Into("projects"), // works
)
```

Using this query mod system, the mods closely match the allowed syntax for each specific query type.

## Quotes

It is often required to quote identifiers in SQL queries. With `bob`  use the `qm.Quote()` where necessary.  
When building the query, the quotes are added correctly by the dialect.

It can take multiple strings that need to be quoted and joined with `.`

```go
// Postgres: "schema_name"."table_name"
// SQLite: "schema_name"."table_name"
// MySQL: `schema_name`.`table_name`
// SQL Server: [schema_name].[table_name]
qm.Quote("schema_name", "table_name")
```

## Expressions

The query mods have methods to fluently build complex expressions.  
It starts with one of several methods which then return a chain that has methods for various operators.

For example:

```go
// Query: ($1 >= 50) AND (name IS NOT NULL)
// Args: 'Stephen'
qm.Arg("Stephen").GTE(50).
    And(qm.X("name").IsNotNull())

// OR

qm.And(
    qm.Arg("Stephen").GTE(50),
    qm.X("name").IsNotNull(),
)
```

### Starters

These methods are embeded in every query mod and can be used to create a chainable expression.

**NOTE:** These are the common starters. Each dialect can sometimes include their own starters.  
For example, starters for common function calls can easily be added

* `X(any)`: Plain start to a chain.
* `NotX(any)`: Creates a `NOT expr` expression that is then chainable.
* `F(name string, args ...any)`: A generic function call. Takes a name and the arguments.
* `OR(...any)`: Joins multiple expressions with "OR"
* `AND(...any)`: Joins multiple expressions with "AND"
* `CONCAT(...any)`: Joins multiple expressions with "||"
* `S(string)`: Create a plain string literal. Single quoted.
* `Arg(...any)`: One or more arguments. These are replaced with placeholders in the query and the args returned.
* `Placeholders(uint)`: Inserts a `count` of placeholders without any specific value yet. Useful for compiling reusable queries.
* `Statement(clause string, args ...any)`: For inserting a raw statement somewhere. To keep it dialect agnostic, placeholders should be inserted with `?` and a literal question mark can be escaped with a backslash `\?`.
* `Group(...any)`: To easily group a number of expressions. Wraps them in parentheses and seperates them with commas.
* `Quote(...string)`: For quoting. [See details](#quotes)
* `P(any)`: To manually wrap an expression with parentheses. This is often not necessary as the parentheses will be added as the expression is built.

### Chaining

The type returned by the starter methods return have methods for common operators.  
**NOTE:** These are the common operators. Each dialect can sometimes include their own starters

* `IsNull()`: X IS NULL
* `IsNotNull()`: X IS NOT NULL
* `Is(y any)`: X IS DISTINCT FROM Y
* `IsNot(y any)`: X IS NOT DISTINCT FROM Y
* `EQ(y any)`: X = Y
* `NE(y any)`: X <> Y
* `LT(y any)`: X < Y
* `LTE(y any)`: X <= Y
* `GT(y any)`: X > Y
* `GTE(y any)`: X >= Y
* `In(...any)`: X IN (y, z)
* `NotIn(...any)`: X NOT IN (y, z)
* `Or(y any)`: X OR Y
* `And(y any)`: X AND Y
* `Concat(y any)`: X || Y

The following expressions cannot be chained and are expected to be used at the end of a chain

* `As(alias string)`: X as "alias". Used for aliasing column names

## Parameters

To prevent SQL injection, it is necessary to use parameters in our queries. With `bob` use `qm.Arg()` where necessary.  
This will write the placeholder correctly in the generated sql, and return the value in the argument slice.

```go
// args: 100, "Stephen"
// Postgres: SELECT * from users WHERE id = $1 AND name = $2
// MySQL: SELECT * from users WHERE id = ? AND name = ?
// SQL Server: SELECT * from users WHERE id = @p1 AND name = @p2
psql.Select(
    qm.From("users"),
    qm.Where(qm.X("id").EQ(qm.Arg(100))),
    qm.Where(qm.X("name".EQ(qm.Arg("Stephen"))),
).WriteQuery(w, 1)
```

## Raw Queries

As any good query builder, you are allowed to use your own raw SQL queries.
Either at the top level with `psql.Raw()` or inside any clause with `qm.Raw()`.

Another option is to use `qm.Raw()` which takes a clause and args. The placeholder in the clauses are question marks `?`.

```go
// SELECT * from users WHERE id = $1 AND name = $2
// args: 100, "Stephen"
psql.Select(
    qm.From("users"),
    qm.Where(qm.Raw("id = ? and name = ?", 100, "Stephen")),
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

The `WriteQuery` method is useful when we want to write to an exisiting `io.Writer`.  
However we often just want the query string and arguments. So the Query objects have the following methods:

* `Build() (query string, args []any, err error)`
* `BuildN(start int) (query string, args []any, err error)`
* `MustBuild() (query string, args []any) // panics on error`
* `MustBuildN(start int) (query string, args []any) // panics on error`

```go
queryString, args, err := psql.Select(...).Build()
```

Since the query is built from scratch every time the `WriteQuery()` method is called,
it can be useful to initialize the query one time and reuse where necessary.

For that, the `MustBuild()` function can be used. This panics on error.

```go
var myquery, myargs = psql.Insert(...).MustBuild()
```

## Roadmap

* **Postgres**
  * [x] Raw
  * [x] Select
  * [x] Insert
  * [x] Update
  * [x] Delete
  * [ ] Postgres Specific Operators
    * [ ] Is [Not] True
    * [ ] Is [Not] False
    * [ ] Is [Not] Unknown
    * [x] [Not] Between Symmetric
    * [ ] Is [Not] [NFC|NFD|NFKC|NFKD] Normalized
* **MySQL**
  * [x] Raw
  * [ ] Select
  * [ ] Insert
  * [ ] Update
  * [ ] Delete
* **SQLite**
  * [x] Raw
  * [x] Select
  * [x] Insert
  * [x] Update
  * [x] Delete
  * [ ] SQLite Specific Operators
    * [ ] GLOB
* **SQL Server**
  * [x] Raw
  * [ ] Select
  * [ ] Insert
  * [ ] Update
  * [ ] Delete
* **Common Operators**
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
  * [x] Between
