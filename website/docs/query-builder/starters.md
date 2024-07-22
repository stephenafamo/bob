---
sidebar_position: 4.5
description: Common starters accross dialects
---

# Starters

There are a number of common starter functions shared by all supported dialects:

- `S(string)`: Create a single quoted string literal.

  ```go
  //	SQL: 'a string'
  psql.S("a string")
  ```

- `F(name string, args ...any)`: A function call. Takes a name and the arguments.

  ```go
  //	SQL: generate_series(1, 3)
  psql.F("generate_series", 1, 3)
  ```

- `Not(Expression)`: Creates a `NOT expr` expression.

  ```go
  //	SQL: Not a = b
  psql.Not("a = b")
  ```

- `Or(...Expression)`: Joins multiple expressions with "OR".

  ```go
  // SQL: a OR b OR c
  psql.Or("a", "b", "c")
  ```

- `And(...Expression)`: Joins multiple expressions with "AND"

  ```go
  // SQL: a AND b AND c
  psql.And("a", "b", "c")
  ```

- `Arg(...any)`: One or more arguments. These are replaced with placeholders in the query and the args returned.

  ```go
  // SQL: $1, $2, $3
  // Args: 'a', 'b', 'c'
  psql.Arg("a", "b", "c")
  ```

- `ArgGroup(...any)`: Similar to `Arg` but wraps the given set of arguments in parentheses.

  ```go
  // SQL: ($1, $2), ($3, $4)
  // Args: ('a', 'b', 'c', 'd')
  psql.Group(psql.ArgGroup("a", "b"), psql.ArgGroup("c", "d"))
  ```

- `Placeholders(uint)`: Inserts a `count` of placeholders without any specific value yet. Useful for compiling reusable queries.

  ```go
  // SQL: $1, $2, $3
  // Args: nil, nil, nil
  psql.Placeholders(3)
  ```

- `Group(...Expression)`: To easily group a number of expressions. Wraps them in parentheses **AND** separates them with commas.

  ```go
  // SQL: (a, b, c)
  psql.Group("a", "b", "c")
  ```

- `Quote(...string)`: For quoting. [See details](./quotes)

  ```go
  // SQL: "table"."column
  psql.Quote("table", "column")
  ```

- `Raw(clause string, args ...any)`: For inserting a raw statement somewhere. To keep it dialect agnostic, placeholders should be inserted with `?` and a literal question mark can be escaped with a backslash `\?`.

  ```go
  // SQL: WHERE a = $1
  // Args: 'something'
  psql.Raw("WHERE a = ?", "something")
  ```

- `As(e Expression, alias string)`: For aliasing expressions.

  ```go
  // SQL: pilots as "p"
  psql.As("pilots", "p")
  ```

See dialect documentation for extra starters
