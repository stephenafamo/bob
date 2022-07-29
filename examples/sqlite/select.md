## Simple Select

SQL:

```sql
SELECT id, name FROM users WHERE (id IN (?1, ?2, ?3))
```

Args:

* `100`
* `200`
* `300`

Code:

```go
sqlite.Select(
  qm.Columns("id", "name"),
  qm.From("users"),
  qm.Where(sqlite.X("id").In(sqlite.Arg(100, 200, 300))),
)
```

## Select Distinct

SQL:

```sql
SELECT DISTINCT id, name FROM users WHERE (id IN (?1, ?2, ?3))
```

Args:

* `100`
* `200`
* `300`

Code:

```go
sqlite.Select(
  qm.Columns("id", "name"),
  qm.Distinct(),
  qm.From("users"),
  qm.Where(sqlite.X("id").In(sqlite.Arg(100, 200, 300))),
)
```

## With Rows From

SQL:

```sql
SELECT * FROM generate_series(1, 3) AS "x" ("p", "q", "s") ORDER BY p
```

Code:

```go
sqlite.Select(
  qm.From(
    sqlite.F("generate_series", 1, 3),
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
     - created_date) AS "difference"
  FROM presales_presalestatus
) AS "differnce_by_status"
WHERE (status IN ('A', 'B', 'C'))
GROUP BY status
```

Code:

```go
sqlite.Select(
  qm.Columns("status", sqlite.F("avg", "difference")),
  qm.From(sqlite.Select(
    qm.Columns(
      "status",
      sqlite.F("LEAD", "created_date", 1, sqlite.F("NOW")).
        Over("").
        PartitionBy("presale_id").
        OrderBy("created_date").
        Minus("created_date").
        As("difference")),
    qm.From("presales_presalestatus")),
    qm.As("differnce_by_status"),
  ),
  qm.Where(sqlite.X("status").In(sqlite.S("A"), sqlite.S("B"), sqlite.S("C"))),
  qm.GroupBy("status"),
)
```
