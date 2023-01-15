# Insert

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
  im.Into("films"),
  im.Values(psql.Arg("UA502", "Bananas", 105, "1971-07-13", "Comedy", "82 mins")),
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
  im.Into("films"),
  im.Values(psql.Arg("UA502", "Bananas", 105, "1971-07-13", "Comedy", "82 mins")),
  im.Values(psql.Arg("UA502", "Bananas", 105, "1971-07-13", "Comedy", "82 mins")),
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
  im.IntoAs("distributors", "d", "did", "dname"),
  im.Values(psql.Arg(8, "Anvil Distribution")),
  im.Values(psql.Arg(9, "Sentry Distribution")),
  im.OnConflict("did").DoUpdate().
    Set("dname", psql.Concat(
      "EXCLUDED.dname", psql.S(" (formerly "), "d.dname", psql.S(")"),
    )).
    Where(psql.X("d.zipcode").NE(psql.S("21201"))),
)
```

## Upsert On Constraint

SQL:

```sql
INSERT INTO distributors AS "d" ("did", "dname")
VALUES ($1, $2), ($3, $4)
ON CONFLICT ON CONSTRAINT distributors_pkey DO UPDATE
SET "dname" = EXCLUDED. "dname"
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
  im.IntoAs("distributors", "d", "did", "dname"),
  im.Values(psql.Arg(8, "Anvil Distribution")),
  im.Values(psql.Arg(9, "Sentry Distribution")),
  im.OnConflictOnConstraint("distributors_pkey").
    DoUpdate().
    SetExcluded("dname").
    Where(psql.X("d.zipcode").NE(psql.S("21201"))),
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
  im.Into("films"),
  im.Values(psql.Arg("UA502", "Bananas", 105, "1971-07-13", "Comedy", "82 mins")),
  im.OnConflict().DoNothing(),
)
```
