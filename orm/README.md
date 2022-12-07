# Bob ORM

Generate an ORM based on your database schema

Pending features

* Relationship methods
  * [x] Add
  * [x] Attach
  * [ ] Remove
* Back-Referencing when loading/adding relationships

## Usage

### PostgreSQL

```sh
PSQL_DSN=postgres://user:pass@host:port/dbname go run github.com/go-bob/bobgen-psql@latest
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
1. Simplified configuration for relationship aliases. No more need for foreign/local.

Like SQLBoiler this is a tool to generate a Go ORM tailored to your database schema.

It is a "database-first" ORM.  That means you must first create your database schema. Please use something
like [sql-migrate](https://github.com/rubenv/sql-migrate) or some other migration tool to manage this part of the database's life-cycle.

## Features

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

### Missing features

* No automatic timestamps (createdAt/UpdatedAt)
* No soft delete support

## Supported Databases

| Database          | Driver Location |
| ----------------- | --------------- |
| PostgreSQL        | <https://github.com/go-bob/bobgen-psql> |

## Configuration

Configuration is done in a bobgen.yaml (also supports toml and json) file in the same directory.
A different configuration file can be passed with the `-c` or `--config` flag.

Create a configuration file. Because the project uses [viper](https://github.com/spf13/viper), TOML, JSON and YAML are all usable.  
Environment variables are also able to be used, but certain configuration options cannot be properly expressed using environmental variables.

The configuration file should be named `bobgen.yaml` and is searched for in
the following directories in this order:

* `./`
* `$XDG_CONFIG_HOME/bobgen/`
* `$HOME/.config/bobgen/`

We will assume YAML for the rest of the documentation.

### Database Driver Configuration

The configuration for a specific driver (in these examples we'll use `psql`)
must all be prefixed by the driver name. You must use a configuration file or
environment variables for configuring the database driver; there are no
command-line options for providing driver-specific configuration.

In the configuration file for postgresql for example you would do:

```yaml
psql:
  dsn: "postgres://user:pass@host:port/dbname"
```

When you use an environment variable it must also be prefixed by the driver
name:

```sh
PSQL_DSN="postgres://user:pass@host:port/dbname"
```

The values that exist for the drivers:

| Name | Required | Postgres Default |
| ---- | -------- | ---------------- |
| schema    | no        | "public"  |
| dsn       | yes       | none      |
| whitelist | no        | []        |
| blacklist | no        | []        |

Example of whitelist/blacklist:

```yaml
psql:
    # Removes migrations table, the name column from the addresses table, and
    # secret_col of any table from being generated. Foreign keys that reference tables
    # or columns that are no longer generated because of whitelists or blacklists may
    # cause problems.
    blacklist: ["migrations", "addresses.name", "*.secret_col"]
```

### General configuration options

You can also pass in these top level configuration values if you would prefer
not to pass them through the command line or environment variables:

| Name                | Defaults  | Description |
| ------------------- | --------- | ----------- |
| pkgname             | "models"  | The package name for the generated models |
| output              | "models"  | The relative path of the output folder |
| concurrency         | 10        | How many tables to fetch in parallel |
| tag-ignore          | []        | List of column names that should have tags values set to '-' |
| relation-tag        | "-"       | Struct tag for the relationship object |

#### Type Replacements

There exists the ability to override types that the driver has inferred.
The way to accomplish this is through the config file.

```yaml
types:
  # Tables is used to limit the replacement to only specific tables
  # if not present, it will match in all tables
  tables: ["users", "authors"]

  # The match is a drivers.Column struct, and matches on almost all fields.
  # Notable exception for the unique bool. Matches are done
  # with "logical and" meaning it must match all specified matchers.
  # Boolean values are only checked if all the string specifiers match first,
  # and they must always match.
  #
  # Not shown here: db_type is the database type and a very useful matcher
  #
  # Note there is precedence for types.match, more specific things should appear
  # further down in the config as once a matching rule is found it is executed
  # immediately.
  match:
    type: "null.String"

  # The replace is what we replace the strings with. You cannot modify any
  # boolean values in here. But we could change the Go type (the most useful thing)
  # or the DBType or FullDBType etc. if for some reason we needed to.
  replace:
    type: "mynull.String"
    imports: ['"github.com/me/mynull"']
```

#### Relationships

Relationships are automatically inferred from foreign keys.
However, in certain cases, it is either not possible or not desireable to add a foreign key relationship.

We can manually describe relationships in the configuration:

```yaml
relationships:
  users: # The table name
    - name: "custom_videos_relationship" # A unique identifier used to configure aliases
      sides:
        - from: "users" # Name of the source of the relationship
          to: "videos" # Table name of the other side of the relation
          # mapping of columns from source to destination
          columns:
            - [id, user_id]

          # Is there a unique constraint on the destination columns?
          # this is used to determine if it is a to-one or to-many relationship
          to_unique: false

          # Is the "key" on the destination table?
          # This is used to determine what side to set.
          # For example, if `users.id` -> `videos.user_id,` `to_key` = true
          # so in the generated code, we know to set `videos.user_id` and not `users.id`
          to_key: true
```

##### Related Through

The configuration also allows us to describe relationships that span multiple tables.  
We achieve this by having multiple `sides`.

In this example configuration, we add a relationship of users to videos through teams.  
The generated user model with have a `Videos` relation.

```yaml
relationships:
  users:
    - name: "users_to_videos_through_teams"
      sides:
        - from: "users"
          to: "teams"
          columns: [[team_id, id]]
          to_unique: true
          to_key: false
        - from: "teams"
          to: "videos"
          columns: [[id, team_id]]
          to_unique: false
          to_key: true
```

##### Related Where

The configuration also allows us to describe relationships that are not only based on matching columns but also columns with static values.  
For example, we may want to add a relationship to teams for verified members.

```yaml
relationships:
  users:
    - name: "users_to_videos_through_teams"
      sides:
        - from: "teams"
          to: "users"
          columns: [[id, team_id]]
          to_unique: false
          to_key: true
          to_where:
            - column: "verified"
              value: "true"
```

#### Aliases

Names are automatically generated for you. If you name your
database entities properly you will likely have descriptive names generated in
the end. However in the case where the names in your database are bad AND
unchangeable, or bob's inference doesn't understand the names you do have
(even though they are good and correct) you can use aliases to change the name
of your tables, columns and relationships in the generated Go code.

*Note: It is not required to provide all parts of all names. Anything left out
will be inferred as it was in the past.*

```yaml
# Although team_names works fine without configuration, we use it here for illustrative purposes
aliases:
  tables:
    team_names:
      up_plural: "TeamNames"
      up_singular: "TeamName"
      down_plural: "teamNames"
      down_singular: "teamName"
      columns: # Columns can be aliased by name
        uuid: "ID"
      relationships: # Relationships can be aliased by name
        team_id_fkey: "Owner"
```

#### Inflections

With inflections, you can control the rules used to generate singular/plural variants. This is useful if a certain word or suffix is used multiple times and you do not wnat to create aliases for every instance.

```yaml
inflections:
  plural: # Rules to convert a suffix to its plural form
    ium: ia
  plural_exact: # Rules to convert an exact word to its plural form
    stadium: stadia
  singular: # Rules to convert a suffix to its singular form
    ia: "ium"
  singular_exact: # Rules to convert an exact word to its singular form
    stadia: "stadium"
  irregular: # Singular -> plural mappings of exact words that don't follow conventional rules
    radius: "radii"
```
