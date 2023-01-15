# Update

## Simple

SQL:

```sql
UPDATE films SET `kind` = ? WHERE (kind = ?)
```

Args:

* `"Dramatic"`
* `"Drama"`

Code:

```go
mysql.Update(
  um.Table("films"),
  um.SetArg("kind", "Dramatic"),
  um.Where(mysql.X("kind").EQ(mysql.Arg("Drama"))),
)
```

## Update Multiple Tables

SQL:

```sql
UPDATE employees, accounts
SET `sales_count` = sales_count + 1 
WHERE (accounts.name = ?)
AND (employees.id = accounts.sales_person)
```

Args:

* `"Acme Corporation"`

Code:

```go
mysql.Update(
  um.Table("employees, accounts"),
  um.Set("sales_count", "sales_count + 1"),
  um.Where(mysql.X("accounts.name").EQ(mysql.Arg("Acme Corporation"))),
  um.Where(mysql.X("employees.id").EQ("accounts.sales_person")),
)
```

## With Sub-Select

SQL:

```sql
UPDATE employees SET `sales_count` = sales_count + 1 WHERE (id =
(SELECT sales_person FROM accounts WHERE (name = ?)))
```

Args:

* `"Acme Corporation"`

Code:

```go
mysql.Update(
  um.Table("employees"),
  um.Set("sales_count", "sales_count + 1"),
  um.Where(mysql.X("id").EQ(mysql.P(mysql.Select(
    sm.Columns("sales_person"),
    sm.From("accounts"),
    sm.Where(mysql.X("name").EQ(mysql.Arg("Acme Corporation"))),
  )))),
)
```
