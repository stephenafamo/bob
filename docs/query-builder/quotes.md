---

sidebar_position: 3
description: How to use quotes with Bob

---

# Quotes

It is often required to quote identifiers in SQL queries. With `bob` use the `sm.Quote()` where necessary. When building the query, the quotes are added correctly by the dialect.

It can take multiple strings that need to be quoted and joined with `.`

```go
// Postgres: "schema_name"."table_name"
// SQLite: "schema_name"."table_name"
// MySQL: `schema_name`.`table_name`
// SQL Server: [schema_name].[table_name]
psql.Quote("schema_name", "table_name")
```

