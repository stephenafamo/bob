# Using Batch Operations with Bob Generated Models

This document demonstrates how to use pgx batch operations with Bob's auto-generated ORM models.

## Overview

Bob's code generation creates table models with Insert, Update, Delete, and Query methods. These integrate seamlessly with the batch API to enable efficient bulk operations with type safety.

## Generated Code Structure

For a `users` table, Bob generates:

```go
// From 001_types.go.tpl
type User struct {
    ID    int
    Name  string
    Email string
}

type UserSlice []*User

// Table variable - the entry point for queries
var Users = psql.NewTablex[*User, UserSlice, *UserSetter](...)

// From 002_setter.go.tpl
type UserSetter struct {
    ID    omit.Val[int]
    Name  omit.Val[string]
    Email omit.Val[string]
}

// Setter has an Apply method that works with InsertQuery
func (s *UserSetter) Apply(q *dialect.InsertQuery)

// From 004_query.go.tpl
func FindUser(ctx context.Context, exec bob.Executor, idPK int, cols ...string) (*User, error)

// From 005_one_methods.go.tpl
func (o *User) Update(ctx context.Context, exec bob.Executor, s *UserSetter) error
func (o *User) Delete(ctx context.Context, exec bob.Executor) error
func (o *User) Reload(ctx context.Context, exec bob.Executor) error

// From 007_slice_methods.go.tpl
func (o UserSlice) UpdateAll(ctx context.Context, exec bob.Executor, vals UserSetter) error
func (o UserSlice) DeleteAll(ctx context.Context, exec bob.Executor) error
func (o UserSlice) ReloadAll(ctx context.Context, exec bob.Executor) error
```

## Basic Insert with Batch

### Pattern 1: BatchBuilder (Sequential Processing)

```go
import (
    "context"
    "github.com/stephenafamo/bob/drivers/pgx"
    "github.com/stephenafamo/bob/orm/omit"
    "your/project/models"
)

func insertUsers(ctx context.Context, db *pgx.Pool) error {
    batch := pgx.NewBatchBuilder()

    // Add multiple inserts using the generated table
    names := []string{"Alice", "Bob", "Charlie"}
    for _, name := range names {
        insertQuery := models.Users.Insert(&models.UserSetter{
            Name: omit.From(name),
        })
        batch.AddQuery(insertQuery)
    }

    // Execute batch - NO TRANSACTION WRAPPER!
    // This is a single round trip via SendBatch
    results := batch.Execute(ctx, db)
    defer results.Close()

    // Process results
    for i := 0; i < batch.Len(); i++ {
        res, err := results.Exec()
        if err != nil {
            return fmt.Errorf("insert %d failed: %w", i, err)
        }
        rows, _ := res.RowsAffected()
        fmt.Printf("Inserted %d rows\n", rows)
    }

    return nil
}
```

### Pattern 2: QueuedBatch with RETURNING (Recommended)

```go
import (
    "github.com/stephenafamo/bob/drivers/pgx"
    "github.com/stephenafamo/scan"
    "your/project/models"
)

func insertUsersReturning(ctx context.Context, db *pgx.Pool) ([]*models.User, error) {
    qb := pgx.NewQueuedBatch()

    // Result storage
    var users []*models.User

    // Build insert query with RETURNING
    names := []string{"Alice", "Bob", "Charlie"}
    for _, name := range names {
        insertQuery := models.Users.Insert(
            &models.UserSetter{Name: omit.From(name)},
            im.Returning("*"),
        )

        // Queue the insert - result will be populated during Execute
        var user models.User
        err := pgx.QueueInsertRowReturning(qb, ctx, insertQuery,
            scan.StructMapper[models.User](), &user)
        if err != nil {
            return nil, err
        }
        users = append(users, &user)
    }

    // Execute - single SendBatch round trip!
    if err := qb.Execute(ctx, db); err != nil {
        return nil, err
    }

    // All users are now populated with their IDs from the database
    return users, nil
}
```

## Bulk Insert with Slice

Bob's `Insert` method accepts multiple setters for bulk insert:

```go
func bulkInsertUsers(ctx context.Context, db *pgx.Pool) error {
    qb := pgx.NewQueuedBatch()

    var users models.UserSlice

    // Create bulk insert with multiple values
    insertQuery := models.Users.Insert(
        &models.UserSetter{Name: omit.From("Alice")},
        &models.UserSetter{Name: omit.From("Bob")},
        &models.UserSetter{Name: omit.From("Charlie")},
        im.Returning("*"),
    )

    // Queue to get all inserted users back
    pgx.QueueInsertReturning(qb, ctx, insertQuery,
        scan.StructMapper[*models.User](), &users)

    if err := qb.Execute(ctx, db); err != nil {
        return err
    }

    fmt.Printf("Inserted %d users\n", len(users))
    return nil
}
```

## Mixed Operations with Generated Models

