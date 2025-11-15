# Batch Query Code Generation

Bob can now auto-generate batch-enabled query functions from SQL definitions with the `:batch` annotation.

## Overview

When you mark a query with the `batch` option, Bob automatically generates:
1. A `Batch` type for that query
2. A constructor function `New{QueryName}Batch()`
3. A `Queue()` method to add queries to the batch
4. An `Execute()` method to run all queued queries in a single database round trip
5. A `Results()` method to access the results (for SELECT/RETURNING queries)

This provides a type-safe, ergonomic API for efficient batch operations.

## Syntax

Add `:batch` as the 4th parameter in the query annotation:

```sql
-- QueryName result_type_one:result_type_all:result_type_transformer:batch
```

Any of these values in the 4th position enables batch generation:
- `batch`
- `true`
- `yes`
- `1`

## Examples

### Basic Batch Insert

```sql
-- InsertUser :::batch
INSERT INTO users (id, name, email) VALUES ($1, $2, $3)
RETURNING *;
```

This generates:

```go
type InsertUserBatch struct {
    qb      *pgx.QueuedBatch
    results []InsertUserRow
}

func NewInsertUserBatch() *InsertUserBatch {
    return &InsertUserBatch{
        qb: pgx.NewQueuedBatch(),
    }
}

func (b *InsertUserBatch) Queue(ctx context.Context, ID int32, Name string, Email string) error {
    query := InsertUser(ID, Name, Email)
    var result InsertUserRow
    err := pgx.QueueInsertRowReturning(b.qb, ctx, query,
        scan.StructMapper[InsertUserRow](), &result)
    if err != nil {
        return err
    }
    b.results = append(b.results, result)
    return nil
}

func (b *InsertUserBatch) Execute(ctx context.Context, exec bob.Executor) error {
    return b.qb.Execute(ctx, exec)
}

func (b *InsertUserBatch) Results() []InsertUserRow {
    return b.results
}

func (b *InsertUserBatch) Len() int {
    return len(b.results)
}
```

### Usage

```go
// Create a new batch
batch := NewInsertUserBatch()

// Queue multiple inserts
batch.Queue(ctx, 1, "Alice", "alice@example.com")
batch.Queue(ctx, 2, "Bob", "bob@example.com")
batch.Queue(ctx, 3, "Charlie", "charlie@example.com")

// Execute all in one round trip
if err := batch.Execute(ctx, db); err != nil {
    return err
}

// Get all results
users := batch.Results()
for _, user := range users {
    fmt.Printf("Inserted user: %+v\n", user)
}
```

### Batch Select

```sql
-- GetUser :::batch
SELECT * FROM users WHERE id = $1;
```

Usage:

```go
batch := NewGetUserBatch()

// Queue multiple selects
for _, id := range userIDs {
    batch.Queue(ctx, id)
}

// Execute
batch.Execute(ctx, db)

// Get results
users := batch.Results()
```

### Batch Update with RETURNING

```sql
-- UpdateUserEmail :::batch
UPDATE users SET email = $1 WHERE id = $2
RETURNING *;
```

Usage:

```go
batch := NewUpdateUserEmailBatch()

batch.Queue(ctx, "new1@example.com", 1)
batch.Queue(ctx, "new2@example.com", 2)
batch.Queue(ctx, "new3@example.com", 3)

batch.Execute(ctx, db)

updatedUsers := batch.Results()
```

### Batch Delete (No RETURNING)

```sql
-- DeleteUser :::batch
DELETE FROM users WHERE id = $1;
```

For queries without RETURNING, no Results() method is generated:

```go
batch := NewDeleteUserBatch()

batch.Queue(ctx, 1)
batch.Queue(ctx, 2)
batch.Queue(ctx, 3)

// Execute all deletes
batch.Execute(ctx, db)
```

## Combined with Other Annotations

You can combine batch with result type annotations:

```sql
-- InsertUser *models.User::scan.StructMapper[ONETYPE, ALLTYPE]:batch
INSERT INTO users (name, email) VALUES ($1, $2)
RETURNING *;
```

This uses custom result types with batch functionality:

```go
func (b *InsertUserBatch) Queue(ctx context.Context, Name string, Email string) error {
    query := InsertUser(Name, Email)
    var result *models.User
    err := pgx.QueueInsertRowReturning(b.qb, ctx, query,
        scan.StructMapper[*models.User](), &result)
    if err != nil {
        return err
    }
    b.results = append(b.results, result)
    return nil
}

func (b *InsertUserBatch) Results() []*models.User {
    return b.results
}
```

## Single Column Results

For queries returning a single column:

```sql
-- GetUserEmails :::batch
SELECT email FROM users WHERE id = $1;
```

Generates:

```go
func (b *GetUserEmailsBatch) Queue(ctx context.Context, ID int32) error {
    query := GetUserEmails(ID)
    var result string
    err := pgx.QueueSelectRow(b.qb, ctx, query,
        scan.SingleColumnMapper[string](&result), &result)
    if err != nil {
        return err
    }
    b.results = append(b.results, GetUserEmailsRow{Email: result})
    return nil
}
```

## Performance Benefits

Batch operations execute all queries in a single round trip to the database:

```go
// Without batch: 100 round trips
for i := 0; i < 100; i++ {
    user, _ := GetUser(i).One(ctx, db)  // 100 database calls
}

// With batch: 1 round trip
batch := NewGetUserBatch()
for i := 0; i < 100; i++ {
    batch.Queue(ctx, i)
}
batch.Execute(ctx, db)  // Single database call
users := batch.Results()
```

