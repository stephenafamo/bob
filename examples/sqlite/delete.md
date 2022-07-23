## Simple

SQL:

```sql
DELETE FROM films WHERE (kind = ?1)
```

Args:

* `"Drama"`

Code:

```go
sqlite.Delete(
  qm.From("films"),
  qm.Where(qm.X("kind").EQ(qm.Arg("Drama"))),
)
```
