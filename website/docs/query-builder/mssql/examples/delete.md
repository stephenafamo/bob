# Delete

## Simple

SQL:

```sql
DELETE FROM films WHERE ([kind] = @p1)
```

Args:

* `"Drama"`

Code:

```go
mssql.Delete(
  dm.From("films"),
  dm.Where(mssql.Quote("kind").EQ(mssql.Arg("Drama"))),
)
```

## Delete with OUTPUT clause to capture deleted rows

SQL:

```sql
DELETE FROM employees OUTPUT deleted.id, deleted.name WHERE ([terminated] = @p1)
```

Args:

* `true`

Code:

```go
mssql.Delete(
  dm.From("employees"),
  dm.Output("deleted.id", "deleted.name"),
  dm.Where(mssql.Quote("terminated").EQ(mssql.Arg(true))),
)
```

## Delete with TOP to limit number of rows deleted

SQL:

```sql
DELETE TOP (1000) FROM logs WHERE ([created_at] < @p1)
```

Args:

* `"2020-01-01"`

Code:

```go
mssql.Delete(
  dm.From("logs"),
  dm.Top(1000),
  dm.Where(mssql.Quote("created_at").LT(mssql.Arg("2020-01-01"))),
)
```

## Delete with FROM/JOIN for multi-table delete

SQL:

```sql
DELETE FROM order_items FROM orders
WHERE ([order_items].[order_id] = [orders].[id])
AND ([orders].[status] = @p1)
```

Args:

* `"cancelled"`

Code:

```go
mssql.Delete(
  dm.From("order_items"),
  dm.Using("orders"),
  dm.Where(mssql.Quote("order_items", "order_id").EQ(mssql.Quote("orders", "id"))),
  dm.Where(mssql.Quote("orders", "status").EQ(mssql.Arg("cancelled"))),
)
```

## Delete with OUTPUT positioned between target table and second FROM

SQL:

```sql
DELETE FROM order_items
OUTPUT deleted.id, deleted.order_id
FROM orders
WHERE ([order_items].[order_id] = [orders].[id])
AND ([orders].[status] = @p1)
```

Args:

* `"cancelled"`

Code:

```go
mssql.Delete(
  dm.From("order_items"),
  dm.Output("deleted.id", "deleted.order_id"),
  dm.Using("orders"),
  dm.Where(mssql.Quote("order_items", "order_id").EQ(mssql.Quote("orders", "id"))),
  dm.Where(mssql.Quote("orders", "status").EQ(mssql.Arg("cancelled"))),
)
```

## Delete with explicit INNER JOIN via second FROM clause

SQL:

```sql
DELETE FROM order_items
FROM orders INNER JOIN products ON ([orders].[product_id] = [products].[id])
WHERE ([order_items].[order_id] = [orders].[id])
AND ([products].[discontinued] = @p1)
```

Args:

* `true`

Code:

```go
mssql.Delete(
  dm.From("order_items"),
  dm.Using("orders"),
  dm.InnerJoin("products").OnEQ(mssql.Quote("orders", "product_id"), mssql.Quote("products", "id")),
  dm.Where(mssql.Quote("order_items", "order_id").EQ(mssql.Quote("orders", "id"))),
  dm.Where(mssql.Quote("products", "discontinued").EQ(mssql.Arg(true))),
)
```

## Delete with LEFT JOIN to find unmatched rows

SQL:

```sql
DELETE FROM users
FROM users AS [u] LEFT JOIN orders AS [o] ON ([u].[id] = [o].[user_id])
WHERE ([users].[id] = [u].[id])
AND ([o].[id] IS NULL)
```

Code:

```go
mssql.Delete(
  dm.From("users"),
  dm.Using("users").As("u"),
  dm.LeftJoin("orders").As("o").OnEQ(mssql.Quote("u", "id"), mssql.Quote("o", "user_id")),
  dm.Where(mssql.Quote("users", "id").EQ(mssql.Quote("u", "id"))),
  dm.Where(mssql.Quote("o", "id").IsNull()),
)
```

## With CTE

SQL:

```sql
WITH [old_users] AS (SELECT id FROM users WHERE ([active] = @p1)) DELETE FROM users WHERE ([id] IN (SELECT id FROM old_users))
```

Args:

* `false`

Code:

```go
mssql.Delete(
  dm.With("old_users").As(mssql.Select(
    sm.Columns("id"),
    sm.From("users"),
    sm.Where(mssql.Quote("active").EQ(mssql.Arg(false))),
  )),
  dm.From("users"),
  dm.Where(mssql.Quote("id").In(mssql.Raw("SELECT id FROM old_users"))),
)
```
