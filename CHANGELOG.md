# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [v0.30.0] - 2025-01-16

### Added

- Added a new field `QueryFolders` to `gen/drivers.DBInfo` for drivers to be able to include information about parsed queries.
- Added `gen/QueriesTemplates` which in the future will contain base templates for generating code for parsed qureries.
- Added a `QueryTemplate` field to `bobgen_helpers.Templates` for drivers to include additional templates for queries.
- Added a new reserved output key `queries`. This is handled specially for each query folder supplied by the driver.
- Added new `wm` package to each dialect for mods that modify `Window` clauses.
- Added a new method `Alias` for `View` struct, for each dialect. It returns the alias of the view. (thanks @Nitjsefni7)

### Changed

- Updated error constant generation to employ specific error types for making error matching easier. (thanks @mbezhanov)
- Collation in `clause.OrderDef` is now a string not an expression and is always quoted
- Calling `UpdateAll`, `DeleteAll` and `ReloadAll` on an empty model slice now returns nil without running any queries.
- `UNION`, `INTERSECT` and `EXCEPT` mods now append to the query instead of replacing it.
- Generated files now end with `.bob.go` instead of `.go` and are always cleaned up before generating new files. Singleton templates are now required to have a `.bob.go.tpl` extension.
- The expected structure for templates have been changed:
  - Previously, singleton templates should be kept in a `singleton` folder. Now, any template not inside a folder is considered a singleton template.
  - Previoulsy, templates in the root folder are merged and run for each table. Now, this will happen to templates in the `table/` folder.
  - Previoulsy, the entire file tree and every subdirectory is walked to find templates. Now only templates in the root folder and the `table/` folder are considered.
- Change `From` in `clause.Window` to `BasedOn` to avoid confusion with `FromPreceding` and `FromFollowing`. Also change `SetFrom` to `SetBasedOn`.
- Embed `clause.OrderBy` in `clause.Window` to make it possible to reuse `OrderBy` mods in window definitions.
- Change the `Definition` field in `clause.NamedWindow` from `any` to `clause.Window` for extra type safety.
- `sm.Window` now takes mods to modify the window clause.
- `fm.Over` now takes mods to modify the window for the window function.

### Deprecated

- Deprecated the `wipe` option to delete all files in the output folder. Files are now generated with a `.bob.go` extension and are always cleaned up before generating new files.

### Removed

- Remove redundatnt type parameters from `orm.ExecQuery`.
- Remove unnecessary interface in `orm.Query` and `orm.ExecQuery`.
- Remove the redundant `clause.IWindow` interface.
- Remove `dialect.WindowMod` and `dialect.WindowMods` which use chainable methods to modify `Window` clauses. This is now handled by the `wm` package which used mods.

### Fixed

- Fix random value generation for pq.Float64Array factory (thanks @felipeparaujo)
- Using the `UpdateMod()` and `DeleteMod()` methods on an empty model slice now appends `WHERE primary_key IN NULL` which will return no results. Instead of `WHERE primary_key IN ()` which is a syntax error.
- Ensure `net/netip` is imported for the `pgtypes.Inet` random expression (thanks @plunkettscott)
- Fix a data race when adding enum types.
- Fix missing schema in table alias in pkEQ and pkIN clauses (thanks @adatob).

## [v0.29.0] - 2024-11-20

### Added

- Added error constants for matching against both specific and generic unique constraint errors raised by the underlying database driver. (thanks @mbezhanov)
- Added support for regular expressions in the `only` and `except` table filters. (thanks @mbezhanov)
- Added `ContextualMods` which are similar to regular mods but take a context argument. They are applied whenever the query is built.  
  This makes it cleaner to do certain things, like populating the select columns of a model if none was explicitly added.  
  The previous way this was done was unreliable since using `q.MustBuild()` would not add the columns while `bob.MustBuild(q)` will add them correctly.
- `modelSlice.UpdateMod()` and `modelSlice.DeleteMod()` are new methods that returns a mod for update and delete queries on a slice of models.  
  It adds `WHERE pk IN (pk1, pk2, pk3, ...)` to the query, and also schedule running the **hooks**.
