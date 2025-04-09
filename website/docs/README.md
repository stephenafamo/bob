---
sidebar_position: 1
---

import DocCardList from '@theme/DocCardList';

# Introduction

## Support

|               | Queries | Models | ORM Gen | Factory Gen | Query Gen |
| ------------- | ------- | ------ | ------- | ----------- | --------- |
| Postgres      | ✅      | ✅     | ✅      | ✅          |           |
| MySQL/MariaDB | ✅      | ✅     | ✅      | ✅          |           |
| SQLite        | ✅      | ✅     | ✅      | ✅          | ✅        |

Bob is a set of Go packages and tools to work with SQL databases.

Bob's philosophy centers around the following:

1. **Correctness**: Things should work correctly. Follow specifications as closely as possible.
2. **Convenience** (not magic): Bob provides convenient ways to perform actions, it does not add unexplainable magic, or needless abstraction.
3. **Cooperation**: Bob should work will with other tools and packages as much as possible, especially the standard library.

**Bob** consists of several tools that build on each other for the full experience.

**Bob** can be progressively adopted from raw SQL query strings, to fully typed queries with models and factories generated for your database.

## 1. Query Builder

Bob can be used to build queries, this is similar to other packages like [squirrel](https://github.com/Masterminds/squirrel) and [goqu](https://github.com/doug-martin/goqu)

However, Bob strives to be fully spec compliant. As a result, you can build **almost any** query permitted by the dialect. And because the query builders are custom crafted for each dialect. You are almost unable to build an invalid query.

To learn more, see the [query builder documentation](./query-builder/intro).

## 2. SQL Executor

Bob includes an SQL executor that conveniently returns types from queries to avoid doing `rows.Scan()` over and over.

Bob's executor can build and execute Bob queries in a single call.

- `One()`: To scan a single row
- `All()`: To scan all rows
- `Cursor()`: To loop through rows. Useful for large results
- `Prepare()`: For prepared statements

In addition, the executor covers the usual range of DB activities:

- Ping the database for health checks
- Start and run transactions
- Commit/Rollback
- Prepare and reuse statements

To learn more, see the [sql executor documentation](./sql-executor/intro).

## 3. Models

Get a model to easily query your entity tables.

**NewView()**: Returns a view (which is read-only table).

```go
type User struct {
    ID    int
    Name  string
    Email string
}

// Views includes methods for Selects and cannot be used to insert/update/delete
var userView = psql.NewView[User]("public", "users")
```

**NewTable()**: This take an extra type that is used as the "setter". The setter is expected to have "Optional" fields used to know which values are being inserted/updated.

```go
type UserSetter struct {
    ID    omit.Val[int]
    Name  omit.Val[string]
    Email omit.Val[string]
}

// Includes methods for Selects, Inserts, Upserts, and Deletes
var userTable = psql.NewTable[User, UserSetter]("public", "users")
```

To learn more about the methods attached to views and tables, see the [models documentation](./models/intro).

## 4. Code Generation

**Bob** includes code generators that will read your database structure and generate a fully featured and type safe ORM for you.

- Work with existing databases. Don't be the tool to define the schema, that's better left to other tools.
- Eliminate all sql boilerplate, have relationships as a first-class concept.
- Work with normal structs, call functions, no hyper-magical struct tags, small interfaces.
- The models package is type safe. This means no chance of random panics due to passing in the wrong type. No need for `interface{}`.
- The generated types closely correlate to your database column types.
- IDE auto-completion due to generated types and functions.
- Clean code that is easy to read and debug.

### Model Generation

Types are generated for your database tables to use [Bob's models](./models/intro), in **addition** to that, the following are also generated:

- Additional convenience methods on the model structs for CRUD.
- A dedicated type for collections.
- Type safe variables for table and column names.
- Type safe representation of ENUMS.
- Query mods used for loading and querying relationships.
  - Relationships are automatically detected by foreign keys.
  - Relationships can also be manually defined.
- Convenient methods for querying a collection of models.

Some other features not found in many other Go ORMs.

- Fine-tuned control over relationship loading.
  - Related-Through: Define relationships that cut across multiple tables
  - Related-When: Define relations based on static values (e.g. `WHEN email_confirmed = true`)
  - Loading relationships with left-joins
  - Loading with additional queries
  - Recursive relationship loading
  - Select **EXACTLY** what columns you want to load.
- Support for cross-schema relationships
- Support for multi-column foreign key relationships
- Support for database views

### Factory Generator

When working with databases, it is often difficult to repeatedly generate mocked data for testing.

Some tools either focus on adding data to the database, other tools focus on randomizing struct fields.

When working with SQL databases,
this can become frustrating because to truly test inserting a row in a table,
we often have to insert other related data.
Without dedicated tools for this, we end up never properly testing our data layer.

Inspired by Ruby's factoryBot, **Bob** uses its knowledge of your database schema to
generate factories that offer many benefits:

- Set base rules for how new objects are created, defaults are preset for required columns.
- Create templates from the rules, overriding specific columns as needed.
- Build models based on the model templates and use them in tests.
- Insert the model in the database to truly test your application. Bob will help you by inserting any dependent models.

<DocCardList items={[
{
type: 'link',
label: 'Query Builder',
href: '/docs/query-builder/intro',
docId: 'query-builder/intro',
autoAddBaseUrl: true,
},
{
type: 'link',
label: 'SQL Executor',
href: '/docs/sql-executor/intro',
docId: 'sql-executor/intro',
autoAddBaseUrl: true,
},
{
type: 'link',
label: 'Models',
href: '/docs/models/intro',
docId: 'models/intro',
autoAddBaseUrl: true,
},
{
type: 'link',
label: 'Code Generation',
href: '/docs/code-generation/intro',
docId: 'code-generation/intro',
autoAddBaseUrl: true,
},
]} />
