# Select

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
  sm.Columns("id", "name"),
  sm.From("users"),
  sm.Where(mysql.X("id").In(mysql.Arg(100, 200, 300))),
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
  sm.Columns("id", "name"),
  sm.Distinct(),
  sm.From("users"),
  sm.Where(mysql.X("id").In(mysql.Arg(100, 200, 300))),
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
  sm.Columns("status", mysql.F("avg", "difference")),
  sm.From(mysql.Select(
    sm.Columns(
      "status",
      mysql.F("LEAD", "created_date", 1, mysql.F("NOW")).
        Over("").
        PartitionBy("presale_id").
        OrderBy("created_date").
        Minus("created_date").
        As("difference")),
    sm.From("presales_presalestatus")),
  ).As("differnce_by_status"),
  sm.Where(mysql.X("status").In(mysql.S("A"), mysql.S("B"), mysql.S("C"))),
  sm.GroupBy("status"),
)
```
