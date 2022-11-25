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
  qm.Table("films"),
  qm.SetArg("kind", "Dramatic"),
  qm.Where(mysql.X("kind").EQ(mysql.Arg("Drama"))),
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
  qm.Table("employees, accounts"),
  qm.Set("sales_count", "sales_count + 1"),
  qm.Where(mysql.X("accounts.name").EQ(mysql.Arg("Acme Corporation"))),
  qm.Where(mysql.X("employees.id").EQ("accounts.sales_person")),
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
  qm.Table("employees"),
  qm.Set("sales_count", "sales_count + 1"),
  qm.Where(mysql.X("id").EQ(mysql.P(mysql.Select(
    selectQM.Columns("sales_person"),
    selectQM.From("accounts"),
    selectQM.Where(mysql.X("name").EQ(mysql.Arg("Acme Corporation"))),
  )))),
)
```
