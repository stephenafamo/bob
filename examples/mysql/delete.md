## Simple

SQL:

```sql
DELETE FROM films WHERE (kind = ?)
```

Args:

* `"Drama"`

Code:

```go
mysql.Delete(
  qm.From("films"),
  qm.Where(mysql.X("kind").EQ(mysql.Arg("Drama"))),
)
```

## Multiple Tables

SQL:

```sql
DELETE FROM films, actors WHERE (kind = ?)
```

Args:

* `"Drama"`

Code:

```go
mysql.Delete(
  qm.From("films"),
  qm.From("actors"),
  qm.Where(mysql.X("kind").EQ(mysql.Arg("Drama"))),
)
```

## With Limit And Offest

SQL:

```sql
DELETE FROM films WHERE (kind = ?) ORDER BY producer DESC LIMIT 10
```

Args:

* `"Drama"`

Code:

```go
mysql.Delete(
  qm.From("films"),
  qm.Where(mysql.X("kind").EQ(mysql.Arg("Drama"))),
  qm.Limit(10),
  qm.OrderBy("producer").Desc(),
)
```

## With Using

SQL:

```sql
DELETE FROM employees USING accounts
WHERE (accounts.name = ?)
AND (employees.id = accounts.sales_person)
```

Args:

* `"Acme Corporation"`

Code:

```go
mysql.Delete(
  qm.From("employees"),
  qm.Using("accounts"),
  qm.Where(mysql.X("accounts.name").EQ(mysql.Arg("Acme Corporation"))),
  qm.Where(mysql.X("employees.id").EQ("accounts.sales_person")),
)
```
