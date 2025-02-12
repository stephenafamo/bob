---

sidebar_position: 0
description: Supported features

---

# How to Use

Import the `mysql` package and the query mod packages for the different query types

```go
import (
    "github.com/twitter-payments/bob/dialect/mysql"
    "github.com/twitter-payments/bob/dialect/mysql/sm"
    "github.com/twitter-payments/bob/dialect/mysql/im"
    "github.com/twitter-payments/bob/dialect/mysql/um"
    "github.com/twitter-payments/bob/dialect/mysql/dm"
)

func main() {
    mysql.Select(
        sm.From("users"),
    )

    mysql.Insert(
        im.Into("users"),
    )

    mysql.Update(
        um.Table("users"),
    )

    mysql.Delete(
        dm.From("users"),
    )

    mysql.Raw()
}
```

## Dialect Support

### Query types

View the reference for the query mod packages:

* [X] Raw
* [X] Select: [Query Mods](https://pkg.go.dev/github.com/stephenafamo/bob/dialect/mysql/sm)
* [X] Insert: [Query Mods](https://pkg.go.dev/github.com/stephenafamo/bob/dialect/mysql/im)
* [X] Update: [Query Mods](https://pkg.go.dev/github.com/stephenafamo/bob/dialect/mysql/um)
* [X] Delete: [Query Mods](https://pkg.go.dev/github.com/stephenafamo/bob/dialect/mysql/dm)

### Starters

These are MySQL specific starters, **in addition** to the [common starters](../starters)

> Empty

### Operators

These are MySQL specific operators, **in addition** to the [common operators](../operators)

> Empty
