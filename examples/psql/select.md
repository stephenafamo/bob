## Simple Select with some conditions

SQL:

```sql
SELECT id, name FROM users WHERE (id IN ($1, $2, $3))
```

Args:

* `100`
* `200`
* `300`

Code:

```go
psql.Select(
  qm.Select("id", "name"),
  qm.From("users"),
  qm.Where(qm.X("id").IN(qm.Arg(100, 200, 300))),
)
```

## Select from group of functions. Automatically uses the `ROWS FROM` syntax

SQL:

```sql
SELECT *
FROM ROWS FROM
  (
    json_to_recordset($1)
      AS (a INTEGER, b TEXT),
    generate_series(1, 3)
  ) AS "x" ("p", "q", "s")
ORDER BY p
```

Args:

* ``[{"a":40,"b":"foo"},{"a":"100","b":"bar"}]``

Code:

```go
psql.Select(
  qm.From(
    expr.Func(
      "json_to_recordset",
      qm.Arg(`[{"a":40,"b":"foo"},{"a":"100","b":"bar"}]`),
    ).Col("a", "INTEGER").Col("b", "TEXT").ToMod(),
    expr.Func("generate_series", 1, 3).ToMod(),
    qm.As("x", "p", "q", "s"),
  ),
  qm.OrderBy("p"),
)
```

## Select from subquery with window function

SQL:

```sql
SELECT status, avg(difference)
FROM (
  SELECT
  status, ((
    LEAD(created_date, 1, NOW())
    OVER(PARTITION BY presale_id ORDER BY created_date)
  ) - created_date) AS "difference"
  FROM presales_presalestatus
) AS "differnce_by_status"
WHERE (status IN ('A', 'B', 'C'))
GROUP BY status
```

Code:

```go
psql.Select(
  qm.Select("status", expr.Func("avg", "difference")),
  qm.From(
    Select(
      qm.Select(
        "status",
        qm.OVER(
          expr.Func("LEAD", "created_date", 1, expr.Func("NOW")),
          expr.Window("").PartitionBy("presale_id").OrderBy("created_date"),
        ).MINUS("created_date").AS("difference"),
      ),
      qm.From("presales_presalestatus"),
    ),
    qm.As("differnce_by_status"),
  ),
  qm.Where(qm.X("status").IN(qm.S("A"), qm.S("B"), qm.S("C"))),
  qm.GroupBy("status"),
)
```
