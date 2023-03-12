# Delete

## Simple

SQL:

```sql
DELETE FROM films WHERE ("kind" = ?1)
```

Args:

* `"Drama"`

Code:

```go
sqlite.Delete(
  dm.From("films"),
  dm.Where(sqlite.Quote("kind").EQ(sqlite.Arg("Drama"))),
)
```
