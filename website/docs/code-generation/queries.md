---
sidebar_position: 7
description: Generating code for SQL queries
---

# Queries

Bob can generate code for SQL queries. This is similar to [sqlc](https://sqlc.dev).

To use this feature, write your SQL queries in `.sql` files, and then point the driver to the folder containing these files using the `queries` configuration option. For example:

```yaml
sqlite:
  dsn: file.db
  queries:
    - ./path/to/folder/containing/sql/files
    - ./another/folder
```

Alongside a few common file, for each `.sql` file found, it will generate two files:

- `file_name.bob.go` - This file contains the generated code for the queries in the SQL file.
- `file_name.bob_test.go` - This file contains the generated test code for the queries in the SQL file.

:::tip

Make sure to run the generated tests. This will ensure that the generated code is correct and that the queries are valid.

:::

## Using the generated code

Given the schema:

```sql
CREATE TABLE users (
    id INT PRIMARY KEY NOT NULL,
    name TEXT
);
```

And the query:

```sql
-- AllUsers
SELECT * FROM users WHERE id = ?;
```

The following code with be generated:

```go
const allUsersSQL = `SELECT "users"."id", "users"."name" FROM "users" WHERE "id" = ?1`

func AllUsers(id int32) orm.ModQuery[*dialect.SelectQuery, AllUsersRow, []AllUsersRow] {
    // ...
}

type AllUsersRow struct {
	ID   int32            `db:"id"`
	Name null.Val[string] `db:"name"`
}
```

:::note

See how `SELECT *` is transformed into `SELECT "users"."id", "users"."name"`. This is done to ensure that the generated code continues to work as expected even if the schema changes.

:::

### Making a query

To make a query, you can use the generated function:

```go
query := AllUsers(1)
```

This will return a `orm.ModQuery` object that you can use to execute the query with any of the expected finishers:

- `One(ctx, db) -> AllUsersRow`
- `All(ctx, db) -> []AllUsersRow`
- `Cursor(ctx, db) -> scan.ICursor[AllUsersRow]`

### Modifying a query

The generated query is a `orm.ModQuery` object, which can also be used as a [`QueryMod`](../query-builder/building-queries#query-mods).

This opens up many use cases, since you can use the generated query as a base and add more mods to it.

```go
// Also filter where name = "Bob"
query := sqlite.Select(
    AllUsers(1),
    psql.Quote("name").EQ(psql.Arg("Bob")),
)
```

## Annotating queries

Each query has the following attributes that can be modified with annotations:

- `query_name`: The name of the query. This is used to generate the function name. **Required**.
- `result_type`: The type of the result. This is used to generate the result type. e.g. `AllUsersRow`.
- `result_type_array`: The type of the result when using `All()`. This is used to generate the result type. e.g. `[]AllUsersRow`.
- `generate_result_type`. If set to `false`, the result type will not be generated. This is useful if you want to replace the result type with a custom type.

Each reuturn column and parameter can also be annotated with the following attributes:

- `name`: The name of the column. This is used to generate the field name.
- `type`: The type of the column. This is used to generate the field type.
- `nullable`: This can be `null` or `notnull` to specify if the column is nullable or not. If it is empty, the nullability will be inferred.

:::tip

Any part of the annotation can be ommited. For example, instead of `name:type:null`, all the following are valid annotations:

- `name`
- `name:type`
- `name::null`
- `:type:null`
- `::null`

The other parts will be inferred from the context.

:::

```sql
-- AllUsers models.User:models.UserSlice:false
SELECT id /* :big.Int:nnull */, name /* username */ FROM users WHERE id = ? /* ::notnull */;
```