```go
func mixedOperations(ctx context.Context, db *pgx.Pool) error {
    qb := pgx.NewQueuedBatch()

    // 1. Insert a new user
    var newUser models.User
    insertQ := models.Users.Insert(
        &models.UserSetter{Name: omit.From("David")},
        im.Returning("*"),
    )
    pgx.QueueInsertRowReturning(qb, ctx, insertQ,
        scan.StructMapper[models.User](), &newUser)

    // 2. Update existing users
    updateQ := models.Users.Update(
        um.Set("active", true),
        um.Where(models.Users.Columns.Name.EQ(psql.Arg("Alice"))),
    )
    pgx.QueueExecRow(qb, ctx, updateQ) // Ensures exactly 1 row updated

    // 3. Query users
    var activeUsers models.UserSlice
    selectQ := models.Users.Query(
        sm.Where(models.Users.Columns.Active.EQ(psql.Arg(true))),
    )
    pgx.QueueSelectAll(qb, ctx, selectQ,
        scan.StructMapper[*models.User](), &activeUsers)

    // Execute all in single batch
    if err := qb.Execute(ctx, db); err != nil {
        return err
    }

    fmt.Printf("New user: %+v\n", newUser)
    fmt.Printf("Active users: %d\n", len(activeUsers))
    return nil
}
```

## Using Generated Methods in Batch

While individual model methods (Update, Delete) execute immediately, you can build equivalent queries for batching:

### Individual Update (not batched)
```go
user := &models.User{ID: 1, Name: "Alice"}
err := user.Update(ctx, db, &models.UserSetter{
    Email: omit.From("alice@example.com"),
})
```

### Batch Equivalent
```go
qb := pgx.NewQueuedBatch()

// Build update query for specific user
updateQ := models.Users.Update(
    um.Set("email", "alice@example.com"),
    um.Where(models.Users.Columns.ID.EQ(psql.Arg(1))),
)
pgx.QueueExecRow(qb, ctx, updateQ)

// Add more updates to batch
updateQ2 := models.Users.Update(
    um.Set("email", "bob@example.com"),
    um.Where(models.Users.Columns.ID.EQ(psql.Arg(2))),
)
pgx.QueueExecRow(qb, ctx, updateQ2)

// Execute all updates in one round trip
qb.Execute(ctx, db)
```

## Batch Updates with RETURNING

```go
func batchUpdateReturning(ctx context.Context, db *pgx.Pool, userIDs []int) (models.UserSlice, error) {
    qb := pgx.NewQueuedBatch()

    var updatedUsers models.UserSlice

    for _, id := range userIDs {
        updateQ := models.Users.Update(
            um.Set("last_login", "NOW()"),
            um.Where(models.Users.Columns.ID.EQ(psql.Arg(id))),
            um.Returning("*"),
        )

        var user models.User
        err := pgx.QueueUpdateRowReturning(qb, ctx, updateQ,
            scan.StructMapper[models.User](), &user)
        if err != nil {
            return nil, err
        }
        updatedUsers = append(updatedUsers, &user)
    }

    if err := qb.Execute(ctx, db); err != nil {
        return nil, err
    }

    return updatedUsers, nil
}
```

## Batch Deletes

```go
func batchDelete(ctx context.Context, db *pgx.Pool, userIDs []int) error {
    qb := pgx.NewQueuedBatch()

    for _, id := range userIDs {
        deleteQ := models.Users.Delete(
            dm.Where(models.Users.Columns.ID.EQ(psql.Arg(id))),
        )
        pgx.QueueExecRow(qb, ctx, deleteQ) // Validates 1 row deleted
    }

    return qb.Execute(ctx, db)
}
```

## Type-Safe Column References

Use generated column helpers for type-safe queries:

```go
func typeSafeQuery(ctx context.Context, db *pgx.Pool) error {
    qb := pgx.NewQueuedBatch()

    // Generated columns provide type-safe references
    cols := models.Users.Columns

    var users models.UserSlice
    selectQ := models.Users.Query(
        sm.Columns(cols.ID, cols.Name, cols.Email),
        sm.Where(cols.Active.EQ(psql.Arg(true))),
        sm.Where(cols.CreatedAt.GT(psql.Arg(time.Now().AddDate(0, -1, 0)))),
        sm.OrderBy(cols.Name).Asc(),
    )

    pgx.QueueSelectAll(qb, ctx, selectQ,
        scan.StructMapper[*models.User](), &users)

    return qb.Execute(ctx, db)
}
```

## Working with Relationships

If you've configured relationships in Bob's code generation:

```go
func batchWithRelationships(ctx context.Context, db *pgx.Pool) error {
    qb := pgx.NewQueuedBatch()

    // Query users with their posts (assuming relationship defined)
    var users models.UserSlice
    userQ := models.Users.Query(
        sm.Where(models.Users.Columns.Active.EQ(psql.Arg(true))),
    )
    pgx.QueueSelectAll(qb, ctx, userQ,
        scan.StructMapper[*models.User](), &users)

    if err := qb.Execute(ctx, db); err != nil {
        return err
    }

    // Now load relationships for all users (separate query, not batched)
    if err := orm.Load(ctx, db, users, "Posts"); err != nil {
        return err
    }

    return nil
}
```

## Complete Example: User Registration Flow

