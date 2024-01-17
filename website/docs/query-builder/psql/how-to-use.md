---

sidebar_position: 0
description: Supported features

---

# How to Use

Import the `psql` package and the query mod packages for the different query types

```go
import (
    "github.com/stephenafamo/bob/dialect/psql"
    "github.com/stephenafamo/bob/dialect/psql/sm"
    "github.com/stephenafamo/bob/dialect/psql/im"
    "github.com/stephenafamo/bob/dialect/psql/um"
    "github.com/stephenafamo/bob/dialect/psql/dm"
)

func main() {
    psql.Select(
        sm.From("users"),
    )

    psql.Insert(
        im.Into("users"),
    )

    psql.Update(
        um.Table("users"),
    )

    psql.Delete(
        dm.From("users"),
    )

    psql.Raw()
}
```

## Dialect Support

### Query types

View the reference for the query mod packages:

* [X] Raw
* [X] Select: [Query Mods](https://pkg.go.dev/github.com/stephenafamo/bob/dialect/psql/sm)
* [X] Insert: [Query Mods](https://pkg.go.dev/github.com/stephenafamo/bob/dialect/psql/im)
* [X] Update: [Query Mods](https://pkg.go.dev/github.com/stephenafamo/bob/dialect/psql/um)
* [X] Delete: [Query Mods](https://pkg.go.dev/github.com/stephenafamo/bob/dialect/psql/dm)

### Starters

These are Postgres specific starters, **in addition** to the [common starters](../starters)

* `CONCAT(...any)`: Joins multiple expressions with "||"

    ```go
    // SQL: a || b || c
    psql.Concat("a", "b", "c")
    ```

### Operators

These are Postgres specific operators, **in addition** to the [common operators](../operators)

* `BetweenSymmetric(y, z any)`: X BETWEEN SYMMETRIC Y AND Z
* `NotBetweenSymmetric(y, z any)`: X NOT BETWEEN SYMMETRIC Y AND Z
