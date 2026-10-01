# Select

## Simple Select with some conditions

SQL:

```sql
SELECT id, name FROM users WHERE ([id] IN (@p1, @p2, @p3))
```

Args:

* `100`
* `200`
* `300`

Code:

```go
mssql.Select(
  sm.Columns("id", "name"),
  sm.From("users"),
  sm.Where(mssql.Quote("id").In(mssql.Arg(100, 200, 300))),
)
```

## Select Distinct

SQL:

```sql
SELECT DISTINCT id, name FROM users WHERE ([id] IN (@p1, @p2, @p3))
```

Args:

* `100`
* `200`
* `300`

Code:

```go
mssql.Select(
  sm.Columns("id", "name"),
  sm.Distinct(),
  sm.From("users"),
  sm.Where(mssql.Quote("id").In(mssql.Arg(100, 200, 300))),
)
```

## Select with TOP clause

SQL:

```sql
SELECT TOP (10) id, name FROM users
```

Code:

```go
mssql.Select(
  sm.Columns("id", "name"),
  sm.Top(10),
  sm.From("users"),
)
```

## DISTINCT appears before TOP in SELECT clause

SQL:

```sql
SELECT DISTINCT TOP (5) id, name FROM users
```

Code:

```go
mssql.Select(
  sm.Columns("id", "name"),
  sm.Distinct(),
  sm.Top(5),
  sm.From("users"),
)
```

## Select with TOP PERCENT clause

SQL:

```sql
SELECT TOP (50) PERCENT id, name FROM users
```

Code:

```go
mssql.Select(
  sm.Columns("id", "name"),
  sm.TopPercent(50),
  sm.From("users"),
)
```

## Select with TOP WITH TIES (requires ORDER BY)

SQL:

```sql
SELECT TOP (5) WITH TIES id, salary FROM employees ORDER BY salary DESC
```

Code:

```go
mssql.Select(
  sm.Columns("id", "salary"),
  sm.TopWithTies(5),
  sm.From("employees"),
  sm.OrderBy("salary").Desc(),
)
```

## Select with OFFSET/FETCH NEXT pagination

SQL:

```sql
SELECT id, name FROM users ORDER BY id OFFSET @p1 ROWS FETCH NEXT @p2 ROWS ONLY
```

Args:

* `10`
* `5`

Code:

```go
mssql.Select(
  sm.Columns("id", "name"),
  sm.From("users"),
  sm.OrderBy("id"),
  sm.Offset(mssql.Arg(10)),
  sm.Fetch(mssql.Arg(5)),
)
```

## TOP and OFFSET/FETCH can coexist — TOP limits before ORDER BY, OFFSET/FETCH paginate after

SQL:

```sql
SELECT TOP (100) id, name FROM users ORDER BY id OFFSET @p1 ROWS FETCH NEXT @p2 ROWS ONLY
```

Args:

* `20`
* `10`

Code:

```go
mssql.Select(
  sm.Columns("id", "name"),
  sm.Top(100),
  sm.From("users"),
  sm.OrderBy("id"),
  sm.Offset(mssql.Arg(20)),
  sm.Fetch(mssql.Arg(10)),
)
```

## Case With Else

SQL:

```sql
SELECT id, name, (CASE WHEN ([id] = 'A') THEN 'Yes' ELSE 'No' END) AS [C] FROM users
```

Code:

```go
mssql.Select(
  sm.Columns(
    "id",
    "name",
    mssql.Case().
      When(mssql.Quote("id").EQ(mssql.S("A")), mssql.S("Yes")).
      Else(mssql.S("No")).
      As("C"),
  ),
  sm.From("users"),
)
```

## Join Using On

SQL:

```sql
SELECT id FROM test1 INNER JOIN test2 ON ([test1].[id] = [test2].[id])
```

Code:

```go
mssql.Select(
  sm.Columns("id"),
  sm.From("test1"),
  sm.InnerJoin("test2").OnEQ(mssql.Quote("test1", "id"), mssql.Quote("test2", "id")),
)
```

## Left Join

SQL:

```sql
SELECT u.id, o.total FROM users AS [u] LEFT JOIN orders AS [o] ON ([u].[id] = [o].[user_id]) WHERE ([o].[total] > @p1)
```

Args:

* `100`

Code:

```go
mssql.Select(
  sm.Columns("u.id", "o.total"),
  sm.From("users").As("u"),
  sm.LeftJoin("orders").As("o").OnEQ(mssql.Quote("u", "id"), mssql.Quote("o", "user_id")),
  sm.Where(mssql.Quote("o", "total").GT(mssql.Arg(100))),
)
```

## CTE With Column Aliases

SQL:

```sql
WITH [c]([id], [data]) AS (SELECT id FROM test1 INNER JOIN test2 ON ([test1].[id] = [test2].[id])) SELECT * FROM c
```

Code:

```go
mssql.Select(
  sm.With("c", "id", "data").As(mssql.Select(
    sm.Columns("id"),
    sm.From("test1"),
    sm.InnerJoin("test2").OnEQ(mssql.Quote("test1", "id"), mssql.Quote("test2", "id")),
  )),
  sm.From("c"),
)
```

## Union

SQL:

```sql
SELECT id, name FROM users WHERE ([status] = @p1) UNION (SELECT id, name FROM archived_users WHERE ([status] = @p2))
```

Args:

* `"active"`
* `"archived"`

Code:

```go
mssql.Select(
  sm.Columns("id", "name"),
  sm.From("users"),
  sm.Where(mssql.Quote("status").EQ(mssql.Arg("active"))),
  sm.Union(mssql.Select(
    sm.Columns("id", "name"),
    sm.From("archived_users"),
    sm.Where(mssql.Quote("status").EQ(mssql.Arg("archived"))),
  )),
)
```

