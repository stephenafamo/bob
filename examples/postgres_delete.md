# Examples of PostgreSQL Delete Queries

## Simple Delete Query

```sql
-- Query
DELETE FROM films WHERE kind = $1

-- Args
'Drama'
```

```go
psql.Delete(
    qm.From("films"),
    qm.Where(expr.EQ("kind", expr.Arg("Drama"))),
)
```

## Delete with USING clause

```sql
-- Query
DELETE FROM employees USING accounts
WHERE accounts.name = $1
AND employees.id = accounts.sales_person

-- Args
'Acme Corporation'
```

```go
psql.Delete(
    qm.From("employees"),
    qm.Using("accounts"),
    qm.Where(expr.EQ("accounts.name", expr.Arg("Acme Corporation"))),
    qm.Where(expr.EQ("employees.id", "accounts.sales_person")),
)
```
