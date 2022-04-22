# Examples of PostgreSQL Insert Queries

## Simple Insert

```sql
-- Query
INSERT INTO films VALUES ($1, $2, $3, $4, $5, $6);

-- Args
'UA502', 'Bananas', 105, '1971-07-13', 'Comedy', '82 minutes'
```

```go
psql.Insert(
    qm.Into("films"),
    qm.Values(expr.Arg("UA502", "Bananas", 105, "1971-07-13", "Comedy", "82 minutes")),
)
```

## Bulk Insert

```sql
-- Query
INSERT INTO films VALUES
  ($1, $2, $3, $4, $5, $6),
  ($7, $8, $9, $10, $11, $12);

-- Args
'UA502', 'Bananas', 105, '1971-07-13', 'Comedy', '82 minutes'
'UA502', 'Bananas', 105, '1971-07-13', 'Comedy', '82 minutes'
```

```go
psql.Insert(
    qm.Into("films"),
    qm.Values(expr.Arg("UA502", "Bananas", 105, "1971-07-13", "Comedy", "82 minutes")),
    qm.Values(expr.Arg("UA502", "Bananas", 105, "1971-07-13", "Comedy", "82 minutes")),
)
```

## Upsert DO NOTHING

```sql
-- Query
INSERT INTO films VALUES ($1, $2, $3, $4, $5, $6) ON CONFLICT DO NOTHING;

-- Args
'UA502', 'Bananas', 105, '1971-07-13', 'Comedy', '82 minutes'
```

```go
psql.Insert(
    qm.Into("films"),
    qm.Values(expr.Arg("UA502", "Bananas", 105, "1971-07-13", "Comedy", "82 minutes")),
    qm.OnConflict(nil).DoNothing(),
)
```

## Upsert DO UPDATE

```sql
INSERT INTO distributors AS "d" ("did", "dname")
VALUES ($1, $2), ($3, $4)
ON CONFLICT (did) DO UPDATE
SET dname = EXCLUDED.dname || ' (formerly ' || d.dname || ')'
WHERE d.zipcode <> '21201'
```

```go
Insert(
    qm.Into(expr.T("distributors").As("d", "did", "dname")),
    qm.Values(expr.Arg(8, "Anvil Distribution")),
    qm.Values(expr.Arg(9, "Sentry Distribution")),
    qm.OnConflict("did").DoUpdate().SetEQ(
        "dname",
        expr.CONCAT("EXCLUDED.dname", expr.S(" (formerly "), "d.dname", expr.S(")")),
    ).Where(expr.NE("d.zipcode", expr.S("21201"))),
)
```