- Added `bob.ToMods` which a slice of structs that implement `bob.Mod[T]` to a Mod. This is useful since Go does not allow using a slice of structs as a slice of an interface the struct implements.
- Added `bob.HookableQuery` interface. If a query implements this interface, the method `RunHooks(ctx, exec)` will be called before the query is executed.
- Added `bob.HookableType` interface. If a type implements this interface, the method `AfterQueryHook(ctx, exec, bob.QueryType)` will be called after the query is executed.  
  This is how `AfterSeleect/Insert/Update/DeleteHooks` hooks are now implemented.
- Added `Type() QueryType` method to `bob.Query` to get the type of query it is. Available constants are `Unknown, Select, Insert, Update, Delete`.
- Postgres and SQLite Update/Delete queries now refresh the models after the query is executed. This is enabled by the `RETURNING` clause, so it is not available in MySQL.
- Added the `Case()` starter to all dialects to build `CASE` expressions. (thanks @k4n4ry)
- Added `bob.Named()` which is used to add named arguments to the query and bind them later.
- Added `bob.BindNamed` which takes an argument (struct, map, or a single value type) to be used to bind named arguments in a query. See changes to `bob.Prepare()` for details of which type can be used.
- Indexes now include more information such as the type, unique and comment fields.
- Constraints now include a comment field.
- Added `Checks` field to DBConstraints so that drivers can also load check constraints. (not yet supported by the SQLite driver).
- Added comments field to Table definitions.

### Changed

- `context.Context` is now passed to `Query.WriteQuery()` and `Expression.WriteSQL()` methods. This allows for more control over how the query is built and executed.  
  This change made is possible to delete some hacks and simplify the codebase.
  - The `Name()` and `NameAs()` methods of Views/Tables no longer need the context argument since the context will be passed when writing the expression. The API then becomes cleaner.
  - Preloading mods no longer need to store a context internally. `SetLoadContext()` and `GetLoadContext()` have removed.
  - The `ToExpr` field in `orm.RelSide` which was used for preloading is no longer needed and has been removed.
- Moved `orm.Hooks` to `bob.Hooks` since it should not be limited to only ORM queries.
- Moved `mods.QueryModFunc` to `bob.ModFunc` since it should be available to all packages.
- The mod capability for `orm.Setter` is now reversed. It should now be a mod for Insert and have a method that returns a mod for Update.  
   This makes more sense since one would at most use one setter during updates, but can use multiple setters in a bulk insert.
- `table.InsertQ` has been renamed to `table.Insert`. The old implementation of `Insert` has been removed.  
   The same functionality can be achieved in the following way:

  ```go
  //----------------------------------------------
  // OLD WAY
  //----------------------------------------------
  user, err := models.Users.Insert(ctx, db, setter) // insert one
  users, err := models.Users.InsertMany(ctx, db, setters...) // insert many

  //----------------------------------------------
  // NEW WAY
  //----------------------------------------------
  user, err := models.Users.Insert(setter).One(ctx, db) // insert one
  users, err := models.Users.Insert(setters[0], setters[1]).All(ctx, db) // insert many

  // For cases where you already have a slice of setters and you want to pass them all, you can use `bob.ToMods`
  users, err := models.Users.Insert(bob.ToMods(setters)).All(ctx, db) // insert many
  ```

- `table.UpdateQ` has been renamed to `table.Update`. The old implementation of `Update` has been removed.  
   The same functionality can be achieved by using `model.Update()` or `modelSlice.UpdateAll()`.
- `table.DeleteQ` has been renamed to `table.Delete`. The old implementation of `Delete` has been removed.  
   The same functionality can be achieved by using `modelSlice.DeleteAll()` or creating an `Delete` query using `table.Delete()`.
- `BeforeInsertHooks` now only takes a single `ModelSetter` at a time.  
   This is because it is not possible to know before executing the queries exactly how many setters are being used since additional rows can be inserted by applying another setter as a mod.
- `bob.Cache()` now requires an `Executor`. This is used to run any query hooks.
- `bob.Prepare()` now requires a type parameter to be used to bind named arguments. The type can either be:
  - A struct with fields that match the named arguments in the query
  - A map with string keys. When supplied, the values in the map will be used to bind the named arguments in the query.
  - When there is only a single named argument, one of the following can be used:
    - A primitive type (int, bool, string, etc)
    - `time.Time`
    - Any type that implements `driver.Valuer`.
- `Index` columns are no longer just strings, but are a struct to include more information such as the sort order.

### Removed

