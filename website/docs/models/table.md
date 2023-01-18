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
    ID        int     `db:",pk"` // needed to know the primary key when updating
    VehicleID int
    Name      string
    Email     string
}

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

A setter is necessary because if we run `userTable.Insert(User{})`, due to Go's zero values it will be difficult to know which fields we purposefully set.

Typically, we can leave out fields that we never intend to manually set, such as auto increment or generated columns.

```go
userTable.Insert(ctx, db, UserSetter{
    Name: omit.From("Stephen"), // we know the name was set and not the email
}) 
```

## Queries

The `Query()` method returns a `TableQuery` on the model's database table.

In addition to the methods provided by `ViewQuery` -- `One()`, `All()`, `Cursor()`, `Count()`, `Exists()`,  
`TableQuery` also has `UpdateAll()` and `DeleteAll()` which works on all the columns matching the current query.

```go
// DELETE FROM "users" WHERE id IN (SELECT "users"."id" FROM "users" LIMIT 10)
models.Users(ctx, db, sm.Limit(10)).DeleteAll()

// UPDATE "users" SET "vehicle_id" = 100 WHERE id IN (SELECT "users"."id" FROM "users" LIMIT 10)
models.Users(ctx, db, sm.Limit(10)).UpdateAll(&UserSetter{VehicleID: omit.From(100)})
```

## Insert

```go
// INSERT INTO "users" ("id") VALUES (100)
user, err := models.UsersTable.Insert(ctx, db, &UserSetter{
    ID: omit.From(100),
    // add other columns
})
```

## InsertMany

Bulk insert models

```go
// INSERT INTO "users" ("id") VALUES (100), (101), (102)
users, err := models.UsersTable.InsertMany(ctx, db,
    &UserSetter{ID: omit.From(100)},
    &UserSetter{ID: omit.From(101)},
    &UserSetter{ID: omit.From(102)},
)
```

## Update

```go
_, err := models.UsersTable.Update(ctx, db, user)
```

## UpdateMany

UpdateMany uses a `UserSetter` to determine which columns to set to and the desired value

```go
// UPDATE "users"
// SET "vehicle_id" = 200
// WHERE "users"."id" IN (10, 11, 12)
_, err := models.UsersTable.UpdateMany(ctx, db, &UserSetter{
    &UserSetter{VehicleID: omit.From(200)},
}, user10, user11, user12)
```

## Upsert

:::info

The method signature for this varies by dialect.

:::

```go
// INSERT INTO "users" ("id") VALUES (100) ON CONFLICT DO UPDATE SET "id" = EXCLUDED."id"
user, err := models.UsersTable.Upsert(ctx, db, true, nil, nil, &UserSetter{
    ID: omit.From(100),
    // add other columns
})
```

## Delete

```go
// DELETE FROM "users" WHERE "id" = 100
_, err := models.UsersTable.Delete(ctx, db, user)
```
