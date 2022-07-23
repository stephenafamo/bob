## Simple

SQL:

```sql
UPDATE films SET kind = ?1 WHERE (kind = ?2)
```

Args:

* `"Dramatic"`
* `"Drama"`

Code:

```go
sqlite.Update(
  qm.Table("films"),
  qm.SetArg("kind", "Dramatic"),
  qm.Where(qm.X("kind").EQ(qm.Arg("Drama"))),
)
```

## With From

SQL:

```sql
UPDATE employees SET sales_count = sales_count + 1 FROM accounts
WHERE (accounts.name = ?1)
AND (employees.id = accounts.sales_person)
```

Args:

* `"Acme Corporation"`

Code:

```go
sqlite.Update(
  qm.Table("employees"),
  qm.Set("sales_count", "sales_count + 1"),
  qm.From("accounts"),
  qm.Where(qm.X("accounts.name").EQ(qm.Arg("Acme Corporation"))),
  qm.Where(qm.X("employees.id").EQ("accounts.sales_person")),
)
```

## With Sub-Select

SQL:

```sql
UPDATE employees AS "e" NOT INDEXED
SET sales_count = sales_count + 1
WHERE (id = (SELECT sales_person FROM accounts WHERE (name = ?1)))
```

Args:

* `"Acme Corporation"`

Code:

```go
sqlite.Update(
  qm.TableAs("employees", "e"),
  qm.NotIndexed(),
  qm.Set("sales_count", "sales_count + 1"),
  qm.Where(qm.X("id").EQ(expr.P(Select(
    selectQM.Select("sales_person"),
    selectQM.From("accounts"),
    selectQM.Where(qm.X("name").EQ(qm.Arg("Acme Corporation"))),
  )))),
)
```
