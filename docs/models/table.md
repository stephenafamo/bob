---

sidebar_position: 3
description: Easily query and modify a database table

---

# Table

:::info

Table models have all the capabilities of [view models](./view). This page will only focus on the additional capabilities.

:::

In addition to **EVERYTHING** a [view model](./view) is capable of, a table model makes it easy to make changes to a database table.

To create a view, use the `NewTable()` function. This takes 2 type parameters:

1. The first should match the general structure of the table.
2. The second is used as the "setter". The setter is expected to have "Optional" fields used to know which values are being inserted/updated.

```go
type User struct {
    ID    int     `db:",pk"` // needed to know the primary key when updating
    Name  string
    Email string
}

type UserSetter struct {
    ID    omit.Val[int]
    Name  omit.Val[string]
    Email omit.Val[string]
}

var userTable = psql.NewTable[User, UserSetter]("public", "users")
```

:::tip

The `NewTablex()` function takes an extra type parameter to determine how slices are returned.

:::

## Why do we need a setter?

A setter is necessary because if we run `userTable.Insert(User{})`, due to Go's zero values it will be difficult to know which fields we purposefully set.

Typically, we can leave out fields that we never intend to manually set, such as auto increment or generated columns.

```go
userTable.Insert(ctx, db, UserSetter{
    Name: omit.From("Stephen"), // we know the name was set and not the email
}) 
```

## Insert()

Inserts a single row into the table.

## InsertMany()

Bulk inserts multiple rows into the table.

## Upsert()

Inserts a single row into the table and handle conflicts as desired.

## UpsertMany()

Bulk inserts a single row into the table and handle conflicts as desired.

## Update()

Syncs the current values in the model into the expected row in the database (by primary key).

## UpdateMany()

Updates all the given models with the values in the provided setter.

## Delete()

Deletes a single row from the table.

## DeleteMany()

Bulk delete multiple rows from the table.

## Query()

The `Query()` method returns a `TableQuery` on the model's database table.

In addition to the methods provided by `ViewQuery` -- `One()`, `All()`, `Cursor()`, `Count()`, `Exists()`,  
`TableQuery` also has `UpdateAll()` and `DeleteAll()` which works on all the columns matching the current query.