## Union All

SQL:

```sql
SELECT 1 UNION ALL (SELECT 2)
```

Code:

```go
mssql.Select(
  sm.Columns(1),
  sm.UnionAll(mssql.Select(
    sm.Columns(2),
  )),
)
```

## Window Function Over Empty Frame

SQL:

```sql
SELECT row_number() OVER () FROM c
```

Code:

```go
mssql.Select(
  sm.Columns(
    mssql.F("row_number")(fm.Over()),
  ),
  sm.From("c"),
)
```

## Window Function With Partition

SQL:

```sql
SELECT avg(salary) OVER (PARTITION BY depname ORDER BY salary) FROM employees
```

Code:

```go
mssql.Select(
  sm.Columns(
    mssql.F("avg", "salary")(fm.Over(
      wm.PartitionBy("depname"),
      wm.OrderBy("salary"),
    )),
  ),
  sm.From("employees"),
)
```

## Select With Grouped IN

SQL:

```sql
SELECT id, name FROM users WHERE (([id], [employee_id]) IN ((@p1, @p2), (@p3, @p4)))
```

Args:

* `100`
* `200`
* `300`
* `400`

Code:

```go
mssql.Select(
  sm.Columns("id", "name"),
  sm.From("users"),
  sm.Where(
    mssql.Group(mssql.Quote("id"), mssql.Quote("employee_id")).
      In(mssql.ArgGroup(100, 200), mssql.ArgGroup(300, 400)),
  ),
)
```

## Subquery In From

SQL:

```sql
SELECT s.id FROM (SELECT id FROM users WHERE ([active] = @p1)) AS [s]
```

Args:

* `true`

Code:

```go
mssql.Select(
  sm.Columns("s.id"),
  sm.From(mssql.Select(
    sm.Columns("id"),
    sm.From("users"),
    sm.Where(mssql.Quote("active").EQ(mssql.Arg(true))),
  )).As("s"),
)
```

## MSSQL uses + for string concatenation

SQL:

```sql
SELECT ([first_name] + ' ' + [last_name]) AS [full_name] FROM users
```

Code:

```go
mssql.Select(
  sm.Columns(mssql.Concat(
    mssql.Quote("first_name"),
    mssql.S(" "),
    mssql.Quote("last_name"),
  ).As("full_name")),
  sm.From("users"),
)
```

## Having Clause

SQL:

```sql
SELECT department, COUNT(*) FROM employees GROUP BY department HAVING COUNT(*) > @p1
```

Args:

* `5`

Code:

```go
mssql.Select(
  sm.Columns("department", "COUNT(*)"),
  sm.From("employees"),
  sm.GroupBy("department"),
  sm.Having(mssql.Raw("COUNT(*) > ?", 5)),
)
```

## Cross Join

SQL:

```sql
SELECT * FROM t1 CROSS JOIN t2
```

Code:

```go
mssql.Select(
  sm.From("t1"),
  sm.CrossJoin("t2"),
)
```

## Intersect

SQL:

```sql
SELECT id FROM t1 INTERSECT (SELECT id FROM t2)
```

Code:

```go
mssql.Select(
  sm.Columns("id"),
  sm.From("t1"),
  sm.Intersect(mssql.Select(
    sm.Columns("id"),
    sm.From("t2"),
  )),
)
```

## Except

SQL:

```sql
SELECT id FROM t1 EXCEPT (SELECT id FROM t2)
```

Code:

```go
mssql.Select(
  sm.Columns("id"),
  sm.From("t1"),
  sm.Except(mssql.Select(
    sm.Columns("id"),
    sm.From("t2"),
  )),
)
```

## Not Expression

SQL:

```sql
SELECT NOT true
```

Code:

```go
mssql.Select(
  sm.Columns(mssql.Not(mssql.Raw("true"))),
)
```

## Or Expression

SQL:

```sql
SELECT * FROM users WHERE (a OR b)
```

Code:

```go
mssql.Select(
  sm.From("users"),
  sm.Where(mssql.Or(mssql.Raw("a"), mssql.Raw("b"))),
)
```

## And Expression

SQL:

```sql
SELECT * FROM users WHERE (a AND b)
```

Code:

```go
mssql.Select(
  sm.From("users"),
  sm.Where(mssql.And(mssql.Raw("a"), mssql.Raw("b"))),
)
```

## Cast Expression

SQL:

```sql
SELECT (CAST([price] AS DECIMAL(10,2))) FROM products
```

Code:

```go
mssql.Select(
  sm.Columns(mssql.Cast(mssql.Quote("price"), "DECIMAL(10,2)")),
  sm.From("products"),
)
```

## Exists Expression

SQL:

```sql
SELECT * FROM users WHERE EXISTS ((SELECT 1 FROM orders WHERE ([user_id] = @p1)))
```

Args:

* `42`

Code:

```go
mssql.Select(
  sm.From("users"),
  sm.Where(mssql.Exists(mssql.Select(
    sm.Columns(1),
    sm.From("orders"),
    sm.Where(mssql.Quote("user_id").EQ(mssql.Arg(42))),
  ))),
)
```

## Minus Expression

SQL:

```sql
SELECT - @p1
```

Args:

* `1`

Code:

```go
mssql.Select(
  sm.Columns(mssql.Minus(mssql.Arg(1))),
)
```

## Placeholder

SQL:

```sql
SELECT @p1, @p2, @p3
```

Args:

* `nil`
* `nil`
* `nil`

Code:

```go
mssql.Select(
  sm.Columns(mssql.Placeholder(3)),
)
```

## Raw Query

SQL:

```sql
SELECT * FROM users WHERE id = @p1
```

Args:

* `42`

Code:

```go
mssql.RawQuery("SELECT * FROM users WHERE id = ?", 42)
```
