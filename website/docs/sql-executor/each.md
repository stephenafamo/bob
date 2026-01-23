---

sidebar_position: 5
description: Execute a query and iterate over its rows (range-over-func)

---

# Each

Execute a query and iterate over its rows (range-over-func).

This is useful for queries that return a **large** result and where we would rather not scan the entire result into memory.

This function is a wrapper around [`scan.Each`](https://pkg.go.dev/github.com/stephenafamo/scan#Each).

```go
type userObj struct {
    ID int
    Name string
}

ctx := context.Background()
db, err := bob.Open("postgres", "...")
if err != nil {
    // ...
}

q := psql.Select(...)

for user, err := range bob.Each(ctx, db, q, scan.StructMapper[userObj]()) {
    // user is of type userObj{}
}
```
