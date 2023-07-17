# Insert

## Simple Insert

SQL:

```sql
INSERT INTO films VALUES (?, ?, ?, ?, ?, ?)
```

Args:

* `"UA502"`
* `"Bananas"`
* `105`
* `"1971-07-13"`
* `"Comedy"`
* `"82 mins"`

Code:

```go
mysql.Insert(
  im.Into("films"),
  im.Values(mysql.Arg("UA502", "Bananas", 105, "1971-07-13", "Comedy", "82 mins")),
)
```

## Insert From Select

SQL:

```sql
INSERT INTO films SELECT * FROM tmp_films WHERE (`date_prod` < ?)
```

Args:

* `"1971-07-13"`

Code:

```go
mysql.Insert(
  im.Into("films"),
  im.Query(mysql.Select(
    sm.From("tmp_films"),
    sm.Where(mysql.Quote("date_prod").LT(mysql.Arg("1971-07-13"))),
  )),
)
```

## Bulk Insert

SQL:

```sql
INSERT INTO films VALUES
(?, ?, ?, ?, ?, ?),
(?, ?, ?, ?, ?, ?)
```

Args:

* `"UA502"`
* `"Bananas"`
* `105`
* `"1971-07-13"`
* `"Comedy"`
* `"82 mins"`
* `"UA502"`
* `"Bananas"`
* `105`
* `"1971-07-13"`
* `"Comedy"`
* `"82 mins"`

Code:

```go
mysql.Insert(
  im.Into("films"),
  im.Values(mysql.Arg("UA502", "Bananas", 105, "1971-07-13", "Comedy", "82 mins")),
  im.Values(mysql.Arg("UA502", "Bananas", 105, "1971-07-13", "Comedy", "82 mins")),
)
```

## With High Priority And Ignore Modifier

SQL:

```sql
INSERT HIGH_PRIORITY IGNORE INTO films VALUES (?, ?, ?, ?, ?, ?)
```

Args:

* `"UA502"`
* `"Bananas"`
* `105`
* `"1971-07-13"`
* `"Comedy"`
* `"82 mins"`

Code:

```go
mysql.Insert(
  im.Into("films"),
  im.HighPriority(),
  im.Ignore(),
  im.Values(mysql.Arg("UA502", "Bananas", 105, "1971-07-13", "Comedy", "82 mins")),
)
```

## With Optimizer Hints

SQL:

```sql
INSERT
/*+
    MAX_EXECUTION_TIME(1000)
    SET_VAR(cte_max_recursion_depth = 1M)
*/ INTO films VALUES (?, ?, ?, ?, ?, ?)
```

Args:

* `"UA502"`
* `"Bananas"`
* `105`
* `"1971-07-13"`
* `"Comedy"`
* `"82 mins"`

Code:

```go
mysql.Insert(
  im.Into("films"),
  im.MaxExecutionTime(1000),
  im.SetVar("cte_max_recursion_depth = 1M"),
  im.Values(mysql.Arg("UA502", "Bananas", 105, "1971-07-13", "Comedy", "82 mins")),
)
```

## Upsert

SQL:

```sql
INSERT INTO distributors (`did`, `dname`)
VALUES (?, ?), (?, ?)
AS new
ON DUPLICATE KEY UPDATE
`did` = `new`.`did`,
`dbname` = (`new`.`dname` || ' (formerly ' || `d`.`dname` || ')')
```

Args:

* `8`
* `"Anvil Distribution"`
* `9`
* `"Sentry Distribution"`

Code:

```go
mysql.Insert(
  im.Into("distributors", "did", "dname"),
  im.Values(mysql.Arg(8, "Anvil Distribution")),
  im.Values(mysql.Arg(9, "Sentry Distribution")),
  im.As("new"),
  im.OnDuplicateKeyUpdate().
    Set("new", "did").
    SetCol("dbname", mysql.Concat(
      mysql.Quote("new", "dname"), mysql.S(" (formerly "),
      mysql.Quote("d", "dname"), mysql.S(")"),
    )),
)
```

## Upsert2

SQL:

```sql
INSERT INTO distributors (`did`, `dname`)
VALUES (?, ?), (?, ?)
ON DUPLICATE KEY UPDATE
`did` = VALUES(`did`),
`dbname` = VALUES(`dbname`)
```

Args:

* `8`
* `"Anvil Distribution"`
* `9`
* `"Sentry Distribution"`

Code:

```go
mysql.Insert(
  im.Into("distributors", "did", "dname"),
  im.Values(mysql.Arg(8, "Anvil Distribution")),
  im.Values(mysql.Arg(9, "Sentry Distribution")),
  im.OnDuplicateKeyUpdate().SetValues("did", "dbname"),
)
```
