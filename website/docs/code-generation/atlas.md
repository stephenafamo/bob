---
sidebar_position: 14
title: Atlas Driver
description: ORM Generation for Atlas
---

# Bob Gen for Atlas

Generates an ORM based on Atlas HCL schema files

## Usage

```sh
# With env variable
ATLAS_DIALECT=psql go run github.com/stephenafamo/bob/gen/bobgen-atlas@latest

# With configuration file
go run github.com/stephenafamo/bob/gen/bobgen-atlas@latest -c ./config/bobgen.yaml
```

### Driver Configuration

#### [Link to general configuration and usage](./configuration)

The configuration for the atlas driver must be prefixed by the driver name. You must use a configuration file or environment variables for configuring the database driver.

In the configuration file for atlas for example you would do:

```yaml
atlas:
  dialect: psql
```

When you use an environment variable it must also be prefixed by the driver name:

```sh
ATLAS_DIALECT=psql
```

The values that exist for the drivers:

| Name          | Description                               | Default            |
| ------------- | ----------------------------------------- | ------------------ |
| dialect       | Database dialect to use (REQUIRED)        |                    |
| dir           | Path to directory containing schema files | .                  |
| shared_schema | Schema to not include prefix in model     | first schema found |
| output        | Folder for generated files                | "models"           |
| pkgname       | Package name for generated code           | "models"           |
| only          | Only generate these                       |                    |
| except        | Skip generation for these                 |                    |

Example of Only/Except:

```yaml
atlas:
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
