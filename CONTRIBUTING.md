# Contributing to Bob

See [Development](README.md#development) for setup, formatting, linting, and tests.

## Add a query-builder example

Bob generates its SQL examples from query-builder tests. A useful contribution is a query mod that has no example yet, or a combination of mods where the expected SQL is not obvious.

1. Choose a dialect in [`dialect/`](dialect/) (`mysql`, `psql`, or `sqlite`). Add a case to the `examples` table in `select_test.go`, `insert_test.go`, `update_test.go`, or `delete_test.go`, depending on the statement.
2. Give the case a descriptive name. Set `Query` to the builder expression, `ExpectedSQL` to the SQL it should produce, and `ExpectedArgs` to any bound values in placeholder order. See [`dialect/sqlite/select_test.go`](dialect/sqlite/select_test.go) for examples.
3. Run the matching test from the repository root. For a SQLite `SELECT` case:

   ```sh
   go test ./dialect/sqlite -run '^TestSelect$'
   ```

4. Regenerate the example pages:

   ```sh
   go run ./website/examples_gen.go
   ```

   Check the new example in `website/docs/query-builder/<dialect>/examples/<statement>.md` and include it with the test in your pull request. Make corrections in the test and regenerate the page.
