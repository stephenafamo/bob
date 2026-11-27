---

sidebar_position: 0
description: Supported features

---

# How to Use

Import the `mssql` package and the query mod packages for the different query types

```go
import (
    "github.com/stephenafamo/bob/dialect/mssql"
    "github.com/stephenafamo/bob/dialect/mssql/sm"
    "github.com/stephenafamo/bob/dialect/mssql/im"
    "github.com/stephenafamo/bob/dialect/mssql/um"
    "github.com/stephenafamo/bob/dialect/mssql/dm"
    "github.com/stephenafamo/bob/dialect/mssql/mm"
)

func main() {
    mssql.Select(
        sm.From("users"),
    )

    mssql.Insert(
        im.Into("users"),
    )

    mssql.Update(
        um.Table("users"),
    )

    mssql.Delete(
        dm.From("users"),
    )

    mssql.Merge(
        mm.Into("users"),
    )

    mssql.RawQuery("SELECT 1")
}
```

Arguments are rendered as `@p1`, `@p2`, ... and identifiers are quoted with square brackets (`[column]`). Named arguments (`sql.NamedArg`) are rendered as `@name`.

## Dialect Support

### Query types

View the reference for the query mod packages:

* [X] Raw
* [X] Select: [Query Mods](https://pkg.go.dev/github.com/stephenafamo/bob/dialect/mssql/sm)
* [X] Insert: [Query Mods](https://pkg.go.dev/github.com/stephenafamo/bob/dialect/mssql/im)
* [X] Update: [Query Mods](https://pkg.go.dev/github.com/stephenafamo/bob/dialect/mssql/um)
* [X] Delete: [Query Mods](https://pkg.go.dev/github.com/stephenafamo/bob/dialect/mssql/dm)
* [X] Values: [Query Mods](https://pkg.go.dev/github.com/stephenafamo/bob/dialect/mssql/vm)
* [X] Merge: [Query Mods](https://pkg.go.dev/github.com/stephenafamo/bob/dialect/mssql/mm)

### Starters

These are SQL Server specific starters, **in addition** to the [common starters](../starters)

* `CONCAT(...any)`: Joins multiple expressions with "+"

    ```go
    // SQL: a + b + c
    mssql.Concat("a", "b", "c")
    ```

### Operators

These are SQL Server specific operators, **in addition** to the [common operators](../operators)

> Empty

### Query specifics

* `TOP`: `sm.Top`, `sm.TopPercent` and `sm.TopWithTies` for `SELECT`, and `im.Top`, `um.Top` and `dm.Top` for `INSERT`, `UPDATE` and `DELETE`

    ```go
    // SQL: SELECT TOP (10) id FROM users
    mssql.Select(sm.Columns("id"), sm.Top(10), sm.From("users"))
    ```

* `OFFSET ... FETCH`: SQL Server has no `LIMIT`. Use `sm.Offset` and `sm.Fetch` together with an `ORDER BY`

    ```go
    // SQL: SELECT id FROM users ORDER BY id OFFSET @p1 ROWS FETCH NEXT @p2 ROWS ONLY
    mssql.Select(
        sm.Columns("id"),
        sm.From("users"),
        sm.OrderBy("id"),
        sm.Offset(mssql.Arg(10)),
        sm.Fetch(mssql.Arg(5)),
    )
    ```

* `OUTPUT`: `im.Output`, `um.Output`, `dm.Output` and `mm.Output` replace `RETURNING`

    ```go
    // SQL: DELETE FROM users OUTPUT deleted.id WHERE ([id] = @p1)
    mssql.Delete(
        dm.From("users"),
        dm.Output("deleted.id"),
        dm.Where(mssql.Quote("id").EQ(mssql.Arg(1))),
    )
    ```
