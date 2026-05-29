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
  um.SetCol("kind").ToArg("Dramatic"),
  um.Where(psql.Quote("kind").EQ(psql.Arg("Drama"))),
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
  um.SetCol("sales_count").To("sales_count + 1"),
  um.From("accounts"),
  um.Where(psql.Quote("accounts", "name").EQ(psql.Arg("Acme Corporation"))),
  um.Where(psql.Quote("employees", "id").EQ(psql.Quote("accounts", "sales_person"))),
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
  um.SetCol("sales_count").To("sales_count + 1"),
  um.Where(psql.Quote("id").EQ(psql.Group(psql.Select(
    sm.Columns("sales_person"),
    sm.From("accounts"),
    sm.Where(psql.Quote("name").EQ(psql.Arg("Acme Corporation"))),
  )))),
)
```

## Assign via Set

SQL:

```sql
UPDATE employees SET "sales_count" = sales_count + 1,
"employees"."dept_id" = "accounts"."dept_id" FROM accounts
WHERE (employees.id = $1)
```

Args:

* `1`

Code:

```go
psql.Update(
  um.Table("employees"),
  um.From("accounts"),
  um.Set(
    psql.Quote("sales_count").Assign(psql.Raw("sales_count + 1")),
    psql.Quote("employees", "dept_id").Assign(psql.Quote("accounts", "dept_id")),
  ),
  um.Where(psql.Quote("employees", "id").EQ(psql.Arg(1))),
)
```
