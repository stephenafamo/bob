# Update

## Simple

SQL:

```sql
UPDATE films SET [kind] = @p1 WHERE ([kind] = @p2)
```

Args:

* `"Dramatic"`
* `"Drama"`

Code:

```go
mssql.Update(
  um.Table("films"),
  um.SetCol("kind").ToArg("Dramatic"),
  um.Where(mssql.Quote("kind").EQ(mssql.Arg("Drama"))),
)
```

## With From

SQL:

```sql
UPDATE employees SET [sales_count] = sales_count + 1 FROM accounts
WHERE ([accounts].[name] = @p1)
AND ([employees].[id] = [accounts].[sales_person])
```

Args:

* `"Acme Corporation"`

Code:

```go
mssql.Update(
  um.Table("employees"),
  um.SetCol("sales_count").To("sales_count + 1"),
  um.From("accounts"),
  um.Where(mssql.Quote("accounts", "name").EQ(mssql.Arg("Acme Corporation"))),
  um.Where(mssql.Quote("employees", "id").EQ(mssql.Quote("accounts", "sales_person"))),
)
```

## Update with OUTPUT clause

SQL:

```sql
UPDATE inventory SET [quantity] = @p1 OUTPUT deleted.product_id, deleted.quantity WHERE ([quantity] < @p2)
```

Args:

* `0`
* `5`

Code:

```go
mssql.Update(
  um.Table("inventory"),
  um.SetCol("quantity").ToArg(0),
  um.Output("deleted.product_id", "deleted.quantity"),
  um.Where(mssql.Quote("quantity").LT(mssql.Arg(5))),
)
```

## Update with TOP to limit rows updated

SQL:

```sql
UPDATE TOP (10) employees SET [bonus] = @p1
```

Args:

* `500`

Code:

```go
mssql.Update(
  um.Table("employees"),
  um.Top(10),
  um.SetCol("bonus").ToArg(500),
)
```

## With Sub-Select

SQL:

```sql
UPDATE employees SET [sales_count] = sales_count + 1 WHERE ([id] = ((SELECT sales_person FROM accounts WHERE ([name] = @p1))))
```

Args:

* `"Acme Corporation"`

Code:

```go
mssql.Update(
  um.Table("employees"),
  um.SetCol("sales_count").To("sales_count + 1"),
  um.Where(mssql.Quote("id").EQ(mssql.Group(mssql.Select(
    sm.Columns("sales_person"),
    sm.From("accounts"),
    sm.Where(mssql.Quote("name").EQ(mssql.Arg("Acme Corporation"))),
  )))),
)
```

## With CTE And Output

SQL:

```sql
WITH [old_data] AS (SELECT id, name FROM users WHERE ([active] = @p1)) UPDATE users SET [active] = @p2 OUTPUT inserted.id, inserted.active WHERE ([id] IN (SELECT id FROM old_data))
```

Args:

* `false`
* `true`

Code:

```go
mssql.Update(
  um.With("old_data").As(mssql.Select(
    sm.Columns("id", "name"),
    sm.From("users"),
    sm.Where(mssql.Quote("active").EQ(mssql.Arg(false))),
  )),
  um.Table("users"),
  um.SetCol("active").ToArg(true),
  um.Output("inserted.id", "inserted.active"),
  um.Where(mssql.Quote("id").In(mssql.Raw("SELECT id FROM old_data"))),
)
```

## With Join

SQL:

```sql
UPDATE t1 SET [col1] = [t2].[col2] FROM t2 INNER JOIN t3 ON ([t2].[id] = [t3].[id]) WHERE ([t1].[id] = [t2].[ref_id])
```

Code:

```go
mssql.Update(
  um.Table("t1"),
  um.SetCol("col1").To(mssql.Quote("t2", "col2")),
  um.From("t2"),
  um.InnerJoin("t3").OnEQ(mssql.Quote("t2", "id"), mssql.Quote("t3", "id")),
  um.Where(mssql.Quote("t1", "id").EQ(mssql.Quote("t2", "ref_id"))),
)
```

## Update with LEFT JOIN to set defaults for unmatched rows

SQL:

```sql
UPDATE t1 SET [status] = @p1
FROM t1 AS [src] LEFT JOIN t2 AS [ref] ON ([src].[ref_id] = [ref].[id])
WHERE ([t1].[id] = [src].[id])
AND ([ref].[id] IS NULL)
```

Args:

* `"orphan"`

Code:

```go
mssql.Update(
  um.Table("t1"),
  um.SetCol("status").ToArg("orphan"),
  um.From("t1").As("src"),
  um.LeftJoin("t2").As("ref").OnEQ(mssql.Quote("src", "ref_id"), mssql.Quote("ref", "id")),
  um.Where(mssql.Quote("t1", "id").EQ(mssql.Quote("src", "id"))),
  um.Where(mssql.Quote("ref", "id").IsNull()),
)
```

## OUTPUT must appear between SET and FROM

SQL:

```sql
UPDATE t1 SET [col1] = [t2].[col2]
OUTPUT inserted.col1, deleted.col1
FROM t2
WHERE ([t1].[id] = [t2].[ref_id])
```

Code:

```go
mssql.Update(
  um.Table("t1"),
  um.SetCol("col1").To(mssql.Quote("t2", "col2")),
  um.Output("inserted.col1", "deleted.col1"),
  um.From("t2"),
  um.Where(mssql.Quote("t1", "id").EQ(mssql.Quote("t2", "ref_id"))),
)
```
