---

sidebar_position: 0
description: Overview of the Bob Query Builder

---

import DocCardList from '@theme/DocCardList';
import {useCurrentSidebarCategory} from '@docusaurus/theme-common';

# Introduction

Bob helps build SQL queries. It does not try to abstract away SQL, or to hide implementation, instead **Bob** adds typed handrails to the query building
for a better development experience.

## Principles

The query builder is designed with the following principles

1. Custom Crafting
2. Progressive Enhancement

Read more about [the principles](./principles).

## Features

* Faster than comparable packages. [See Benchmarks](https://github.com/stephenafamo/go-sql-builder-benchmarks).
* Build any query. Supports the specification as closely as possible.

## Dialect Support

| Dialect       | Raw | Select | Insert | Update | Delete |
|---------------|-----|--------|--------|--------|--------|
| Postgres      | ✅   | ✅      | ✅      | ✅      | ✅      |
| MySQL/MariaDB | ✅   | ✅      | ✅      | ✅      | ✅      |
| SQLite        | ✅   | ✅      | ✅      | ✅      | ✅      |

## Examples

Want to jump straight into examples?

* [Postgres](psql/examples)
* [MySQL](mysql/examples)
* [SQLite](sqlite/examples)

<DocCardList items={useCurrentSidebarCategory().items.filter(i => i.label != 'Introduction')} />
