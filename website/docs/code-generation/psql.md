---
sidebar_position: 11
title: PostgreSQL Driver
description: ORM Generation for PostgreSQL
---

# Bob Gen for Postgres

Generates an ORM based on a postgres database schema

## Usage

```sh
# With env variable
PSQL_DSN=postgres://user:pass@host:port/dbname go run github.com/stephenafamo/bob/gen/bobgen-psql@latest

# With configuration file
go run github.com/stephenafamo/bob/gen/bobgen-psql@latest -c ./config/bobgen.yaml
```

### Driver Configuration

#### [Link to general configuration and usage](./configuration)

The configuration for the postgres driver must all be prefixed by the driver name. You must use a configuration file or environment variables for configuring the database driver.

In the configuration file for postgresql for example you would do:

```yaml
psql:
  dsn: "postgres://user:pass@host:port/dbname"
```

When you use an environment variable it must also be prefixed by the driver name:

```sh
PSQL_DSN="postgres://user:pass@host:port/dbname"
```

Additionally if ssl mode is to be disabled (you will get connection failed error - `unable to fetch table data: unable to load enums: pq: SSL is not enabled on the server`), you can add `sslmode` to the dsn:

```sh
PSQL_DSN="postgres://user:pass@host:port/dbname?sslmode=disable"
```

The values that exist for the drivers:

| Name          | Description                                       | Default                  |
| ------------- | ------------------------------------------------- | ------------------------ |
| dsn           | URL to connect to                                 |                          |
| driver        | Driver to use for generating driver-specific code | `github.com/lib/pq`      |
| schemas       | Schemas find tables in                            | ["public"]               |
| shared_schema | Schema to not include prefix in model             | first value in "schemas" |
| output        | Folder for generated files                        | "models"                 |
| pkgname       | Package name for generated code                   | "models"                 |
| uuid_pkg      | UUID package to use (gofrs or google)             | "gofrs"                  |
| concurrency   | How many tables to fetch in parallel              | 10                       |
| only          | Only generate these                               |                          |
| except        | Skip generation for these                         |                          |

## Driver-specific code

The `driver` configuration option enables Bob to generate code that is tailored to the specifics of the selected `database/sql` driver.

For Postgres, the supported drivers are:

- [github.com/lib/pq](https://pkg.go.dev/github.com/lib/pq) (default)
- [github.com/jackc/pgx](https://pkg.go.dev/github.com/jackc/pgx)
- [github.com/jackc/pgx/v4](https://pkg.go.dev/github.com/jackc/pgx/v4)
- [github.com/jackc/pgx/v5](https://pkg.go.dev/github.com/jackc/pgx/v5)

Bob leverages driver-specific code to perform precise error matching for [generated error constants](./usage#generated-error-constants).

## Only/Except:

The `only` and `except` configuration options can be used to specify which tables to include or exclude from code generation. You can either supply a list of table names or use regular expressions to match multiple tables.

Consider the example below:

```yaml
psql:
  only:
    "/^foo/":
    bar_baz:
```

This configuration only generates models for tables that start with `foo` and the table named `bar_baz`.

Alternatively, the following example excludes these tables from code generation rather than including them:

```yaml
psql:
  except:
    "/^foo/":
    bar_baz:
```

You may also exclude specific columns:

```yaml
psql:
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
