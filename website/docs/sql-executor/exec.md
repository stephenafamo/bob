---

sidebar_position: 2
description: Execute a query without returning any rows.

---

# Exec

Execute a query without returning any rows.  
Similar to `ExecContext`, but works directly on a `bob.Query` object.

```go
ctx := context.Background()
db, err := bob.Open("postgres", "...")
if err != nil {
    // ...
}

q := psql.Update(...)

result, err := bob.Exec(ctx, db, q)
if err != nil {
    // ...
}
```
