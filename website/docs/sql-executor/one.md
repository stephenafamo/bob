---

sidebar_position: 3
description: Executes a query and return a single concrete type

---

# One

Execute a query and return a type representing a single row
Similar to `QueryRowContext`, but works directly on a `bob.Query` object.

This function is a wrapper around [`scan.One`](https://pkg.go.dev/github.com/stephenafamo/scan#One).

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
user, err := bob.One(ctx, db, q, scan.StructMapper[userObj]())
if err != nil {
    // ...
}
```
