# Delete

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
  dm.From("films"),
  dm.Where(psql.Quote("kind").EQ(psql.Arg("Drama"))),
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
  dm.From("employees"),
  dm.Using("accounts"),
  dm.Where(psql.Quote("accounts", "name").EQ(psql.Arg("Acme Corporation"))),
  dm.Where(psql.Quote("employees", "id").EQ(psql.Quote("accounts", "sales_person"))),
)
```