- Remove MS SQL artifacts. (thanks @mbezhanov)
- Remove redundant type parameter from `bob.Load`.
- Removed `Before/AfterUpsertMods`. Upserts are really just inserts with a conflict clause and should be treated as such.
- Removed `Insert/InsertMany/Upsert/UpsertMany` methods from `orm.Table` since they are not needed.  
  It is possible to do the same thing, with similar effor using the the `InsertQ` method (which is now renamed to `Insert`).
- Remove `Update` and `Delete` methods from `orm.Table` since they are not needed.  
  It is possible to do the same thing, with similar effor using the the `UpdateQ` and `DeleteQ` methods (which are now renamed to `Update` and `Delete`).
- `context.Context` and `bob.Executor` are no longer passed when creating a Table/ViewQuery. It is now passed at the point of execution with `Exec/One/All/Cursor`.
- Remove `Prepare` methods from table and view qureries. Since `bob.Prepare()` now takes a type parameter, it is not possible to prepare from a method since Go does not allow additional type parameters in methods.
- Removed the **Prisma** and **Atlas** code generation drivers. It is better for Bob to focus on being able to generate code from the database in the most robust and detailed way and if the user wants, they can use other tools (such as prisma and atlas) to manage migrations before the code generation.
- Removed `Expressions` from Index definitions. It is now merged with the `Columns` field with an `IsExpression` field to indicate if the column is an expression.

### Fixed

- Removed unnecessary import of `strings` in `bobfactory_random.go`.
- Fixed data races in unit tests. (thanks @mbezhanov)
- Fixed invalid SQL statements generated by `sm.OrderBy().Collate()`. (thanks @mbezhanov)
- Fixed a bug preventing specific columns from being excluded when generating models from SQLite. (thanks @mbezhanov)
- Fixed an issue where invalid code is generated if a configured relationship has `from_where` or `to_where`.
- Fixed `ModelSlice.ReloadAll()` method for models with multiple primary keys.

## [v0.28.1] - 2024-06-28

### Fixed

- Also add the enum to the type array if an array of the enum is added. This is to prvent issues if the enum is only used in an array.
- Handle null column names in expression indexes. (thanks @mbezhanov)
- CROSS JOINS now allow aliases

## [v0.28.0] - 2024-06-25

### Added

- Added the `pgtypes.Inet` for `inet` type in PostgreSQL. (thanks @gstarikov)
- Added the `pgtypes.Macaddr` for `macaddr` and `macaddr8` types in PostgreSQL.
- Added the `pgtypes.LSN` type for the `pg_lsn` type in PostgreSQL.
- Added the `pgtypes.TxIDSnapshot` type for the `txid_snapshot` type in PostgreSQL.
- Added the `pgtypes.TSVector` type for the `tsvector` type in PostgreSQL.
- Added `AliasOf` property to codegen type definitions to allow for defining types that have their own randomization logic.
- Added `DependsOn` property to codegen type definitions to allow for defining types that depend on other types. This ensures that necessary code for the dependent types is generated.
- Add `xml` type definition for custom randomization logic.
- Add the `Cast()` starter to all dialects to build `CAST(expr AS type)` expressions.
- Load index information for MySQL, PostgreSQL, and SQLite tables. (thanks @mbezhanov)

### Changed

- Changed the `parray` package to `pgtypes`.
- Moved `HStore` to the `pgtypes` package.
- Simplified how random expressions are written for types by using standalone functions instead of a single generic function.
- The default `DebugPrinter` now prints args a bit more nicely by using thier `Value()` method if they implement the `driver.Valuer` interface.
- Only generate 2 random values when testing random expressions.

### Removed

- Removed `types.Stringer[T]`. It makes assumptions for how the type should be scanned and is not reliable.

### Fixed

- Do not add `FROM` clause to `SELECT` queries that are used as subqueries.
- Enum values are now checked for validity after scanning.

## [v0.27.1] - 2024-06-05

### Fixed

- Fixed bug in `Count()` queries not removing the offset from the original query. (thanks @daddz)

## [v0.27.0] - 2024-06-05

### Added

- Add PreloadAs PreloadOption to override the join alias when preloading a relationship with a left join. (thanks @daddz)
- Add `AliasedAs()` method to `tableColumns` and `tableWhere` types to use a custom alias.
- Add `AliasedAs()` method to generated relationship join mods. This is avaible in two places:

  - one to change the alias of the table being queried

    ```go
    models.SelectJoins.Jets.AliasedAs("j").InnerJoin.Pilots(ctx)
    ```

  - and the other to change the alias of the relationship.

    ```go
    models.SelectJoins.Jets.InnerJoin.Pilots(ctx).AliasedAs("p")
    ```

