---

sidebar_position: 0
description: Learn about Bob's View and Table models

---

import DocCardList from '@theme/DocCardList';
import {useCurrentSidebarCategory} from '@docusaurus/theme-common';

# Introduction

Each implemented dialect provides two model types. `View` and `Table`.

While views only provide methods to query the underlying database, tables also add methods to insert, update and delete from the database.

| Dialect       | View | Table |
|---------------|------|-------|
| Postgres      | ✅    | ✅     |
| MySQL/MariaDB | ✅    | ✅     |
| SQLite        | ✅    | ✅     |

<DocCardList items={useCurrentSidebarCategory().items.filter(i => i.label != 'Introduction')} />
