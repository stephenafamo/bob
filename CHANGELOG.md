# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## Added

- Add new properties `cmp_opts` and `cmp_opts_imports` to the `types` configuration. To enable better control over testing randomization.

## [v0.25.0] - 2024-01-20

### Changed

- Update `github.com/jaswdr/faker` dependency from `v1` to `v2`

## [v0.24.0] - 2024-01-20

### Added

- Add randomization for all primitive types
- Add test for factory type randomization
- Drivers are now required to send in type definitions for generated types
- Custom types can now be configured at a top level in the config file

### Changed

- Format generated files with `gofumpt`
- Replace key in `replacements` configuration is now a string referring to the type defined in the `types` configuration.

### Removed

- Remove `Imports` from column definition.

### Fixed

- `INTEGER` columns are now correctly generated as `int32` not `int`

## [v0.23.2] - 2024-01-04

### Fixed

- Fix panic when inferring `modify` for relationships with no primary key
- Skip factory enum generation if there are no enums
- Return `sql.ErrNoRows` when Insert/Upsert returns nothing

## [v0.23.1] - 2024-01-03

### Fixed

- Do not wrap `Setter.Expressions()` in parenthesis

## [v0.23.0] - 2024-01-03

### Added

- Add `bob.Cache()` which saves the built SQL and args to prevent rebuilding the same query multiple times.
- Add `As()` starter to alias expressions
- Add `OP()` builder method for using custom operators
- Add `table.InsertQ(ctx, db)` now includes the insert columns from the table model.
- It is now possible to configure additional constraints for code generation.
- Add `um.SetCol()` which maintains the old behavior of `um.Set()`.
- Generate additional `Expressions()` method for Setters to make it easier to use them in `um.Set()` or `im.Set()`.

### Changed

- Aliases configuration for code generation no longer has a top level `table` key
- When configuring relationships, `from_unique`, `to_unique`, `key_nullable` can no longer be configured. They are now inferred from the database.
- When configuring relationships, `to_key` has been changed to `modify` and should be set to `from`, `to` or `""` to indicate which side of the relationship to modify.
  If left empty, Bob will try to guess which side to modify based on the presence of primary keys and unique columns.
- `RelWhere.Value` is now `RelWhere.SQLValue`
- Change CONFLICT/DUPLICATE KEY UPDATE to use mods instead of a chainable methods.
- Change `um.Set()` to take a list of expressions.
- Rename Setter method from `Insert()` to `InsertMod()` to avoid confusion.

### Fixed

- Prevent generating duplicate relationships for many-to-many self-join relationships
- Correctly use table alias in generated relationship join mods
- Fix an issue where CTEs were encased in double parenthesis
- Fix invalid SQL generated when doing `JOIN USING`
- Correctly include "AS" in function query if alias is set
- Setters are also generated for tables that have relationships, even if they have no primary key
- Column aliases in CTEs are now correctly included in the final query
- Fix several issues with generating code for multi-sided relationships
- Fix an issue where loading many-to-many relationships cause no columns to be selected unless specified
- Fix an issue where many-to-many relationships would not be able to use nested loaders

## [v0.22.0] - 2023-08-18

### Added

- Expand expressions when used in Raw (thanks @RangelReale)
- Add `InsertQ`, `UpdateQ`, and `DeleteQ` methods to Table models to start INSERT, UPDATE and DELETE queries respectively.
- Allow column comment for replacement matching (thanks @jroenf)
- Add table query hooks to modify model queries
- Include `WhereOr` and `WhereAnd` to make it easier to combine multiple generated where clauses
- Print a warning if a replacement rule did not find a match (thanks @jacobmolby)

### Changed

- Export generated factory.Factory struct
- Allow Limit and Offset to be used as Arguments in PostgreSQL (thanks @RangelReale)
- Make model hooks take slices not single objects
- Return rows affected from `Exec()` method of view queries instead of `sql.Result`
- Chain comparison methods now take an `Expression` instead of `any`
- Table models now require the types to implement `orm.Table` and `orm.Setter` interfaces.

### Removed

- Remove UpdateAll and DeleteAll methods on the Table models.

### Fixed

