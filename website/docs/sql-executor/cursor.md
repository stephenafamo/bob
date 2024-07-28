---

sidebar_position: 5
description: Execute a query and returns a cursor that yields a defined struct per row

---

# Cursor

Execute a query and returns a cursor that yields a defined struct per row.

This is useful for queries that return a **large** result and where we would rather not scan the entire result into memory.

Using this is very similar to using `*sql.Rows` so most Go developers should be familiar with it.

This function is a wrapper around [`scan.Cursor`](https://pkg.go.dev/github.com/stephenafamo/scan#Cursor).

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

// user is of type userObj{}
cursor, err := bob.Cursor(ctx, db, q, scan.StructMapper[userObj]())
if err != nil {
    // ...
}
defer cursor.Close() // make sure to close

for cursor.Next() {
    user, err := cursor.Get() // scan the next row into the concrete type
}
```
