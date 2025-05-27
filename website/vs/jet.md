---
sidebar_position: 4
---

# Bob vs Jet

Bob and Jet share similar principles and lead to a similar development experience. The main difference comes because Jet is purposefully **NOT AN ORM**.
In practice this means the following:

## Querying and Mapping

Like Bob, Jet generates models for your tables, however, since Jet is only query builder, every query has to be built and mapped manually.

For example, here is how we would retrieve a model by primary key:

### Bob: Get by primary key

```go
user, err := models.FindUser(ctx, db, 1)
```

### Jet: Get by primary key

```go
var user *model.Users
err = postgres.
    SELECT(table.Users.AllColumns).
    FROM(table.Users).
    WHERE(table.Users.ID.EQ(postgres.Int(1))).
    QueryContext(ctx, db, user)
```

## Relationships

Because Jet does not aim to be an ORM, it does not provides an easy way to work with relationships:

### Bob: Retrieve with relations

```go
// User will contain the videos
user, err := models.Users(
    models.SelectWhere.Users.ID.EQ(1),
    models.SelectThenLoad.User.Videos(),
).One(ctx, db)
```

### Jet: Retrieve with relations

```go
var dest struct {
    model.Users
    Videos []model.Videos
}
err = postgres.
    SELECT(
        table.Users.AllColumns,
        table.Videos.AllColumns,
    ).
    FROM(
        table.Users.
            INNER_JOIN(table.Videos, table.Users.ID.EQ(table.Videos.UserID)),
    ).
    WHERE(table.Users.ID.EQ(postgres.Int(1))).
    QueryContext(ctx, db, &dest)
```

## Factory

In addition to the models, Bob also generates factories to help with testing. [See Documentation](../docs/code-generation/factories)

## Summary

While the query building experience is similar, since Jet does not aim to be an ORM, Jet skips some features that make day-to-day use less verbose.
