## Simple

SQL:

```sql
DELETE FROM films WHERE (kind = $1)
```

Args:

* `"Drama"`

Code:

```go
psql.Delete(
  qm.From("films"),
  qm.Where(psql.X("kind").EQ(psql.Arg("Drama"))),
)
```

## With Using

SQL:

```sql
DELETE FROM employees USING accounts
WHERE (accounts.name = $1)
AND (employees.id = accounts.sales_person)
```

Args:

* `"Acme Corporation"`

Code:

```go
psql.Delete(
  qm.From("employees"),
  qm.Using("accounts"),
  qm.Where(psql.X("accounts.name").EQ(psql.Arg("Acme Corporation"))),
  qm.Where(psql.X("employees.id").EQ("accounts.sales_person")),
)
```
