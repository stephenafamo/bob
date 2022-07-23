## Simple Insert

SQL:

```sql
INSERT INTO films VALUES ($1, $2, $3, $4, $5, $6)
```

Args:

* `"UA502"`
* `"Bananas"`
* `105`
* `"1971-07-13"`
* `"Comedy"`
* `"82 mins"`

Code:

```go
psql.Insert(
  qm.Into("films"),
  qm.Values(qm.Arg("UA502", "Bananas", 105, "1971-07-13", "Comedy", "82 mins")),
)
```

## Bulk Insert

SQL:

```sql
INSERT INTO films VALUES
($1, $2, $3, $4, $5, $6),
($7, $8, $9, $10, $11, $12)
```

Args:

* `"UA502"`
* `"Bananas"`
* `105`
* `"1971-07-13"`
* `"Comedy"`
* `"82 mins"`
* `"UA502"`
* `"Bananas"`
* `105`
* `"1971-07-13"`
* `"Comedy"`
* `"82 mins"`

Code:

```go
psql.Insert(
  qm.Into("films"),
  qm.Values(qm.Arg("UA502", "Bananas", 105, "1971-07-13", "Comedy", "82 mins")),
  qm.Values(qm.Arg("UA502", "Bananas", 105, "1971-07-13", "Comedy", "82 mins")),
)
```

## Upsert

SQL:

```sql
INSERT INTO distributors AS "d" ("did", "dname")
VALUES ($1, $2), ($3, $4)
ON CONFLICT (did) DO UPDATE
SET dname = (EXCLUDED.dname || ' (formerly ' || d.dname || ')')
WHERE (d.zipcode <> '21201')
```

Args:

* `8`
* `"Anvil Distribution"`
* `9`
* `"Sentry Distribution"`

Code:

```go
psql.Insert(
  qm.IntoAs("distributors", "d", "did", "dname"),
  qm.Values(qm.Arg(8, "Anvil Distribution")),
  qm.Values(qm.Arg(9, "Sentry Distribution")),
  qm.OnConflict("did").DoUpdate().Set(
    "dname",
    qm.CONCAT(
      "EXCLUDED.dname", expr.S(" (formerly "), "d.dname", expr.S(")"),
    ),
  ).Where(qm.X("d.zipcode").NE(expr.S("21201"))),
)
```

## Upsert DO NOTHING

SQL:

```sql
INSERT INTO films VALUES ($1, $2, $3, $4, $5, $6) ON CONFLICT DO NOTHING
```

Args:

* `"UA502"`
* `"Bananas"`
* `105`
* `"1971-07-13"`
* `"Comedy"`
* `"82 mins"`

Code:

```go
psql.Insert(
  qm.Into("films"),
  qm.Values(qm.Arg("UA502", "Bananas", 105, "1971-07-13", "Comedy", "82 mins")),
  qm.OnConflict(nil).DoNothing(),
)
```
