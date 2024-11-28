---
sidebar_position: 3
description: How to use the Generated Models
---

# Usage

Given a database table like this:

```sql
CREATE TABLE jets (
    id serial PRIMARY KEY NOT NULL,
    pilot_id integer NOT NULL,
    airport_id integer NOT NULL,
    name text NOT NULL,
    color text,
    uuid text NOT NULL,
    identifier text NOT NULL,
    cargo bytea NOT NULL,
    manifest bytea NOT NULL
);
```

The following are generated:

:::note

A lot of other helpful methods and functions are generated, but let us look at these first

:::

```go
// Jet is an object representing the database table.
// The required methods to satisfy orm.Table are also generated
type Jet struct {
    ID         int              `db:"id,pk" json:"id" yaml:"id"`
    PilotID    int              `db:"pilot_id" json:"pilot_id" yaml:"pilot_id"`
    AirportID  int              `db:"airport_id" json:"airport_id" yaml:"airport_id"`
    Name       string           `db:"name" json:"name" yaml:"name"`
    Color      null.Val[string] `db:"color" json:"color,omitempty" yaml:"color,omitempty"`
    UUID       string           `db:"uuid" json:"uuid" yaml:"uuid"`
    Identifier string           `db:"identifier" json:"identifier" yaml:"identifier"`
    Cargo      []byte           `db:"cargo" json:"cargo" yaml:"cargo"`
    Manifest   []byte           `db:"manifest" json:"manifest" yaml:"manifest"`
}

// JetSetter is used for insert/upsert/update operations
// All values are optional, and do not have to be set
// Generated columns are not included
// The required methods to satisfy orm.Setter are also generated
type JetSetter struct {
    ID         omit.Val[int]        `db:"id,pk"`
    PilotID    omit.Val[int]        `db:"pilot_id"`
    AirportID  omit.Val[int]        `db:"airport_id"`
    Name       omit.Val[string]     `db:"name"`
    Color      omitnull.Val[string] `db:"color"`
    UUID       omit.Val[string]     `db:"uuid"`
    Identifier omit.Val[string]     `db:"identifier"`
    Cargo      omit.Val[[]byte]     `db:"cargo"`
    Manifest   omit.Val[[]byte]     `db:"manifest"`
}

// JetSlice is an alias for a slice of pointers to Jet.
// This should almost always be used instead of []*Jet.
type JetSlice []*Jet

// JetsTable contains methods to work with the jets table
var JetsTable = psql.NewTablex[*Jet, JetSlice, *JetSetter]("", "jets")
```

:::tip

**JetsTable** gives the full range of capabilities of a Bob model, including

- Flexible queries: One, All, Cursor, Count, Exists
- Expressions for names and column lists
- Hooks

[Read the documentation to see how to use](../models/table)

:::

## Generated Methods

### Update

Update values in the database based on the values set in the `Setter`.

```go
err := jet.Update(ctx, db, &models.JetSetter{
    AirportID: omit.From(100),
    Name:      omit.From("new name"),
})
```

### UpdateAll

UpdateAll is a method on the collection type `JetSlice`.

All rows matching the primary keys of the members of the slice are updated with the given values.

```go
err := jets.UpdateAll(ctx, db, &JetSetter{
    AirportID: omit.From(100),
})
```

### Delete

Delete a row from the database matching the primary key of the struct.

```go
_, err := jet.Delete(ctx, db)
```

### DeleteAll

DeleteAll is a method on the collection type `JetSlice`.

Works just like `Delete`, but for all the members of the slice.

```go
_, err := jets.DeleteAll(ctx, db)
```

### Reload

Reload all columns from the database into the struct

```go
_, err := jet.Reload(ctx, db)
```

### ReloadAll

ReloadAll is a method on the collection type `JetSlice`.

Works just like `Reload`, but for all the members of the slice.

```go
_, err := jets.ReloadAll(ctx, db)
```

## Generated Functions

The following helper methods are also generated.

### Find

A function for finding by primary key is also generated.

```go
// SELECT * FROM "jets" WHERE "jets"."id" = 10
models.FindJet(ctx, db, 10).All()
```

We can decide to only select a few columns

```go
// SELECT "jets"."id", "jets"."cargo" FROM "jets" WHERE "jets"."id" = 10
jet, err := models.FindJet(ctx, db, 10, "id", "cargo")
```

### Exists

Use exists to quickly check if a model with a given PK exists.

```go
hasJet, err := models.JetExists(ctx, db, 10).All()
```

## Generated Error Constants

Generated error constants allow for matching against specific errors raised by the underlying database driver.

Take the following database table, for example:

```sql
CREATE TABLE pilots (
    id serial PRIMARY KEY NOT NULL,
    first_name text NOT NULL,
    last_name text NOT NULL,
    UNIQUE (first_name, last_name)
);
```

Bob will define the following `struct` inside the generated `pilots.go` file:

```go
type pilotErrors struct {
    ErrUniqueFirstNameAndLastName *UniqueConstraintError
}
```

The `struct` is initialized and exported through a `PilotErrors` variable, which can be used for error matching in one of the following ways:


```go
pilot, err := models.Pilots.Insert(ctx, db, setter)
if errors.Is(models.PilotErrors.ErrUniqueFirstNameAndLastName, err) {
    // handle the error
}
```

or

```go
pilot, err := models.Pilots.Insert(ctx, db, setter)
if models.PilotErrors.ErrUniqueFirstNameAndLastName.Is(err) {
    // handle the error
}
```

Bob furthermore defines a generic `ErrUniqueConstraint` constant, which can be used for matching and handling all unique constraint errors:

```go
pilot, err := models.Pilots.Insert(ctx, db, setter)
if errors.Is(models.ErrUniqueConstraint, err) {
    // handle the error
}
```

or

```go
pilot, err := models.Pilots.Insert(ctx, db, setter)
if models.ErrUniqueConstraint.Is(err) {
    // handle the error
}
```

When using the `errors.Is()` variant, be mindful that the order of arguments matters due to Go's internal implementation. The first argument should be the error constant, while the second argument should be the actual error returned from the database. If you flip the arguments, the function won't work as intended and won't catch the unique constraint error.

## Query Building

Several constants[^1] are also generated to help with query building. As with all queries built with [Bob's query builder](../query-builder/intro), the building blocks are expressions and mods.

### Filters

To fluently build type safe queries, mods are generated to easily add `WHERE` filters.

```go
// SELECT * FROM "jets" WHERE "jets"."id" = 100
models.Jets.Query(
    ctx, db,
    models.SelectWhere.Jets.ID.EQ(100),
)
```

Where filters can be combined using `WhereOr` and `WhereAnd`. These can be nested to create even more complex queries.

```go
// SELECT * FROM "users"
// WHERE "users"."name" IS NULL
// OR "users"."email" IS NOT NULL
// OR ("users"."age" > 21 AND "users"."location" IS NOT NULL)
users, err := models.Users.Query(
    ctx, db,
    psql.WhereOr(
        models.SelectWhere.Users.Name.IsNull(),
        models.SelectWhere.Users.Email.IsNotNull(),
        psql.WhereAnd(
            models.SelectWhere.Users.Age.GT(21),
            models.SelectWhere.Users.Location.IsNotNull(),
        ),
    ),
).All()
```

Since each query type has its own mods, `SelectWhere`, `InsertWhere`, `UpdateWhere` and `DeleteWhere` are all generated.

:::tip

To use these filters with an aliased table name, use the `AliasedAs` method.

```go
models.SelectWhere.Users.AliasedAs("u").Name.IsNull()
```

:::

### Join Helpers

To make joining tables easier, join helpers are generated for each table. The generated joins are based on the [relationships defined for each table](./relationships).

```go
// SELECT * FROM "jets"
// INNER JOIN "pilots" ON "pilots"."id" = "jets"."pilot_id"
// INNER JOIN "airports" ON "airports"."id" = "jets"."airport_id"
models.Jets.Query(
    ctx, db,
    models.SelectJoins.Jets.InnerJoin.Pilots(ctx),
    models.SelectJoins.Jets.InnerJoin.Airports(ctx),
).All()
```

Since each query type has its own mods, `SelectJoins`, `InsertJoins`, `UpdateJoins` and `DeleteJoins` are all generated.

:::tip

To use these join mods with an aliased table name, use the `AliasedAs` method.

```go
models.SelectJoins.Jets.AliasedAs("j").InnerJoin.Airports(ctx).AliasedAs("a")
```

:::

### Column Expressions

For even more control, expressions are generated for every column to be used in any part of the query.

```go
// SELECT "jets"."name", count(1) FROM "jets"
// WHERE "jet"."id" BETWEEN 50 AND 5000
// GROUP BY "jets"."pilot_id"
// ORDER BY "jets"."pilot_id"
psql.Select(
    sm.Columns(models.JetColumns.Name, "count(1)"),
    sm.From(models.JetsTable.Name()),
    sm.Where(models.JetColumns.ID.Between(50, 5000)),
    sm.OrderBy(models.JetColumns.PilotID),
)

```

:::tip

To use these expressions with an aliased table name, use the `AliasedAs` method.

```go
models.JetColumns.AliasedAs("j").Name.IsNull()
```

:::

[^1]: Some are technically just global variables. But they are never mutated by Bob, or expected to be mutated by the user.