- Add `fm` mods to all supported dialects (psql, mysql and sqlite). These are mods for functions and are used to modify the function call. For example:

  ```go
  // import "github.com/stephenafamo/bob/dialect/psql/fm"
  psql.F( "count", "*",)(fm.Filter(psql.Quote("status").EQ(psql.S("done"))))
  ```

- Add `MustCreate`, `MustCreateMany`, `CreateOrFail` and `CreateManyOrFail` methods to generated factory Templates

### Changed

- Change the function call point for generated relationship join mods. This reduces the amount of allocations and only does the work for the relationship being used.

  ```go
  // Before
  models.SelectJoins(ctx).Jets.InnerJoin.Pilots
  // After
  models.SelectJoins.Jets.InnerJoin.Pilots(ctx)
  ```

- Changed the `Count()` function on `Views` to clone the query instead of changing the existing one. This makes queries reusable and the `Count()` function to behave as one would expect.

  ```go
  // This now works as expected
  query := models.Jets.Query(ctx, db, /** list of various mods **/)
  count, err := query.Count()
  items, err := query.All()
  ```

- Changed how functions are modified. Instead of chained methods, the `F()` starter now returns a function which can be called with mods:

  ```go
  // Before
  psql.F( "count", "*",).FilterWhere(psql.Quote("status").EQ(psql.S("done"))),
  // After
  // import "github.com/stephenafamo/bob/dialect/psql/fm"
  psql.F( "count", "*",)(fm.Filter(psql.Quote("status").EQ(psql.S("done")))),
  ```

  This makes it possible to support more queries.

- Use `netip.Addr` instead of `netip.Prefix` for Postgres `cidr` type.
- Use `decimal.Decimal` instead of `string` for Postgres `money` type.
- Use `net.HardwareAddr` for Postgres `macaddr8` type, in addition to the `macaddr` type.
- Code generation now generates struct tags for the generated model Setters as well, if configured through the `Tags` configuration option. Previoulsy, only the model struct fields were tagged. (thanks @singhsays)

### Removed

- Remove `TableWhere` function from the generated code. It was not used by the rest of the generated code and offered no clear benefit.
- Removed `As` starter. It takes an `Expression` and is not needed since the `Expression` has an `As` method which can be used directly.

### Fixed

- Fix a bug with `types.Stringer[T]` where the wrong value was returned in the `Value()` method.

## [v0.26.1] - 2024-05-26

### Fixed

- Use `netip.Prefix` instead of `netip.Addr` for postgres inet column type. This makes it possible to contain a subnet.
- Allow underscores in enum variable names.
- Fix an issue with title casing enum values

## [v0.26.0] - 2024-05-21

### Added

- Add `bobgen-sql` a code generation driver for SQL schema files. Supports PostgreSQL and SQLite.
- Add new properties `compare_expr` and `compare_expr_imports` to the `types` configuration. This is used when comparing primary keys and in testing.
- Add `never_required` to relationships configuration. This makes sure the factories does not require the relationship to be set. Useful if you're not using foreign keys. (thanks @jacobmolby)
- Add wrapper types for Stringer, TextMarshaler/Unmarshaler, and BinaryMarshaler/Unmarshaler to the `types` configuration.
- Make generated enum types implement the `fmt.Stringer`, `encoding.TextMarshaler`, `encoding.TextUnmarshaler`, `encoding.BinaryMarshaler` and `encoding.BinaryUnmarshaler` interfaces.

### Fixed

- Properly detect reference columns for implicit foreign keys in SQLite.
- Fix panic when generating random values for nullable columns. (thanks @jacobmolby)
- Sort relationships and imports for deterministic generated code. (thanks @jacobmolby)
- Correctly allow `OVER ()` with an empty window definition in PostgreSQL. (thanks @relvacode)
- Set `GROUP BY` to `NULL` if there are no expressions to group by.
- Replace symbols in enum values with their unicode point to make them valid Go identifiers.
- Properly detect implicit foreign keys in SQLite.
- Fix issue with attaching multi-sided relationships. (thanks @jacobmolby)

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
