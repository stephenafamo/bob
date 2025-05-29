---
sidebar_position: 10
description: Why Bob uses a custom executor interface
---

# Why not `*sql.DB`?

By default, `*sql.DB` does not implement the `bob.Executor` interface.

This is because the `QueryContext()` method of `*sql.DB` return an `*sql.Rows` object which is very difficult to mock or implement.

To be able to interoperate with other libraries and for ease of testing/mocking, **Bob**'s Executor instead return a [`scan.Rows`](https://pkg.go.dev/github.com/stephenafamo/scan#Rows) interface.

```go
// Rows is an interface that is expected to be returned as the result of a query
type Rows interface {
	Scan(...any) error
	Columns() ([]string, error)
	Next() bool
	Close() error
	Err() error
}
```

It is easy to convert an `*sql.{DB,Tx,Conn}` to a `bob.Executor` with `bob.New()`

```go
db, err := sql.Open("postgres", "postgres://...")
bobExec := bob.NewDB(db)

// For Transactions
tx, err := db.Begin()
bobExec = bob.NewTx(tx) // using the transaction
```
