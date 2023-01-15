---

sidebar_position: 0
description: Understand the `bob.Executor` interface and the provided types

---

import DocCardList from '@theme/DocCardList';
import {useCurrentSidebarCategory} from '@docusaurus/theme-common';

# Introduction

Bob's Executor is an interface that looks like this.

```go
type Executor interface {
	QueryContext(ctx context.Context, query string, args ...any) (scan.Rows, error)
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}
```

This executor is used to run queries

<DocCardList items={useCurrentSidebarCategory().items.filter(i => i.label != 'Introduction')} />
