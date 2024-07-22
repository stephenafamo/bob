---

sidebar_position: 4
description: Manipulate column listes

---

# Columns

The [`orm.Columns`](https://pkg.go.dev/github.com/stephenafamo/bob/orm#Columns) type is a bob [expression](../query-builder/building-queries#expressions).

To create a new columns list, use `orm.NewColumns(names ...string)`. You can then add the parent table with [WithParent()](#withparent)

```go
orm.NewColumns("id", "name", "email").WithParent("public.users")
```

```sql
-- table_alias.column_name
"public.users"."id" AS "id",
"public.users"."name" AS "name",
"public.users"."email" AS "email"
```

It has many convenient methods to manipulate the columns list.

## `Only`

Returns the column list with ONLY the given column names.

```go
userView.Columns().Only("email")
```

```sql
"public.users"."email" AS "email",
```

## `Except`

Returns the columns list without the given column names.

```go
userView.Columns().Except("email")
```

```sql
"public.users"."id" AS "id",
"public.users"."name" AS "name",
```

## `WithParent`

Changes the parent of the column list. For example if selecting with an alias.  
Multiple strings are quoted and joined with a dot.

```go
userView.Columns().WithParent("backup", "users_old")
```

```sql
"backup"."users_old"."id" AS "id",
"backup"."users_old"."name" AS "name",
"backup"."users_old"."email" AS "email"
```

## `WithPrefix`

Sets a prefix for all columns, useful for joins with duplicate column names.

```go
userView.Columns().WithPrefix("users.")
```

```sql
"public.users"."id" AS "users.id",
"public.users"."name" AS "users.name",
"public.users"."email" AS "users.email"
```

