# Select

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
  sm.Columns("id", "name"),
  sm.From("users"),
  sm.Where(psql.Quote("id").In(psql.Arg(100, 200, 300))),
)
```

## Select Distinct

SQL:

```sql
SELECT DISTINCT id, name FROM users WHERE (id IN ($1, $2, $3))
```

Args:

* `100`
* `200`
* `300`

Code:

```go
psql.Select(
  sm.Columns("id", "name"),
  sm.Distinct(),
  sm.From("users"),
  sm.Where(psql.Quote("id").In(psql.Arg(100, 200, 300))),
)
```

## Select Distinct On

SQL:

```sql
SELECT DISTINCT ON(id) id, name FROM users WHERE (id IN ($1, $2, $3))
```

Args:

* `100`
* `200`
* `300`

Code:

```go
psql.Select(
  sm.Columns("id", "name"),
  sm.Distinct("id"),
  sm.From("users"),
  sm.Where(psql.Quote("id").In(psql.Arg(100, 200, 300))),
)
```

## Select From Function

SQL:

```sql
SELECT * FROM generate_series(1, 3) AS "x" ("p", "q", "s")
```

Code:

```go
psql.Select(
  sm.From(psql.F("generate_series", 1, 3)).As("x", "p", "q", "s"),
)
```

## Select from group of functions. Automatically uses the `ROWS FROM` syntax

SQL:

```sql
SELECT *
FROM ROWS FROM
  (
    json_to_recordset($1) AS (a INTEGER, b TEXT),
    generate_series(1, 3)
  ) AS "x" ("p", "q", "s")
ORDER BY p
```

Args:

* ``[{"a":40,"b":"foo"},{"a":"100","b":"bar"}]``

Code:

```go
psql.Select(
  sm.FromFunction(
    psql.F(
      "json_to_recordset",
      psql.Arg(`[{"a":40,"b":"foo"},{"a":"100","b":"bar"}]`),
    )(
      fm.Columns("a", "INTEGER"),
      fm.Columns("b", "TEXT"),
    ),
    psql.F("generate_series", 1, 3)(),
  ).As("x", "p", "q", "s"),
  sm.OrderBy("p"),
)
```

## Select from subquery with window function

SQL:

```sql
SELECT status, avg(difference)
FROM (
  SELECT
    status, 
    (LEAD(created_date, 1, NOW())
    OVER(PARTITION BY presale_id ORDER BY created_date)
     - "created_date") AS "difference"
  FROM presales_presalestatus
) AS "differnce_by_status"
WHERE status IN ('A', 'B', 'C')
GROUP BY status
```

Code:

```go
psql.Select(
  sm.Columns("status", psql.F("avg", "difference")),
  sm.From(psql.Select(
    sm.Columns(
      "status",
      psql.F("LEAD", "created_date", 1, psql.F("NOW"))(
        fm.Over().PartitionBy("presale_id").OrderBy("created_date"),
      ).Minus(psql.Quote("created_date")).As("difference")),
    sm.From("presales_presalestatus")),
  ).As("differnce_by_status"),
  sm.Where(psql.Quote("status").In(psql.S("A"), psql.S("B"), psql.S("C"))),
  sm.GroupBy("status"),
)
```

## Select With Grouped IN

SQL:

```sql
SELECT id, name FROM users WHERE (id, employee_id) IN (($1, $2), ($3, $4))
```

Args:

* `100`
* `200`
* `300`
* `400`

Code:

```go
psql.Select(
  sm.Columns("id", "name"),
  sm.From("users"),
  sm.Where(
    psql.Group(psql.Quote("id"), psql.Quote("employee_id")).
      In(psql.ArgGroup(100, 200), psql.ArgGroup(300, 400))),
)
```

## Simple select with limit and offset as argument

SQL:

```sql
SELECT id, name FROM users LIMIT $1 OFFSET $2
```

Args:

* `10`
* `15`

Code:

```go
psql.Select(
  sm.Columns("id", "name"),
  sm.From("users"),
  sm.Offset(psql.Arg(15)),
  sm.Limit(psql.Arg(10)),
)
```

## Join Using

SQL:

```sql
SELECT id FROM test1 LEFT JOIN test2 USING (id)
```

Code:

```go
psql.Select(
  sm.Columns("id"),
  sm.From("test1"),
  sm.LeftJoin("test2").Using("id"),
)
```

## CTE With Column Aliases

SQL:

```sql
WITH c(id, data) AS (SELECT id FROM test1 LEFT JOIN test2 USING (id)) SELECT * FROM c
```

Code:

```go
psql.Select(
  sm.With("c", "id", "data").As(psql.Select(
    sm.Columns("id"),
    sm.From("test1"),
    sm.LeftJoin("test2").Using("id"),
  )),
  sm.From("c"),
)
```

## Window Function Over Empty Frame

SQL:

```sql
SELECT row_number() OVER () FROM c
```

Code:

```go
psql.Select(
  sm.Columns(
    psql.F("row_number")(fm.Over()),
  ),
  sm.From("c"),
)
```

## Window Function Over Window Name

SQL:

```sql
SELECT avg(salary) OVER (w)
FROM c 
WINDOW w AS (PARTITION BY depname ORDER BY salary)
```

Code:

```go
psql.Select(
  sm.Columns(
    psql.F("avg", "salary")(fm.Over().From("w")),
  ),
  sm.From("c"),
  sm.Window("w").PartitionBy("depname").OrderBy("salary"),
)
```
