# bobcheck

Bob runs `bobcheck` automatically after writing all generated files. No separate
installation is required. It checks `./...` from the root of the Go module it was started
in, even when started from a subdirectory. This includes application code,
generated code, and tests for the current build configuration. A diagnostic
or package-loading error makes the command fail after writing the files.
If Go reports missing module requirements or `go.sum` entries, run
`go mod tidy` from the module root, then rerun the generator.

To skip the check, set this top-level option in `bobgen.yaml`:

```yaml
disable_bobcheck: true
```

To run the check independently from the module root:

```sh
go run github.com/stephenafamo/bob/cmd/bobcheck ./...
```

`bobcheck` reports direct method calls to `WithMods` that discard a single
result implementing `bob.Query`, including calls to promoted methods on
generated query wrappers. For example, if `WithMods` returns a new query:

```go
q.WithMods(filter)          // Reported: q still has its old value.
q = q.WithMods(filter)      // OK: replace q.
page := q.WithMods(mods...) // OK: derive another query.
```

All dialects and `bob.QueryBuilder` return queries from `WithMods`. Calling
`mod.Edit(editor)` returns nothing and is not reported.
An explicit discard (`_ = q.WithMods(filter)`) is also accepted.

In a Bob checkout, use `go run ./cmd/bobcheck ./...`. The command can also
run as a vet tool (`go vet -vettool=/path/to/bobcheck ./...`).

## Runtime

On an Apple M4 with Go 1.27.3, a five-table SQLite project (5 packages,
39 generated Go files) took **0.72 s** to regenerate with the check off and
**1.07 s** with it on. These are median wall times from six runs of each
configuration with a warm Go build cache and a prebuilt generator.

The prebuilt checker alone took **0.38 s** on that module and **1.67 s** on
Bob's 70-package checkout (warm medians, five runs each). The same SQLite
check took **26 s** with an empty Go build cache and dependencies already
downloaded. To measure your module, build `bobcheck` once and run
`time bobcheck ./...` from the module root.
