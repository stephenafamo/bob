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
  sm.Columns("id", "name"),
  sm.From("users"),
  sm.Where(sqlite.X("id").In(sqlite.Arg(100, 200, 300))),
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
  sm.Columns("id", "name"),
  sm.Distinct(),
  sm.From("users"),
  sm.Where(sqlite.X("id").In(sqlite.Arg(100, 200, 300))),
)
```

## From Function

SQL:

```sql
SELECT * FROM generate_series(1, 3) AS "x"
```

Code:

```go
sqlite.Select(
  sm.From(sqlite.F("generate_series", 1, 3)).As("x"),
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
  sm.Columns("status", sqlite.F("avg", "difference")),
  sm.From(sqlite.Select(
    sm.Columns(
      "status",
      sqlite.F("LEAD", "created_date", 1, sqlite.F("NOW")).
        Over("").
        PartitionBy("presale_id").
        OrderBy("created_date").
        Minus("created_date").
        As("difference")),
    sm.From("presales_presalestatus")),
  ).As("differnce_by_status"),
  sm.Where(sqlite.X("status").In(sqlite.S("A"), sqlite.S("B"), sqlite.S("C"))),
  sm.GroupBy("status"),
)
```
