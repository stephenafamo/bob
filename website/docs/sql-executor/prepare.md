---

sidebar_position: 9
description: Creates a prepared statement for later queries or executions.

---

# Prepare

## Exec Statements

A statement is created with `bob.Prepare()`.

```go
ctx := context.Background()
db, err := bob.Open("postgres", "...")
if err != nil {
    // ...
}

q := psql.Update(...)

// Prepare the statement
stmt, err := bob.Prepare(ctx, db, q)
if err != nil {
    // ...
}
```

Prepared statements can then be reused as many times as we want.

```go
// Use our prepared statement
_, err := stmt.Exec(ctx)
if err != nil {
    // ...
}
```

## Query Statements

Statements that are expected to return rows, are instead prepared with `bob.PrepareQuery` or `bob.PrepareQueryx`.

In addition to `Exec`, this statement also has `One`, `All` and `Cursor` methods.

```go
q := psql.Select(...)

// Prepare the statement
stmt, err := bob.PrepareQuery(ctx, db, q, scan.StructMapper[userObj]())
if err != nil {
    // ...
}

// Use our prepared statement
users, err := stmt.All(ctx)
if err != nil {
    // ...
}
```

