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
  dm.From("films"),
  dm.Where(mysql.X("kind").EQ(mysql.Arg("Drama"))),
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
  dm.From("films"),
  dm.From("actors"),
  dm.Where(mysql.X("kind").EQ(mysql.Arg("Drama"))),
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
  dm.From("films"),
  dm.Where(mysql.X("kind").EQ(mysql.Arg("Drama"))),
  dm.Limit(10),
  dm.OrderBy("producer").Desc(),
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
  dm.From("employees"),
  dm.Using("accounts"),
  dm.Where(mysql.X("accounts.name").EQ(mysql.Arg("Acme Corporation"))),
  dm.Where(mysql.X("employees.id").EQ("accounts.sales_person")),
)
```
