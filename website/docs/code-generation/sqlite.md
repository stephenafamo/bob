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

| Name          | Description                               | Default             |
| ------------- | ----------------------------------------- | ------------------- |
| dsn           | Path to database                          |                     |
| attach        | Schemas to attach and the path the the db | map[string]string{} |
| shared_schema | Schema to not include prefix in model     | "main"              |
| output        | Folder for generated files                | "models"            |
| pkgname       | Package name for generated code           | "models"            |
| only          | Only generate these                       |                     |
| except        | Skip generation for these                 |                     |

Example of Only/Except:

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
