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

The values that exist for the drivers:

| Name          | Description                           | Default                  |
| ------------- | ------------------------------------- | ------------------------ |
| dsn           | URL to connect to                     |                          |
| schemas       | Schemas find tables in                | ["public"]               |
| shared_schema | Schema to not include prefix in model | first value in "schemas" |
| output        | Folder for generated files            | "models"                 |
| pkgname       | Package name for generated code       | "models"                 |
| uuid_pkg      | UUID package to use (gofrs or google) | "gofrs"                  |
| concurrency   | How many tables to fetch in parallel  | 10                       |
| only          | Only generate these                   |                          |
| except        | Skip generation for these             |                          |

Example of Only/Except:

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
