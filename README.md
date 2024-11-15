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

Check out [the documentation][docs] for more information.

## Support

|               | Queries | Models | ORM Gen | Factory Gen |
| ------------- | ------- | ------ | ------- | ----------- |
| Postgres      | ✅      | ✅     | ✅      | ✅          |
| MySQL/MariaDB | ✅      | ✅     | ✅      | ✅          |
| SQLite        | ✅      | ✅     | ✅      | ✅          |

## Comparisons

1. [Bob vs GORM](https://bob.stephenafamo.com/vs/gorm)
1. [Bob vs Ent](https://bob.stephenafamo.com/vs/ent)
1. [Bob vs SQLBoiler](https://bob.stephenafamo.com/vs/sqlboiler)
1. [Bob vs Jet](https://bob.stephenafamo.com/vs/jet)

[docs]: https://bob.stephenafamo.com/docs
[reference]: https://pkg.go.dev/github.com/stephenafamo/bob

## Contributing

Thanks to all the people who have contributed to Bob!

[![contributors](https://contributors-img.web.app/image?repo=stephenafamo/bob)](https://github.com/stephenafamo/bob/graphs/contributors)
