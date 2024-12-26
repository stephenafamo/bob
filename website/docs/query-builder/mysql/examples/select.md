# Select

## Simple Select

SQL:

```sql
SELECT id, name FROM users WHERE (`id` IN (?, ?, ?))
```

Args:

* `100`
* `200`
* `300`

Code:

```go
mysql.Select(
  sm.Columns("id", "name"),
  sm.From("users"),
  sm.Where(mysql.Quote("id").In(mysql.Arg(100, 200, 300))),
)
```

## Case With Else

SQL:

```sql
SELECT id, name, (CASE WHEN (`id` = '1') THEN 'A' ELSE 'B' END) AS `C` FROM users
```

Code:

```go
mysql.Select(
  sm.Columns(
    "id",
    "name",
    mysql.Case().
      When(mysql.Quote("id").EQ(mysql.S("1")), mysql.S("A")).
      Else(mysql.S("B")).
      As("C"),
  ),
  sm.From("users"),
)
```

## Case Without Else

SQL:

```sql
SELECT id, name, (CASE WHEN (`id` = '1') THEN 'A' END) AS `C` FROM users
```

Code:

```go
mysql.Select(
  sm.Columns(
    "id",
    "name",
    mysql.Case().
      When(mysql.Quote("id").EQ(mysql.S("1")), mysql.S("A")).
      End().
      As("C"),
  ),
  sm.From("users"),
)
```

## Select Distinct

SQL:

```sql
SELECT DISTINCT id, name FROM users WHERE (`id` IN (?, ?, ?))
```

Args:

* `100`
* `200`
* `300`

Code:

```go
mysql.Select(
  sm.Columns("id", "name"),
  sm.Distinct(),
  sm.From("users"),
  sm.Where(mysql.Quote("id").In(mysql.Arg(100, 200, 300))),
)
```

## With Sub-Select

SQL:

```sql
SELECT status, avg(difference)
FROM (
  SELECT
    status,
    (LEAD(created_date, 1, NOW())
    OVER (PARTITION BY presale_id ORDER BY created_date)
     - `created_date`) AS `difference`
  FROM presales_presalestatus
) AS `differnce_by_status`
WHERE (`status` IN ('A', 'B', 'C'))
GROUP BY status
```

Code:

```go
mysql.Select(
  sm.Columns("status", mysql.F("avg", "difference")),
  sm.From(mysql.Select(
    sm.Columns(
      "status",
      mysql.F("LEAD", "created_date", 1, mysql.F("NOW"))(
        fm.Over(
          wm.PartitionBy("presale_id"),
          wm.OrderBy("created_date"),
        ),
      ).Minus(mysql.Quote("created_date")).As("difference")),
    sm.From("presales_presalestatus")),
  ).As("differnce_by_status"),
  sm.Where(mysql.Quote("status").In(mysql.S("A"), mysql.S("B"), mysql.S("C"))),
  sm.GroupBy("status"),
)
```

## Select With Grouped IN

SQL:

```sql
SELECT id, name FROM users WHERE ((`id`, `employee_id`) IN ((?, ?), (?, ?)))
```

Args:

* `100`
* `200`
* `300`
* `400`

Code:

```go
mysql.Select(
  sm.Columns("id", "name"),
  sm.From("users"),
  sm.Where(mysql.Group(mysql.Quote("id"), mysql.Quote("employee_id")).In(mysql.ArgGroup(100, 200), mysql.ArgGroup(300, 400))),
)
```

## Select With Order By And Collate

SQL:

```sql
SELECT id, name FROM users ORDER BY name COLLATE `utf8mb4_bg_0900_as_cs` ASC
```

Code:

```go
mysql.Select(
  sm.Columns("id", "name"),
  sm.From("users"),
  sm.OrderBy("name").Collate("utf8mb4_bg_0900_as_cs").Asc(),
)
```
