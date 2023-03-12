# Update

## Simple

SQL:

```sql
UPDATE films SET `kind` = ? WHERE (`kind` = ?)
```

Args:

* `"Dramatic"`
* `"Drama"`

Code:

```go
mysql.Update(
  um.Table("films"),
  um.Set("kind").ToArg("Dramatic"),
  um.Where(mysql.Quote("kind").EQ(mysql.Arg("Drama"))),
)
```

## Update Multiple Tables

SQL:

```sql
UPDATE employees, accounts SET `sales_count` = sales_count + 1 WHERE (`accounts`.`name` = ?) AND (`employees`.`id` = `accounts`.`sales_person`)
```

Args:

* `"Acme Corporation"`

Code:

```go
mysql.Update(
  um.Table("employees, accounts"),
  um.Set("sales_count").To("sales_count + 1"),
  um.Where(mysql.Quote("accounts", "name").EQ(mysql.Arg("Acme Corporation"))),
  um.Where(mysql.Quote("employees", "id").EQ(mysql.Quote("accounts", "sales_person"))),
)
```

## Update Multiple Tables 2

SQL:

```sql
UPDATE `table1` AS `T1` LEFT JOIN `table2` AS `T2` ON (`T1`.`some_id` = `T2`.`id`) SET `T1`.`some_value` = ? WHERE (`T1`.`id` = ?) AND (`T2`.`other_value` = ?)
```

Args:

* `"test"`
* `1`
* `"something"`

Code:

```go
mysql.Update(
  um.Table(mysql.Quote("table1").As("T1")),
  um.LeftJoin(mysql.Quote("table2").As("T2")).
    OnEQ(mysql.Quote("T1", "some_id"), mysql.Quote("T2", "id")),
  um.Set("T1", "some_value").ToArg("test"),
  um.Where(mysql.Quote("T1", "id").EQ(mysql.Arg(1))),
  um.Where(mysql.Quote("T2", "other_value").EQ(mysql.Arg("something"))),
)
```

## With Sub-Select

SQL:

```sql
UPDATE employees SET `sales_count` = sales_count + 1 WHERE (`id` = (SELECT sales_person FROM accounts WHERE (`name` = ?)))
```

Args:

* `"Acme Corporation"`

Code:

```go
mysql.Update(
  um.Table("employees"),
  um.Set("sales_count").To("sales_count + 1"),
  um.Where(mysql.Quote("id").EQ(mysql.Group(mysql.Select(
    sm.Columns("sales_person"),
    sm.From("accounts"),
    sm.Where(mysql.Quote("name").EQ(mysql.Arg("Acme Corporation"))),
  )))),
)
```