- Honor Only and Except in sqlite driver
- Always surround subqueries with parenthesis when used as an expression
- Fix mysql `TablesInfo` method and make sure we don't exclude entire table when targeting columns (thanks @jacobmolby)
- Fix bug in sqlite foreign key and join table detection

## [v0.21.1] - 2023-05-22

### Fixed

- Fix `Upsert` and `UpsertAll` methods of `mysql.Table`

## [v0.21.0] - 2023-05-09

### Changed

- Force uniqueness of relationship names in `psql` driver

### Fixed

- Fix panic when attaching associated relationships
- Make getting a random integer for preloading thread-safe

## [v0.20.6] - 2023-04-25

### Fixed

- Check all members when loading relationships

## [v0.20.5] - 2023-04-14

### Fixed

- Fix panic in Insert/Attach Relationship loop

## [v0.20.4] - 2023-04-07

### Fixed

- Replace `huandu/go-clone` with `qdm12/reprint`

## [v0.20.3] - 2023-04-06

### Fixed

- Fix cloning bug by replacing `jinzhu/copier` with `huandu/go-clone`

## [v0.20.2] - 2023-04-05

### Fixed

- Account for windows when calculating models module path

## [v0.20.1] - 2023-04-04

### Fixed

- Update the generated code to use `github.com/gofrs/uuid/v5`
- Fix bug where auto-increment columns were marked as generated in the SQLite driver

## [v0.20.0] - 2023-04-03

### Added

- Add the `PreloadWhere` preload mod to filter what relation should be preloaded.
- `ViewQuery` now embeds `bob.BaseQuery` giving it additional methods like `Apply` and `Build`
- Add plugin support. Currently 3 plugin hooks are provided. `PlugState`, `PlugDBInfo` and `PlugTemplateData`.

### Changed

- Make `View.Name()` return a dialect-specific expression
- Improve `Debug` helpers.
  - `Debug` writes query output to stdout
  - `DebugToWriter` writes the query output to any `io.Writer` with a fallback to stdout.
  - `DebugToPrinter` prints the query with a given `bob.DebugPrinter`. Also falls back to stdout.
- Rename `OnlyColumns` to `PreloadOnly` and `ExceptColumns` to `PreloadExcept` to be more consistent with the newly added `PreloadWhere`.
- `JoinChain.On` now takes Expressions instead of `any`.
- `JoinChain.Using` now takes strings instead of `any`.
- Export `gen.TemplateData`

### Removed

- Remove `ILIKE` operator from MySQL since it is not supported.
- Removed `Destination()` method in driver interface.
- Removed `PackageName()` method in driver interface.

### Fixed

- Account for possible clashes between column and relationship alias
- Make `Preload` mods work the same way as other query mods
- Avoid overwriting manually flipped relationships.
- Account for potentially null relationship in Load methods

## [v0.19.1] - 2023-03-21

### Fixed

- Fix `On` method for JoinChain in `mysql` and `sqlite`

## [v0.19.0] - 2023-03-19

### Added

- Add `LIKE` and `ILIKE` operators
- Print generated files
- Add `no_reverse` option for user-configured relationships

### Changed

- Move common parts of loading to shared internal package

### Fixed

- Fix generated joins for multi-sided relationships

## [v0.18.2] - 2023-03-18

### Fixed

- Account for relationships with tables that do not have a primary key in models and factories
- Properly extract preloaders nested in `mods.QueryMods`
- Fix bug in preloading with multiple sides
- Fix issue with multi-sided relationships that include views

## [v0.18.1] - 2023-03-16

### Fixed

- Make self-referencing foreign keys work
- Tweak factory types to make collisions even less likely

## [v0.18.0] - 2023-03-12

### Changed

- Comparison methods now require an expression (`bob.Expression`)

### Removed

- No more `X` builder start function
- No more `P` builder method

## [v0.17.3] - 2023-03-11

### Fixed

- Fix bug with args in table updates
- Fix some typos in documentation (thanks @leonardtan13)

## [v0.17.2] - 2023-03-09

### Fixed

- Fix a bug when multiple multi-column foreign keys exist
- Multiple internal changes to the generator to make it easier to write a custom entrypoint
- More robust testing of code generatio
