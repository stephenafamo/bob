# Examples of PostgreSQL Update Queries

## Simple Update

```sql
-- Query
UPDATE films SET kind = $1 WHERE kind = $2;

-- Args
'Dramatic', 'Drama'
```

```go
psql.Update(
    qm.Table("films"),
    qm.SetEQ("kind", expr.Arg("Dramatic")),
    qm.Where(expr.EQ("kind", expr.Arg("Drama"))),
)
```

## Update with From

```sql
-- Query
UPDATE employees SET sales_count = sales_count + 1 FROM accounts
WHERE accounts.name = $1
AND employees.id = accounts.sales_person;

-- Args
'Acme Corporation'
```

```go
psql.Update(
    qm.Table("employees"),
    qm.SetEQ("sales_count", "sales_count + 1"),
    qm.From("accounts"),
    qm.Where(expr.EQ("accounts.name", expr.Arg("Acme Corporation"))),
    qm.Where(expr.EQ("employees.id", "accounts.sales_person")),
)
```

## Update with sub-select

```sql
-- Query
UPDATE employees SET sales_count = sales_count + 1 WHERE id =
(SELECT sales_person FROM accounts WHERE name = $1)

-- Args
'Acme Corporation'
```

```go
psql.Update(
    qm.Table("employees"),
    qm.SetEQ("sales_count", "sales_count + 1"),
    qm.Where(expr.EQ("id", expr.P(Select(
        selectQM.Select("sales_person"),
        selectQM.From("accounts"),
        selectQM.Where(expr.EQ("name", expr.Arg("Acme Corporation"))),
    )))),
)
```
