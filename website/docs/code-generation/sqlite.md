---
sidebar_position: 13
title: SQLite Driver
description: ORM Generation for SQLite
---

# Bob Gen for SQLite

Generates an ORM based on a sqlite database schema

## Usage

```sh
# With env variable
SQLITE_DSN=test.db go run github.com/stephenafamo/bob/gen/bobgen-sqlite@latest

# With configuration file
go run github.com/stephenafamo/bob/gen/bobgen-sqlite@latest -c ./config/bobgen.yaml
```

### Driver Configuration

#### [Link to general configuration and usage](./configuration)

The configuration for the sqlite driver must all be prefixed by the driver name. You must use a configuration file or environment variables for configuring the database driver.

In the configuration file for sqlite for example you would do:

```yaml
sqlite:
  dsn: "file.db"
```

When you use an environment variable it must also be prefixed by the driver name:

```sh
SQLITE_DSN="file.db"
```

The values that exist for the drivers:

| Name          | Description                                       | Default              |
| ------------- | ------------------------------------------------- | -------------------- |
| dsn           | Path to database                                  |                      |
| driver_name   | Driver to use for generating driver-specific code | `modernc.org/sqlite` |
| attach        | Schemas to attach and the path to the db          | map[string]string{}  |
| shared_schema | Schema to not include prefix in model             | "main"               |
| queries       | List of folders containing query files            | []string{}           |
| output        | Folder for generated files                        | "models"             |
| pkgname       | Package name for generated code                   | "models"             |
| only          | Only generate these                               |                      |
| except        | Skip generation for these                         |                      |

## Driver-specific code

The `driver_name` configuration option enables Bob to generate code that is tailored to the specifics of the selected `database/sql` driver.

For SQLite, the supported drivers are:

- [modernc.org/sqlite](https://pkg.go.dev/modernc.org/sqlite) (default)
- [github.com/mattn/go-sqlite3](https://pkg.go.dev/github.com/mattn/go-sqlite3)

Bob leverages driver-specific code to perform precise error matching for [generated error constants](./usage#generated-error-constants).

## Only/Except

The `only` and `except` configuration options can be used to specify which tables to include or exclude from code generation. You can either supply a list of table names or use regular expressions to match multiple tables.

Consider the example below:

```yaml
sqlite:
  only:
    "/^foo/":
    bar_baz:
```

This configuration only generates models for tables that start with `foo` and the table named `bar_baz`.

Alternatively, the following example excludes these tables from code generation rather than including them:

```yaml
sqlite:
  except:
    "/^foo/":
    bar_baz:
```

You may also exclude specific columns:

```yaml
sqlite:
  # Removes public.migrations table, the name column from the addresses table, and
  # secret_col of any table from being generated. Foreign keys that reference tables
  # or columns that are no longer generated may cause problems.
  except:
    public.migrations:
    public.addresses:
      - name
    "*":
      - secret_col
```
