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

## Case With Else

SQL:

```sql
SELECT id, name, (CASE WHEN (id = '1') THEN 'A' ELSE 'B' END) AS "C" FROM users
```

Code:

```go
psql.Select(
  sm.Columns(
    "id",
    "name",
    psql.Case().
      When(psql.Quote("id").EQ(psql.S("1")), psql.S("A")).
      Else(psql.S("B")).
      As("C"),
  ),
  sm.From("users"),
)
```

## Case Without Else

SQL:

```sql
SELECT id, name, (CASE WHEN (id = '1') THEN 'A' END) AS "C" FROM users
```

Code:

```go
psql.Select(
  sm.Columns(
    "id",
    "name",
    psql.Case().
      When(psql.Quote("id").EQ(psql.S("1")), psql.S("A")).
      End().
      As("C"),
  ),
  sm.From("users"),
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
        fm.Over(
          wm.PartitionBy("presale_id"),
          wm.OrderBy("created_date"),
        ),
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
    psql.F("avg", "salary")(fm.Over(wm.BasedOn("w"))),
  ),
  sm.From("c"),
  sm.Window("w", wm.PartitionBy("depname"), wm.OrderBy("salary")),
)
```

## Select With Order By And Collate

SQL:

```sql
SELECT id, name FROM users ORDER BY name COLLATE "bg-BG-x-icu" ASC
```

Code:

```go
psql.Select(
  sm.Columns("id", "name"),
  sm.From("users"),
  sm.OrderBy("name").Collate("bg-BG-x-icu").Asc(),
)
```

## With Cross Join

SQL:

```sql
SELECT id, name, type
FROM users AS u CROSS JOIN (
  SELECT id, type
  FROM clients
  WHERE ("client_id" = $1)
) AS "clients"
WHERE ("id" = $2)
```

Args:

* `"123"`
* `100`

Code:

```go
psql.Select(
  sm.Columns("id", "name", "type"),
  sm.From("users").As("u"),
  sm.CrossJoin(psql.Select(
    sm.Columns("id", "type"),
    sm.From("clients"),
    sm.Where(psql.Quote("client_id").EQ(psql.Arg("123"))),
  )).As("clients"),
  sm.Where(psql.Quote("id").EQ(psql.Arg(100))),
)
```

## With Locking

SQL:

```sql
SELECT id, name FROM users FOR UPDATE OF users SKIP LOCKED
```

Code:

```go
psql.Select(
  sm.Columns("id", "name"),
  sm.From("users"),
  sm.ForUpdate("users").SkipLocked(),
)
```

## Multiple Unions

SQL:

```sql
SELECT id, name FROM users UNION select id, name FROM admins UNION select id, name FROM mods
```

Code:

```go
psql.Select(
  sm.Columns("id", "name"),
  sm.From("users"),
  sm.Union(psql.Select(
    sm.Columns("id", "name"),
    sm.From("admins"),
  )),
  sm.Union(psql.Select(
    sm.Columns("id", "name"),
    sm.From("mods"),
  )),
)
```

## Union With Combined Args

SQL:

```sql
(SELECT id, name FROM users ORDER BY id LIMIT 100) UNION (SELECT id, name FROM admins ORDER BY id LIMIT 10)
ORDER BY id LIMIT 1000
```

Code:

```go
psql.Select(
  sm.Columns("id", "name"),
  sm.From("users"),
  sm.Limit(100),
  sm.OrderBy("id"),
  sm.Union(psql.Select(
    sm.Columns("id", "name"),
    sm.From("admins"),
    sm.Limit(10),
    sm.OrderBy("id"),
  )),
  sm.OrderCombined("id"),
  sm.LimitCombined(1000),
)
```

## Union With Uncombined Args

SQL:

```sql
(SELECT id, name FROM users ORDER BY id LIMIT 1) UNION (SELECT id, name FROM admins ORDER BY id LIMIT 1)
```

Code:

```go
psql.Select(
  sm.Columns("id", "name"),
  sm.From("users"),
  sm.Limit(1),
  sm.OrderBy("id"),
  sm.Union(psql.Select(
    sm.Columns("id", "name"),
    sm.From("admins"),
    sm.Limit(1),
    sm.OrderBy("id"),
  )),
)
```