```go
package main

import (
    "context"
    "fmt"
    "github.com/stephenafamo/bob/drivers/pgx"
    "github.com/stephenafamo/bob/orm/omit"
    "github.com/stephenafamo/scan"
    "your/project/models"
)

type RegistrationRequest struct {
    Name  string
    Email string
}

func registerUsers(ctx context.Context, db *pgx.Pool,
    requests []RegistrationRequest) (models.UserSlice, error) {

    qb := pgx.NewQueuedBatch()
    var users models.UserSlice

    for _, req := range requests {
        // Validate email doesn't exist
        var exists bool
        checkQ := models.Users.Query(
            sm.Columns("1"),
            sm.Where(models.Users.Columns.Email.EQ(psql.Arg(req.Email))),
        )
        // Note: Exists() would execute immediately, so we build a SELECT instead
        var count int
        pgx.QueueSelectRow(qb, ctx, checkQ, scan.SingleColumnMapper[int](&count), &count)

        // Insert user
        var user models.User
        insertQ := models.Users.Insert(
            &models.UserSetter{
                Name:  omit.From(req.Name),
                Email: omit.From(req.Email),
            },
            im.Returning("*"),
        )
        pgx.QueueInsertRowReturning(qb, ctx, insertQ,
            scan.StructMapper[models.User](), &user)

        users = append(users, &user)
    }

    // Execute all queries in single batch
    if err := qb.Execute(ctx, db); err != nil {
        return nil, err
    }

    return users, nil
}
```

## Error Handling Patterns

```go
func robustBatchInsert(ctx context.Context, db *pgx.Pool,
    names []string) error {

    qb := pgx.NewQueuedBatch()

    for _, name := range names {
        insertQ := models.Users.Insert(
            &models.UserSetter{Name: omit.From(name)},
        )
        if err := pgx.QueueExec(qb, ctx, insertQ); err != nil {
            return fmt.Errorf("failed to queue insert for %s: %w", name, err)
        }
    }

    // Execute returns first error encountered
    if err := qb.Execute(ctx, db); err != nil {
        return fmt.Errorf("batch execution failed: %w", err)
    }

    return nil
}
```

## Performance Comparison

### Without Batch (N Round Trips)
```go
// BAD - Each insert is a separate round trip
for _, name := range []string{"Alice", "Bob", "Charlie"} {
    _, err := models.Users.Insert(&models.UserSetter{
        Name: omit.From(name),
    }).One(ctx, db)
    if err != nil {
        return err
    }
}
// Total: 3 round trips
```

### With Batch (1 Round Trip)
```go
// GOOD - Single SendBatch call
qb := pgx.NewQueuedBatch()
var users models.UserSlice

for _, name := range []string{"Alice", "Bob", "Charlie"} {
    var user models.User
    insertQ := models.Users.Insert(
        &models.UserSetter{Name: omit.From(name)},
        im.Returning("*"),
    )
    pgx.QueueInsertRowReturning(qb, ctx, insertQ,
        scan.StructMapper[models.User](), &user)
    users = append(users, &user)
}

qb.Execute(ctx, db)
// Total: 1 round trip - 3x faster!
```

## When to Use Batch Operations

### Good Use Cases
- Bulk inserts of multiple records
- Updating multiple records individually
- Mixed operations that don't depend on each other
- Importing data from external sources
- Processing queued jobs

### Bad Use Cases
- Operations that depend on previous results
- Very large datasets (consider COPY instead)
- Single operations (just use normal queries)

## Limitations

1. **Results must be processed in order** - You can't skip to the 3rd result
2. **No inter-query dependencies** - Query 2 can't use results from Query 1
3. **PostgreSQL/pgx specific** - Not portable to MySQL/SQLite
4. **No automatic transactions** - Batch provides atomicity but wrap in tx if you need rollback on partial failure

## Transactions vs Batches

### When You Need a Transaction
```go
// Multiple batches + other operations that must be atomic
tx, _ := db.Begin(ctx)
defer tx.Rollback(ctx)

// Batch 1
batch1 := pgx.NewBatchBuilder()
// ... add queries
batch1.Execute(ctx, tx) // Execute on transaction

// Some non-batch operation
_, err := models.Users.Query(...).One(ctx, tx)

// Batch 2
batch2 := pgx.NewBatchBuilder()
// ... add queries
batch2.Execute(ctx, tx)

tx.Commit(ctx)
```

### When Batch Alone is Sufficient
```go
// Single batch of independent operations
batch := pgx.NewBatchBuilder()
// ... add all queries
batch.Execute(ctx, db) // Direct on pool - atomic by default
```

## Summary

Bob's generated models integrate perfectly with pgx batch operations:

1. Use `models.Users.Insert()` to build insert queries
2. Use `models.Users.Update()` to build update queries
3. Use `models.Users.Query()` to build select queries
4. Queue them with `pgx.Queue*()` functions
5. Execute with `qb.Execute(ctx, db)` - **no transaction wrapper needed!**
6. Enjoy type-safe, efficient batch operations with auto-generated code

The combination of Bob's code generation and pgx batching provides the best of both worlds: type safety with performance.
