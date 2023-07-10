---

sidebar_position: 1
description: Understanding the Bob Excecutor Types

---

# Bob Executors

While it is enough to use a `bob.Executor` for queries, for a full application, there are other things needed.

* Ability to ping the database for health checks
* Ability to start transactions with `*sql.DB`
* Ability to commit/rollback with `*sql.Tx`
* Ability to prepare and reuse statements

For these purposes, bob has concrete `DB`, `Tx` and `Conn` structs that wrap the `database/sql` equivalents but implement the `bob.Executor` interface too.

See the reference for these types:

## `bob.DB`

[Reference](https://pkg.go.dev/github.com/stephenafamo/bob#DB)

Convert an existing `*sql.DB` to `bob.DB` with `bob.NewDB()`

Open a DB connection and return `bob.DB` with `bob.Open()`. This is the same as running `sql.Open()` and then `bob.NewDB()`.

## `bob.Tx`

[Reference](https://pkg.go.dev/github.com/stephenafamo/bob#Tx)

Convert an existing `*sql.Tx` to `bob.Tx` with `bob.NewTx()`

## `bob.Conn`

[Reference](https://pkg.go.dev/github.com/stephenafamo/bob#Conn)

Convert an existing `*sql.Conn` to `bob.Conn` with `bob.NewConn()`

## Prepared Statements

All of the above executors can be used to create prepared statements too.

[Read More](./prepare)
