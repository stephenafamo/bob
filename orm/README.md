# Bob ORM

Generate an ORM based on your database schema

Pending features

* Relationship methods
  * [x] Add
  * [x] Attach
  * [ ] Remove
* Find
* Exists (by PK)
* Back-Referencing when loading/adding relationships

## Usage

### PostgreSQL

```sh
DSN=postgres://user:pass@host:port/dbname go run github.com/go-bob/bobgen-psql@latest
```

## About

This is largely based on [SQLBoiler](https://github.com/volatiletech/sqlboiler),
however, many large scale improvements have been made.

1. Query building is based on [Bob](https://github.com/stephenafamo/bob), which is dialect specific and allows for far more possiblites and less bugs.
1. Composite primary keys and foreign keys are now supported.
1. `qm.Load` is entirely reworked.
    1. Loaders are generated specifically for each relation and can be nested to load sub-objects
    1. `qm.Load("relationship")` is split into `PreloadPilotJets` and `ThenLoadPlotJets`. The `Preload` variants load the relationship in a single call using left joins
       while the `ThenLoad` variants make a new query similar to how the currentl `qm.Load` works.
1. All the Column names are now in a single top level variable similar to table names.
1. Where helpers are in a top level global variables split down into query types. e.g. `SelectWhere.Pilot.ID.EQ(10)`
1. Enums types are generated for every enum in the schema, whether or not they were used in a column.
1. Enums are properly detected even if they are used only as an array.
1. Nullable types are now just their concrete type with a generic wrapper. Individual null type variants are no longer needed
1. Inserts and Upserts are not done with the concrete model type, but with an `Optional` version where every field has to be explicitly set. Removes the need for `boil.Infer()`
1. Column lists are generated for every table, which can be filtered with `Only` or `Except` and are built to the correctly quoted versions.
1. Hooks now return a context which will be chained and eventually passed to the query.
1. AutoTimestamps are not implemented.
1. Soft Deletes are not implemented.

Like SQLBoiler this is a tool to generate a Go ORM tailored to your database schema.

It is a "database-first" ORM.  That means you must first create your database schema. Please use something
like [sql-migrate](https://github.com/rubenv/sql-migrate) or some other migration tool to manage this part of the database's life-cycle.

### Features

* Full model generation
* Extremely fast code generation
* High performance through generation & intelligent caching
* Uses bob.Executor (simple interface, sql.DB, sql.Tx, sqlx.DB etc. compatible)
* Uses context.Context
* Easy workflow (models can always be regenerated, full auto-complete)
* Strongly typed querying (usually no converting or binding to pointers)
* Hooks (Before/After Select/Insert/Update/Delete/Upsert)
* Table and column whitelist/blacklist
* Custom struct tags
* Raw SQL fallback
* Basic multiple schema support (no cross-schema support)
* 1d arrays, json, hstore & more
* Enum types
* Out of band driver support
* Support for database views
* Supports generated/computed columns
* Materialized view support
* Multi-column foreign key support
* Relationships/Associations
  * Eager loading (recursive)
  * Automatically detects relationships based on foreign keys
  * Can load related models both by a left-join and a 2nd query
  * Supports user-configured relationships
  * Can configure relationships based on static column values. For example, (`WHERE object_type = 'car' AND object_id = cars.id`)
  * Support for `has-one-through` and `has-many-through`.

## Missing features

* No automatic timestamps (createdAt/UpdatedAt)
* No soft delete support

## Supported Databases

| Database          | Driver Location |
| ----------------- | --------------- |
| PostgreSQL        | <https://github.com/go-bob/bobgen-psql> |

