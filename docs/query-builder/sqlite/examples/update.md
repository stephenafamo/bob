# Update

## Simple

SQL:

```sql
UPDATE films SET "kind" = ?1 WHERE (kind = ?2)
```

Args:

* `"Dramatic"`
* `"Drama"`

Code:

```go
sqlite.Update(
  um.Table("films"),
  um.SetArg("kind", "Dramatic"),
  um.Where(sqlite.X("kind").EQ(sqlite.Arg("Drama"))),
)
```

## With From

SQL:

```sql
UPDATE employees SET "sales_count" = sales_count + 1 FROM accounts
WHERE (accounts.name = ?1)
AND (employees.id = accounts.sales_person)
```

Args:

* `"Acme Corporation"`

Code:

```go
sqlite.Update(
  um.Table("employees"),
  um.Set("sales_count", "sales_count + 1"),
  um.From("accounts"),
  um.Where(sqlite.X("accounts.name").EQ(sqlite.Arg("Acme Corporation"))),
  um.Where(sqlite.X("employees.id").EQ("accounts.sales_person")),
)
```

## With Sub-Select

SQL:

```sql
UPDATE employees AS "e" NOT INDEXED
SET "sales_count" = sales_count + 1
WHERE (id = (SELECT sales_person FROM accounts WHERE (name = ?1)))
```

Args:

* `"Acme Corporation"`

Code:

```go
sqlite.Update(
  um.TableAs("employees", "e"),
  um.TableNotIndexed(),
  um.Set("sales_count", "sales_count + 1"),
  um.Where(sqlite.X("id").EQ(sqlite.P(sqlite.Select(
    sm.Columns("sales_person"),
    sm.From("accounts"),
    sm.Where(sqlite.X("name").EQ(sqlite.Arg("Acme Corporation"))),
  )))),
)
```
