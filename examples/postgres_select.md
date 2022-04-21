# Examples of PostgreSQL Select Queries

## Simple Select with some conditions

```sql
SELECT id, name FROM users WHERE id IN ($1, $2, $3)
[100 200 300]
```

```go
psql.Select(
    qm.Select("id", "name"),
    qm.From("users"),
    qm.Where(expr.IN("id", expr.Arg(100), expr.Arg(200), expr.Arg(300))),
)
```

## Select from group of functions. Automatically uses the `ROWS FROM` syntax

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

```go
psql.Select(
    qm.From(expr.TFunc(
        expr.Func(
            "json_to_recordset",
            expr.Arg(`[{"a":40,"b":"foo"},{"a":"100","b":"bar"}]`),
        ).Col("a", "INTEGER").Col("b", "TEXT"),
        expr.Func("generate_series", 1, 3),
    ).As("x", "p", "q", "s")),
    qm.OrderBy("p"),
)
```

## Select from subquery with window function

```sql
SELECT status, avg(difference)
FROM (
    SELECT
    status,
    LEAD(created_date, 1, NOW())
    OVER(PARTITION BY presale_id ORDER BY created_date) -
    created_date AS "difference"
    FROM presales_presalestatus
) AS "differnce_by_status"
WHERE status IN ('A', 'B', 'C')
GROUP BY status
```

```go
psql.Select(
    qm.Select("status", expr.Func("avg", "difference")),
    qm.From(expr.TQuery(Select(
        qm.Select(
            "status",
            expr.C(expr.MINUS(expr.OVER(
                expr.Func("LEAD", "created_date", 1, expr.Func("NOW")),
                expr.Window("").PartitionBy("presale_id").OrderBy("created_date"),
            ), "created_date"), "difference"),
        ),
        qm.From("presales_presalestatus"),
    ), false).As("differnce_by_status")),
    qm.Where(expr.IN(
        "status",
        expr.S("A"), expr.S("B"), expr.S("C"),
    )),
    qm.GroupBy("status"),
)
```
