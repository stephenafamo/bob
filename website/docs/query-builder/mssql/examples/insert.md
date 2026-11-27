# Insert

## Simple Insert

SQL:

```sql
INSERT INTO films VALUES (@p1, @p2, @p3, @p4, @p5, @p6)
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
mssql.Insert(
  im.Into("films"),
  im.Values(mssql.Arg("UA502", "Bananas", 105, "1971-07-13", "Comedy", "82 mins")),
)
```

## Insert With Columns

SQL:

```sql
INSERT INTO films ([code], [title], [len]) VALUES (@p1, @p2, @p3)
```

Args:

* `"UA502"`
* `"Bananas"`
* `105`

Code:

```go
mssql.Insert(
  im.Into("films", "code", "title", "len"),
  im.Values(mssql.Arg("UA502", "Bananas", 105)),
)
```

## Insert From Select

SQL:

```sql
INSERT INTO films SELECT * FROM tmp_films WHERE ([date_prod] < @p1)
```

Args:

* `"1971-07-13"`

Code:

```go
mssql.Insert(
  im.Into("films"),
  im.Query(mssql.Select(
    sm.From("tmp_films"),
    sm.Where(mssql.Quote("date_prod").LT(mssql.Arg("1971-07-13"))),
  )),
)
```

## Bulk Insert

SQL:

```sql
INSERT INTO films VALUES
(@p1, @p2, @p3),
(@p4, @p5, @p6)
```

Args:

* `"UA502"`
* `"Bananas"`
* `105`
* `"UA503"`
* `"Annie Hall"`
* `93`

Code:

```go
mssql.Insert(
  im.Into("films"),
  im.Values(mssql.Arg("UA502", "Bananas", 105)),
  im.Values(mssql.Arg("UA503", "Annie Hall", 93)),
)
```

## Insert with OUTPUT clause (T-SQL equivalent of RETURNING)

SQL:

```sql
INSERT INTO films ([title], [len]) OUTPUT inserted.id, inserted.title VALUES (@p1, @p2)
```

Args:

* `"Bananas"`
* `105`

Code:

```go
mssql.Insert(
  im.Into("films", "title", "len"),
  im.Output("inserted.id", "inserted.title"),
  im.Values(mssql.Arg("Bananas", 105)),
)
```

## Insert with TOP to limit rows inserted from a select

SQL:

```sql
INSERT TOP (100) INTO target_table SELECT * FROM source_table
```

Code:

```go
mssql.Insert(
  im.Into("target_table"),
  im.Top(100),
  im.Query(mssql.Select(
    sm.From("source_table"),
  )),
)
```

## Insert With CTE

SQL:

```sql
WITH [new_films] AS (SELECT title, len FROM tmp_films WHERE ([len] > @p1)) INSERT INTO films ([title], [len]) SELECT * FROM new_films
```

Args:

* `120`

Code:

```go
mssql.Insert(
  im.With("new_films").As(mssql.Select(
    sm.Columns("title", "len"),
    sm.From("tmp_films"),
    sm.Where(mssql.Quote("len").GT(mssql.Arg(120))),
  )),
  im.Into("films", "title", "len"),
  im.Query(mssql.Select(
    sm.From("new_films"),
  )),
)
```
