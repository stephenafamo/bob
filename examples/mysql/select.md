## Simple Select

SQL:

```sql
SELECT id, name FROM users WHERE (id IN (?, ?, ?))
```

Args:

* `100`
* `200`
* `300`

Code:

```go
mysql.Select(
  qm.Columns("id", "name"),
  qm.From("users"),
  qm.Where(mysql.X("id").In(mysql.Arg(100, 200, 300))),
)
```

## Select Distinct

SQL:

```sql
SELECT DISTINCT id, name FROM users WHERE (id IN (?, ?, ?))
```

Args:

* `100`
* `200`
* `300`

Code:

```go
mysql.Select(
  qm.Columns("id", "name"),
  qm.Distinct(),
  qm.From("users"),
  qm.Where(mysql.X("id").In(mysql.Arg(100, 200, 300))),
)
```

## With Rows From

SQL:

```sql
SELECT * FROM generate_series(1, 3) AS `x` (`p`, `q`, `s`) ORDER BY p
```

Code:

```go
mysql.Select(
  qm.From(
    mysql.F("generate_series", 1, 3),
    qm.As("x", "p", "q", "s"),
  ),
  qm.OrderBy("p"),
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
     - created_date) AS `difference`
  FROM presales_presalestatus
) AS `differnce_by_status`
WHERE (status IN ('A', 'B', 'C'))
GROUP BY status
```

Code:

```go
mysql.Select(
  qm.Columns("status", mysql.F("avg", "difference")),
  qm.From(mysql.Select(
    qm.Columns(
      "status",
      mysql.F("LEAD", "created_date", 1, mysql.F("NOW")).
        Over("").
        PartitionBy("presale_id").
        OrderBy("created_date").
        Minus("created_date").
        As("difference")),
    qm.From("presales_presalestatus")),
    qm.As("differnce_by_status"),
  ),
  qm.Where(mysql.X("status").In(mysql.S("A"), mysql.S("B"), mysql.S("C"))),
  qm.GroupBy("status"),
)
```
