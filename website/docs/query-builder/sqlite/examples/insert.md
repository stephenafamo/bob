# Insert

## Simple Insert

SQL:

```sql
INSERT INTO films VALUES (?1, ?2, ?3, ?4, ?5, ?6)
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
sqlite.Insert(
  im.Into("films"),
  im.Values(sqlite.Arg("UA502", "Bananas", 105, "1971-07-13", "Comedy", "82 mins")),
)
```

## Bulk Insert

SQL:

```sql
INSERT INTO films VALUES
(?1, ?2, ?3, ?4, ?5, ?6),
(?7, ?8, ?9, ?10, ?11, ?12)
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
sqlite.Insert(
  im.Into("films"),
  im.Values(sqlite.Arg("UA502", "Bananas", 105, "1971-07-13", "Comedy", "82 mins")),
  im.Values(sqlite.Arg("UA502", "Bananas", 105, "1971-07-13", "Comedy", "82 mins")),
)
```

## On Conflict Do Nothing

SQL:

```sql
INSERT INTO films VALUES (?1, ?2, ?3, ?4, ?5, ?6) ON CONFLICT DO NOTHING
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
sqlite.Insert(
  im.Into("films"),
  im.Values(sqlite.Arg("UA502", "Bananas", 105, "1971-07-13", "Comedy", "82 mins")),
  im.OnConflict().DoNothing(),
)
```

## Upsert

SQL:

```sql
INSERT INTO distributors AS "d" ("did", "dname")
VALUES (?1, ?2), (?3, ?4)
ON CONFLICT (did) DO UPDATE
SET "dname" = EXCLUDED. "dname"
WHERE ("d"."zipcode" <> '21201')
```

Args:

* `8`
* `"Anvil Distribution"`
* `9`
* `"Sentry Distribution"`

Code:

```go
sqlite.Insert(
  im.IntoAs("distributors", "d", "did", "dname"),
  im.Values(sqlite.Arg(8, "Anvil Distribution")),
  im.Values(sqlite.Arg(9, "Sentry Distribution")),
  im.OnConflict("did").DoUpdate().
    SetExcluded("dname").
    Where(sqlite.Quote("d", "zipcode").NE(sqlite.S("21201"))),
)
```

## Or Replace

SQL:

```sql
INSERT OR REPLACE INTO distributors ("did", "dname")
VALUES (?1, ?2), (?3, ?4)
```

Args:

* `8`
* `"Anvil Distribution"`
* `9`
* `"Sentry Distribution"`

Code:

```go
sqlite.Insert(
  im.OrReplace(),
  im.Into("distributors", "did", "dname"),
  im.Values(sqlite.Arg(8, "Anvil Distribution")),
  im.Values(sqlite.Arg(9, "Sentry Distribution")),
)
```
