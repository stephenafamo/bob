# Bob: Go SQL Access Toolkit

[![Test Status](https://github.com/stephenafamo/bob/actions/workflows/test.yml/badge.svg)](https://github.com/stephenafamo/bob/actions/workflows/test.yml) ![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/stephenafamo/bob) [![Go Reference](https://pkg.go.dev/badge/github.com/stephenafamo/bob.svg)](https://pkg.go.dev/github.com/stephenafamo/bob) [![Go Report Card](https://goreportcard.com/badge/github.com/stephenafamo/bob)](https://goreportcard.com/report/github.com/stephenafamo/bob) ![GitHub tag (latest SemVer)](https://img.shields.io/github/v/tag/stephenafamo/bob) [![Coverage Status](https://coveralls.io/repos/github/stephenafamo/bob/badge.svg)](https://coveralls.io/github/stephenafamo/bob)

## Links

- [Documentation][docs]
- [Reference][reference]

## About

Bob is a set of Go packages and tools to work with SQL databases.

Bob's philosophy centres around the following:

1. **Correctness**: Things should work correctly. Follow specifications as closely as possible.
2. **Convenience** (not magic): Bob provides convenient ways to perform actions, it does not add unexplainable magic, or needless abstraction.
3. **Cooperation**: Bob should work well with other tools and packages as much as possible, especially the standard library.

**Bob** can be progressively adopted from raw SQL query strings, to fully typed queries with models and factories generated for your database.

## Components of Bob

**Bob** consists of several components that build on each other for the full experience.

1. Query Builder
1. SQL Executor for convenient scanning of results
1. Models for convenient database queries
1. Code generation of Models and Factories from your database schema
1. Code generation of Queries similar to [`sqlc`](https://sqlc.dev).

Check out [the documentation][docs] for more information.

## Support

|               | Queries | Models | ORM Gen | Factory Gen | Query Gen |
| ------------- | ------- | ------ | ------- | ----------- | --------- |
| Postgres      | ✅      | ✅     | ✅      | ✅          | ✅        |
| MySQL/MariaDB | ✅      | ✅     | ✅      | ✅          | ✅        |
| SQLite        | ✅      | ✅     | ✅      | ✅          | ✅        |

## Comparisons

1. [Bob vs GORM](https://bob.stephenafamo.com/vs/gorm)
1. [Bob vs Ent](https://bob.stephenafamo.com/vs/ent)
1. [Bob vs SQLBoiler](https://bob.stephenafamo.com/vs/sqlboiler)
1. [Bob vs Jet](https://bob.stephenafamo.com/vs/jet)

[docs]: https://bob.stephenafamo.com/docs
[reference]: https://pkg.go.dev/github.com/stephenafamo/bob

## The layers of Bob

### Layer 1: [The query builder](https://bob.stephenafamo.com/docs/query-builder/intro) - Similar to [squirrel](https://github.com/Masterminds/squirrel)

This is just a fluent query builder that has no concept of your DB, and by extension cannot offer any type-safety.

The main reason, I consider it better than most alternatives is that since each dialect is hand-crafted, it can support building ANY query for that dialect.

However, each dialect is also independent, so you don't have to worry about creating an invalid query.

> **IMPORTANT: Queries are built using "Query Mods"**

```go
psql.Select(
    sm.From("users"), // This is a query mod
    sm.Where(psql.Quote("age").GTE(psql.Arg(21))), // This is also a mod
)
```

### Layer 2: [ORM Code Generation](https://bob.stephenafamo.com/docs/code-generation/intro) - Similar to [SQLBoiler](https://github.com/volatiletech/sqlboiler)

This is where the type safety comes.

A full ORM, and query mods that is based on the database schema. If you use the generated query mods, these will ensure correct type safety.

Here is the above query using generated query-mods.

```go
models.Users.Query(
    models.SelectWhere.Users.Age.GTE(21), // This is type-safe
)
```

### Layer 3: [Factory Code Generation](https://bob.stephenafamo.com/docs/code-generation/factories) - Inspired by [Ruby's FactoryBot](https://github.com/thoughtbot/factory_bot)

Factories make testing much much easier. Especially when the test depends on a database entry that depends on relations in other tables (e.g. testing comments that rely on posts which in turn rely on users).

With knowledge of the database schema, Bob can generate factories for each table.

```go
// Quickly create a 10 comments (posts and users are created appropriately)
comments, err := f.NewComment().CreateMany(ctx, db, 10)
```

### Layer 4: Generating code for [SQL Queries](https://bob.stephenafamo.com/docs/code-generation/queries) - similar to [sqlc](https://github.com/sqlc-dev/sqlc)

I believe this is the final peice of the puzzle, and extends the type-safety to hand-crafted SQL queries.

For example, you could generate code for the query:

```sql
-- UserPosts
SELECT * FROM posts WHERE user_id = $1
```

This will generate a function `UserPosts` that takes an `int32`.

```go
// UserPosts
userPosts, err := queries.UserPosts(1).All(ctx, db)
```

Then, if you need to, you can add an extra filter to get only published posts.

However whether it is type safe or not depends on if you use the generated mods or not:

```go
// Get only published posts
query := psql.Select(
    UserPosts(1),
    models.PostWhere.Status.EQ("published"), // type-safe
    sm.Where(psql.Quote("posts", "status").Eq(psql.Arg("published"))), // not type-safe
)
```

## Development

### Nix

You can get all the tools you need for developing against this repository with [nix](https://nixos.org/). Use `nix-shell` in the root of the repository to get a shell with all the dependencies and tools for development.

### Lint

This repository uses [golangci-lint](https://github.com/golangci/golangci-lint) for linting. You can run the linter with:

```
$ golangci-lint run
```

Before submitting pull requests you should ensure that your changes lint clean:

```
$ golangci-lint run
0 issues.
```

### Formatting

This repository uses [gofumpt](https://github.com/mvdan/gofumpt) for formatting. It's more strict than `go fmt`. The linter will fail if you haven't formatted for code correctly. You can format your code with:

```
$ gofumpt -l -w ./some/file
```

### Test

You can test this repository using [go test](https://pkg.go.dev/testing). A simple test of a single module can be run with:

```
$ go test ./dialect/psql/
```

A comprehensive test of all modules could be done with:

```
$ go test ./...
```

If you're interested in data [race detection](https://go.dev/doc/articles/race_detector) and [coverage](https://go.dev/blog/integration-test-coverage), both of which are done by the Github workflow prior to merging code, you'll need to add a few more arguments:

```
$ go test -race -covermode atomic --coverprofile=covprofile.out -coverpkg=github.com/stephenafamo/bob/... ./...
```

### Workflows

The project uses [Github Actions](https://docs.github.com/en/actions) as defined in [.github/workflows](.github/workflows). Pull requests are expected to pass linting and testing workflows before being accepted.

## Contributing

Thanks to all the people who have contributed to Bob!

[![contributors](https://contributors-img.web.app/image?repo=stephenafamo/bob)](https://github.com/stephenafamo/bob/graphs/contributors)
