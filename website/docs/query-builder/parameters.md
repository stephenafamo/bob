---

sidebar_position: 4
descripton: Adding parameters and arguments to Bob queries

---

# Parameters

To prevent SQL injection, it is necessary to use parameters in our queries. With `bob` use `sm.Arg()` where necessary. This will write the placeholder correctly in the generated sql, and return the value in the argument slice.

```go
// args: 100, "Stephen"
// Postgres: SELECT * from users WHERE "id" = $1 AND "name" = $2
// SQLite: SELECT * from users WHERE "id" = ?1 AND "name" = ?2
// MySQL: SELECT * from users WHERE "id" = ? AND "name" = ?
psql.Select(
    sm.From("users"),
    sm.Where(psql.Quote("id").EQ(psql.Arg(100))),
    sm.Where(psql.Quote("name").EQ(psql.Arg("Stephen"))),
)
```
