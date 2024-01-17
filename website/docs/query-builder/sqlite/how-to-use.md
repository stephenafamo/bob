---

sidebar_position: 0
description: Supported features

---

# How to Use

Import the `sqlite` package and the query mod packages for the different query types

```go
import (
    "github.com/stephenafamo/bob/dialect/sqlite"
    "github.com/stephenafamo/bob/dialect/sqlite/sm"
    "github.com/stephenafamo/bob/dialect/sqlite/im"
    "github.com/stephenafamo/bob/dialect/sqlite/um"
    "github.com/stephenafamo/bob/dialect/sqlite/dm"
)

func main() {
    sqlite.Select(
        sm.From("users"),
    )

    sqlite.Insert(
        im.Into("users"),
    )

    sqlite.Update(
        um.Table("users"),
    )

    sqlite.Delete(
        dm.From("users"),
    )

    sqlite.Raw()
}
```

## Dialect Support

### Query types

View the reference for the query mod packages:

* [X] Raw
* [X] Select: [Query Mods](https://pkg.go.dev/github.com/stephenafamo/bob/dialect/sqlite/sm)
* [X] Insert: [Query Mods](https://pkg.go.dev/github.com/stephenafamo/bob/dialect/sqlite/im)
* [X] Update: [Query Mods](https://pkg.go.dev/github.com/stephenafamo/bob/dialect/sqlite/um)
* [X] Delete: [Query Mods](https://pkg.go.dev/github.com/stephenafamo/bob/dialect/sqlite/dm)

### Starters

These are SQLite specific starters, **in addition** to the [common starters](../starters)

> Empty

### Operators

These are SQLite specific operators, **in addition** to the [common operators](../operators)

> Empty
