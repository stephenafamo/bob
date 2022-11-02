# Bob Gen

Generates an ORM based on a postgres database schema

Pending features

* Relationship methods
  * [ ] Set
  * [x] Add
  * [x] Attach
  * [ ] Remove
* Find
* Exists (by PK)
* Back-Referencing when loading/adding relationships

## Usage

```sh
DSN=postgres://user:pass@host:port/dbname go run github.com/go-bob/bobgen-psql@latest
```

## About

This is largely based on [SQLBoiler](https://github.com/volatiletech/sqlboiler), and is currently an experiment for version 5.

Many large scale improvements have been made.

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