Performance improvement: **100x fewer round trips**

## When to Use Batch Queries

### Good Use Cases
- Bulk inserts of many records
- Fetching multiple records by ID
- Updating multiple records individually
- Mixed operations that don't depend on each other
- High-latency database connections

### Not Recommended
- Single operations (just use normal queries)
- Operations that depend on previous results
- Very large datasets (consider COPY for PostgreSQL)

## Requirements

- **PostgreSQL only** - Uses pgx driver's batch capabilities
- **Bob drivers/pgx** - Requires the pgx driver package
- **RETURNING clause** - For INSERT/UPDATE/DELETE to get results back

## Generated Files

For each `.sql` file with batch queries:
- `filename.bob.go` - Contains both regular query functions and batch types
- `filename.bob_test.go` - Contains generated tests

## Complete Example

```sql
-- users.sql

-- InsertUser :::batch
INSERT INTO users (name, email) VALUES ($1, $2)
RETURNING *;

-- GetUserByID :::batch
SELECT * FROM users WHERE id = $1;

-- UpdateUserEmail :::batch
UPDATE users SET email = $1 WHERE id = $2
RETURNING *;

-- DeleteUser :::batch
DELETE FROM users WHERE id = $1;
```

Usage:

```go
package main

import (
    "context"
    "fmt"
    "your/project/queries"
)

func processUsers(ctx context.Context, db *pgx.Pool) error {
    // Batch insert
    insertBatch := queries.NewInsertUserBatch()
    insertBatch.Queue(ctx, "Alice", "alice@example.com")
    insertBatch.Queue(ctx, "Bob", "bob@example.com")

    if err := insertBatch.Execute(ctx, db); err != nil {
        return err
    }

    insertedUsers := insertBatch.Results()

    // Batch select
    selectBatch := queries.NewGetUserByIDBatch()
    for _, user := range insertedUsers {
        selectBatch.Queue(ctx, user.ID)
    }

    if err := selectBatch.Execute(ctx, db); err != nil {
        return err
    }

    fetchedUsers := selectBatch.Results()

    // Batch update
    updateBatch := queries.NewUpdateUserEmailBatch()
    for _, user := range fetchedUsers {
        updateBatch.Queue(ctx, fmt.Sprintf("new_%s", user.Email), user.ID)
    }

    if err := updateBatch.Execute(ctx, db); err != nil {
        return err
    }

    updatedUsers := updateBatch.Results()

    return nil
}
```

## Error Handling

Errors can occur at two stages:

1. **Queue time** - Invalid query parameters
2. **Execute time** - Database errors

```go
batch := NewInsertUserBatch()

// Queue errors (parameter validation)
if err := batch.Queue(ctx, name, email); err != nil {
    return fmt.Errorf("failed to queue: %w", err)
}

// Execute errors (database errors)
if err := batch.Execute(ctx, db); err != nil {
    return fmt.Errorf("batch execution failed: %w", err)
}
```

## Migration Guide

To add batch support to existing queries:

1. Add `:::batch` annotation to SQL queries
2. Re-run code generation: `go generate ./...`
3. Update code to use batch types:

```go
// Before
for _, data := range items {
    user, err := InsertUser(data.Name, data.Email).One(ctx, db)
    if err != nil {
        return err
    }
    results = append(results, user)
}

// After
batch := NewInsertUserBatch()
for _, data := range items {
    if err := batch.Queue(ctx, data.Name, data.Email); err != nil {
        return err
    }
}

if err := batch.Execute(ctx, db); err != nil {
    return err
}

results = batch.Results()
```

## Comparison with Manual Batching

### Manual Approach
```go
qb := pgx.NewQueuedBatch()
var users []User

for _, name := range names {
    var user User
    query := models.Users.Insert(&models.UserSetter{Name: omit.From(name)})
    pgx.QueueInsertRowReturning(qb, ctx, query,
        scan.StructMapper[User](), &user)
    users = append(users, user)
}

qb.Execute(ctx, db)
```

### Generated Batch Approach
```go
batch := NewInsertUserBatch()

for _, name := range names {
    batch.Queue(ctx, name)
}

batch.Execute(ctx, db)
users := batch.Results()
```

Both are equivalent, but the generated approach:
- ✅ Type-safe with compile-time checks
- ✅ Less boilerplate
- ✅ Consistent API across all queries
- ✅ Auto-updated when SQL changes

## Limitations

1. **PostgreSQL/pgx only** - Not available for MySQL or SQLite
2. **No inter-query dependencies** - Query N can't use results from Query N-1
3. **Results processed in order** - Cannot skip to middle result
4. **No partial success** - All queries succeed or all fail (atomic)

## Best Practices

1. **Batch similar operations** - Group inserts together, updates together, etc.
2. **Reasonable batch sizes** - Don't queue thousands of queries (consider chunking)
3. **Handle errors gracefully** - Log which specific Queue call failed
4. **Use with RETURNING** - Get inserted IDs back for relationship handling
5. **Profile performance** - Measure actual improvement for your use case

## Summary

The batch query generation feature provides:
- 🚀 **Performance** - Reduce round trips from N to 1
- 🛡️ **Type Safety** - Compile-time parameter checking
- 📦 **Auto-Generated** - Consistent API from SQL definitions
- 🎯 **Ergonomic** - Clean, simple API surface
- ⚡ **Efficient** - Built on pgx's optimized batch execution

Simply add `:::batch` to any query annotation and Bob generates everything you need for efficient batch operations!
