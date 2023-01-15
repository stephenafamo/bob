# Update

## Simple

SQL:

```sql
UPDATE films SET "kind" = $1 WHERE (kind = $2)
```

Args:

* `"Dramatic"`
* `"Drama"`

Code:

```go
psql.Update(
  um.Table("films"),
  um.SetArg("kind", "Dramatic"),
  um.Where(psql.X("kind").EQ(psql.Arg("Drama"))),
)
```

## With From

SQL:

```sql
UPDATE employees SET "sales_count" = sales_count + 1 FROM accounts
WHERE (accounts.name = $1)
AND (employees.id = accounts.sales_person)
```

Args:

* `"Acme Corporation"`

Code:

```go
psql.Update(
  um.Table("employees"),
  um.Set("sales_count", "sales_count + 1"),
  um.From("accounts"),
  um.Where(psql.X("accounts.name").EQ(psql.Arg("Acme Corporation"))),
  um.Where(psql.X("employees.id").EQ("accounts.sales_person")),
)
```

## With Sub-Select

SQL:

```sql
UPDATE employees SET "sales_count" = sales_count + 1 WHERE (id =
(SELECT sales_person FROM accounts WHERE (name = $1)))
```

Args:

* `"Acme Corporation"`

Code:

```go
psql.Update(
  um.Table("employees"),
  um.Set("sales_count", "sales_count + 1"),
  um.Where(psql.X("id").EQ(psql.P(psql.Select(
    sm.Columns("sales_person"),
    sm.From("accounts"),
    sm.Where(psql.X("name").EQ(psql.Arg("Acme Corporation"))),
  )))),
)
```
