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
  qm.Into("films"),
  qm.Values(mysql.Arg("UA502", "Bananas", 105, "1971-07-13", "Comedy", "82 mins")),
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
  qm.Into("films"),
  qm.Values(mysql.Arg("UA502", "Bananas", 105, "1971-07-13", "Comedy", "82 mins")),
  qm.Values(mysql.Arg("UA502", "Bananas", 105, "1971-07-13", "Comedy", "82 mins")),
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
  qm.Into("films"),
  qm.HighPriority(),
  qm.Ignore(),
  qm.Values(mysql.Arg("UA502", "Bananas", 105, "1971-07-13", "Comedy", "82 mins")),
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
  qm.Into("films"),
  qm.MaxExecutionTime(1000),
  qm.SetVar("cte_max_recursion_depth = 1M"),
  qm.Values(mysql.Arg("UA502", "Bananas", 105, "1971-07-13", "Comedy", "82 mins")),
)
```

## Upsert

SQL:

```sql
INSERT INTO distributors (`did`, `dname`)
VALUES (?, ?), (?, ?)
AS new
ON DUPLICATE KEY UPDATE
`dbname` = (new.dname || ' (formerly ' || d.dname || ')')
```

Args:

* `8`
* `"Anvil Distribution"`
* `9`
* `"Sentry Distribution"`

Code:

```go
mysql.Insert(
  qm.Into("distributors", "did", "dname"),
  qm.Values(mysql.Arg(8, "Anvil Distribution")),
  qm.Values(mysql.Arg(9, "Sentry Distribution")),
  qm.As("new"),
  qm.OnDuplicateKeyUpdate().
    Set("dbname", mysql.Concat(
      "new.dname", mysql.S(" (formerly "), "d.dname", mysql.S(")"),
    )),
)
```
