---
sidebar_position: 12
title: MySQL Driver
description: ORM Generation for MySQL
---

# Bob Gen for MySQL

Generates an ORM based on a MySQL database schema

## Usage

```sh
# With env variable
MYSQL_DSN=user:pass@tcp(host:port)/dbname go run github.com/stephenafamo/bob/gen/bobgen-mysql@latest

# With configuration file
go run github.com/stephenafamo/bob/gen/bobgen-mysql@latest -c ./config/bobgen.yaml
```

### Driver Configuration

#### [Link to general configuration and usage](./configuration)

The configuration for the MySQL driver must all be prefixed by the driver name. You must use a configuration file or environment variables for configuring the database driver.

In the configuration file for MySQL for example you would do:

```yaml
mysql:
  dsn: "user:pass@tcp(host:port)/dbname"
```

When you use an environment variable it must also be prefixed by the driver name:

```sh
MYSQL_DSN="user:pass@tcp(host:port)/dbname"
```

The values that exist for the drivers:

| Name        | Description                          | Default  |
| ----------- | ------------------------------------ | -------- |
| dsn         | URL to connect to                    |          |
| output      | Folder for generated files           | "models" |
| pkgname     | Package name for generated code      | "models" |
| concurrency | How many tables to fetch in parallel | 10       |
| only        | Only generate these                  |          |
| except      | Skip generation for these            |          |

Example of Only/Except:

```yaml
mysql:
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
