---
sidebar_position: 0
description: Generating an ORM and Factories tailored to your schema
---

import DocCardList from '@theme/DocCardList';
import {useCurrentSidebarCategory} from '@docusaurus/theme-common';

# Introduction

Bob is a "database-first" ORM. That means you must first create your database schema. Please use something like [sql-migrate](https://github.com/rubenv/sql-migrate) or some other migration tool to manage this part of the database's life-cycle.

## Available Drivers

| Sources    | Driver           |
| ---------- | ---------------- |
| PostgreSQL | [LINK](./psql)   |
| MySQL      | [LINK](./mysql)  |
| SQLite     | [LINK](./sqlite) |
| SQL files  | [LINK](./sql)    |

## Features

- Full model generation
- Generates **factories** for easy testing
- Generates code for hand-written **SQL** queries (similar to [sqlc](https://sqlc.dev)).
- Extremely fast code generation
- High performance through generation & intelligent caching
- Uses bob.Executor (simple interface, sql.DB, sql.Tx, sqlx.DB etc. compatible)
- Uses context.Context
- Easy workflow (models can always be regenerated, full auto-complete)
- Strongly typed querying (usually no converting or binding to pointers)
- Hooks (Before/After Select/Insert/Update/Delete/Upsert)
- Table and column filtering for generation
- Custom struct tags
- Raw SQL fallback
- Cross-schema support
- 1d arrays, json, hstore & more
- Enum types
- Support for database views
- Supports generated/computed columns
- Multi-column foreign key support
- Relationships/Associations
  - Eager loading (recursive)
  - Automatically detects relationships based on foreign keys
  - Can load related models both by a left-join and a 2nd query
  - Supports user-configured relationships
  - Can configure relationships based on static column values. For example, (`WHERE object_type = 'car' AND object_id = cars.id`)
  - Support for `has-one-through` and `has-many-through`.

## Generating code

The code generator is run through the dialect specific command in the [gen](https://pkg.go.dev/github.com/stephenafamo/bob/gen#section-directories) package.
All code generator commands require connection information for your database (to parse the database structure).
This can be provided either through an (dialect specific) environment variable or through a configuration file.
See [configuration](./configuration) for details and other customizable options.

### Example Usage

**Postgres**

```sh
# With env variable
PSQL_DSN=postgres://user:pass@host:port/dbname go run github.com/stephenafamo/bob/gen/bobgen-psql@latest

# With configuration file
go run github.com/stephenafamo/bob/gen/bobgen-psql@latest -c ./config/bobgen.yaml
```
**MySQL**

```sh
# With env variable
MYSQL_DSN=user:pass@tcp(host:port)/dbname go run github.com/stephenafamo/bob/gen/bobgen-mysql@latest

# With configuration file
go run github.com/stephenafamo/bob/gen/bobgen-mysql@latest -c ./config/bobgen.yaml
```

**SQLite**
```sh
# With env variable
SQLITE_DSN=test.db go run github.com/stephenafamo/bob/gen/bobgen-sqlite@latest

# With configuration file
go run github.com/stephenafamo/bob/gen/bobgen-sqlite@latest -c ./config/bobgen.yaml
```

**SQL Files**
```sh
# With env variable
SQL_DIALECT=psql go run github.com/stephenafamo/bob/gen/bobgen-sql@latest

# With configuration file
go run github.com/stephenafamo/bob/gen/bobgen-sql@latest -c ./config/bobgen.yaml
```
Refer to [driver / dialect specific documentation](#available-drivers) for more details

<DocCardList items={useCurrentSidebarCategory().items.filter(i => i.label != 'Introduction')} />
