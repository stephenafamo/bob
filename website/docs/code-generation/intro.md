---
sidebar_position: 0
description: Generating an ORM and Fatory tailored to your schema
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

<DocCardList items={useCurrentSidebarCategory().items.filter(i => i.label != 'Introduction')} />
