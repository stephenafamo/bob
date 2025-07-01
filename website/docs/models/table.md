---
sidebar_position: 3
description: Easily query and modify a database table
---

# Table

:::info

Table models have all the capabilities of [View models](./view). This page will only focus on the additional capabilities.

:::

In addition to **EVERYTHING** a [View model](./view) is capable of, a Table model makes it easy to make changes to a database table.

To create a Table model, use the `NewTable()` function. This takes 2 type parameters:

1. The first type parameter should match the general structure of the table.
2. The second type parameter is used as the "setter". The setter is expected to have "Optional" fields to specify which values are being inserted/updated.

```go
type User struct {
    ID        int     `db:",pk"` // needed to know the primary key when updating
    VehicleID int
    Name      string
    Email     string
}

// UserSetter must implement orm.Setter
type UserSetter struct {
    ID        omit.Val[int]
    VehicleID omit.Val[int]
    Name      omit.Val[string]
    Email     omit.Val[string]
}

var userTable = psql.NewTable[User, UserSetter]("public", "users")
```

:::tip

The `NewTablex()` function takes an extra type parameter to determine how slices are returned.

:::

## Why do we need a setter?

A setter is necessary due to Golang's handling of zero values. As example, when running `userTable.Insert(User{})`, it isn't possible to distinguish which fields were purposefully set by the user vs. fields initialized to their zero values.

Typically, we want to exclude fields that we don't intend to manually set, such as auto increment or generated columns.

```go
userTable.Insert(&UserSetter{
    Name: omit.From("Stephen"), // we want to set name and not the email
}).One(ctx, db)
```

:::tip

The Setter type and methods can also be fully generated from your database structure.
See [Code Generation](../code-generation/intro) for more information.

:::

## Queries

Similar to a [View](./view), the Table model also has a `Query()` method that starts a SELECT query on the table model. It accepts [query mods](../query-builder/building-queries#query-mods) to modify the final query.

In addition, a Table also has `Insert`, `Update` and `Delete` methods which begin insert, update and delete queries on the table. These methods also accept [query mods](../query-builder/building-queries#query-mods) to modify the final query.

```go
// UPDATE "users" SET "kind" = $1 RETURNING *;
updateQ := userTable.Update(
    um.SetCol("kind").ToArg("Dramatic"),
    um.Returning("*"),
)
```

The query can then be executed with the `Exec()` method which returns the rows affected and an error. If the dialect supports the `RETURNING` clause, `One()`, `All()` and `Cursor()` methods are also included.

```go
rowsAffected, _ := updateQ.Exec(ctx, db)
user, _ := updateQ.One(ctx, db)
users, _ := updateQ.All(ctx, db)
userCursor, _ := updateQ.Cursor(ctx, db)
```

## Insert

```go
// INSERT INTO "users" ("id") VALUES (100)
user, err := models.UsersTable.Insert(&UserSetter{
    ID: omit.From(100),
    // add other columns
}).One(ctx, db)
```

## Insert Many

Bulk insert models

```go
// INSERT INTO "users" ("id") VALUES (100), (101), (102)
users, err := models.UsersTable.Insert(
    &UserSetter{ID: omit.From(100)},
    &UserSetter{ID: omit.From(101)},
    &UserSetter{ID: omit.From(102)},
).All(ctx, db)
```

Bulk insert with an existing slice of setters

```go
// INSERT INTO "users" ("id") VALUES (100), (101), (102)
setters := []*UserSetter{
    {ID: omit.From(100)},
    {ID: omit.From(101)},
    {ID: omit.From(102)},
}

users, err := models.UserTable.Insert(bob.ToMods(setters...)).All(ctx, db)
```

## Update

```go
// UPDATE "users"
// SET "vehicle_id" = 200
// WHERE ("users"."id" = 1)
err := user.Update(ctx, db, &UserSetter{VehicleID: omit.From(200)})
```

## Update Many

```go
// UPDATE "users"
// SET "vehicle_id" = 200
// WHERE ("users"."id" IN (1, 2))
err := users.UpdateAll(ctx, db, UserSetter{VehicleID: omit.From(200)})
```

## Upsert

:::info

The method signature for this method varies by dialect.

:::

```go
// PostgreSQL and SQLite
// INSERT INTO "users" ("id", "email") VALUES (1, "bob@foo.bar") ON CONFLICT (id) DO UPDATE SET "email" = EXCLUDED."email"
user, err := models.UsersTable.Insert(
	&UserSetter{
		ID: omit.From(1),
		Email: omit.From("bob@foo.bar"),
	},
	im.OnConflict("id").DoUpdate(im.SetExcluded("email"))).One(ctx, db)

// MySQL
user, err := models.UsersTable.Insert(
    &UserSetter{
        ID: omit.From(1),
        Email: omit.From("bob@foo.bar"),
    },
    im.OnDuplicateKeyUpdate(im.UpdateWithValues("email"))).One(ctx, db)
```

## Delete

```go
// DELETE FROM "users" WHERE "id" = 100
err := user.Delete(ctx, db)
```

## Delete Many

```go
// DELETE FROM "users" WHERE "id" IN (100, 101)
err := users.Delete(ctx, db)
```
