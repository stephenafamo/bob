# pgx.Batch Support in Bob ORM

This document explains how to use PostgreSQL batch operations with Bob ORM through the pgx driver.

## Overview

Batch operations allow you to execute multiple queries in a single round-trip to the database, significantly improving performance when executing many queries. Bob's pgx driver provides full support for pgx.Batch operations with an easy-to-use API that integrates seamlessly with Bob's query builder.

## Features

- **BatchBuilder**: Build and execute batches of Bob queries
- **BatchResults**: Process batch results with Bob's scan package
- **BatchHelper**: Convenience methods for common batch patterns
- **Type-safe**: Full integration with Bob's type-safe query builder
- **Context support**: Proper context handling for cancellation and timeouts

## Quick Start

### Basic Batch Insert

```go
import (
    "context"
    "github.com/stephenafamo/bob/drivers/pgx"
    "github.com/stephenafamo/bob/dialect/psql"
    "github.com/stephenafamo/bob/dialect/psql/im"
)

func main() {
    ctx := context.Background()

    // Create a batch
    batch := pgx.NewBatchBuilder()

    // Add multiple queries
    for _, name := range []string{"Alice", "Bob", "Charlie"} {
        insertQuery := psql.Insert(
            im.Into("users"),
            im.Values(psql.Arg(name)),
        )
        batch.AddQuery(insertQuery)
    }

    // Execute the batch
    results := batch.Execute(ctx, tx)
    defer results.Close()

    // Process results
    for i := 0; i < batch.Len(); i++ {
        res, err := results.Exec()
        if err != nil {
            log.Fatal(err)
        }
        rows, _ := res.RowsAffected()
        fmt.Printf("Inserted %d rows\n", rows)
    }
}
```

## API Reference

### BatchBuilder

`BatchBuilder` is the main type for building batch operations.

#### Creating a Batch

```go
batch := pgx.NewBatchBuilder()
```

#### Adding Queries

```go
// Add a Bob query
err := batch.AddQuery(query)

// Add a Bob query with context
err := batch.AddQueryContext(ctx, query)

// Add raw SQL
batch.AddRawQuery("INSERT INTO users (name) VALUES ($1)", "Alice")
```

#### Executing the Batch

```go
// Execute on a transaction
results := batch.Execute(ctx, tx)
defer results.Close()

// Execute on a connection
results := batch.Execute(ctx, conn)
defer results.Close()
```

#### Getting Batch Length

```go
length := batch.Len() // Returns number of queued queries
```

### BatchResults

`BatchResults` wraps pgx.BatchResults with Bob-friendly methods.

#### Processing Non-Query Results (INSERT, UPDATE, DELETE)

```go
res, err := results.Exec()
if err != nil {
    log.Fatal(err)
}
rowsAffected, _ := res.RowsAffected()
```

#### Processing Query Results (SELECT)

```go
// Scan single row
var user User
err := results.One(ctx, &user)

// Scan multiple rows
var users []User
err := results.All(ctx, &users)

// Low-level access
rows, err := results.Query()
row := results.QueryRow()
```

#### Closing Results

Always close BatchResults when done:

```go
defer results.Close()
```

### BatchHelper

`BatchHelper` provides convenience methods for common patterns.

#### Creating a Helper

```go
helper := pgx.NewBatchHelper(ctx, tx)
```

#### Executing Multiple Queries

```go
results, err := helper.ExecQueries(query1, query2, query3)
for i, res := range results {
    rows, _ := res.RowsAffected()
    fmt.Printf("Query %d: %d rows\n", i+1, rows)
}
```

#### Querying Multiple Result Sets

```go
var activeUsers []User
var inactiveUsers []User

err := helper.QueryAll(
    []bob.Query{activeUsersQuery, inactiveUsersQuery},
    []any{&activeUsers, &inactiveUsers},
)
```

## Common Use Cases

### 1. Bulk Insert

```go
batch := pgx.NewBatchBuilder()

users := []string{"Alice", "Bob", "Charlie", "Dave"}
for _, name := range users {
    query := psql.Insert(
        im.Into("users"),
        im.Values(psql.Arg(name)),
    )
    batch.AddQuery(query)
}

results := batch.Execute(ctx, tx)
defer results.Close()

for i := 0; i < batch.Len(); i++ {
    if _, err := results.Exec(); err != nil {
        return err
    }
}
```

### 2. Mixed Operations

```go
batch := pgx.NewBatchBuilder()

// Insert
batch.AddQuery(psql.Insert(im.Into("users"), im.Values(psql.Arg("Eve"))))

// Update
batch.AddQuery(psql.Update(
    dm.Table("users"),
    dm.Set("active", true),
    dm.Where(psql.Quote("name").EQ(psql.Arg("Eve"))),
))

// Select
batch.AddQuery(psql.Select(
    sm.Columns("*"),
    sm.From("users"),
    sm.Where(psql.Quote("name").EQ(psql.Arg("Eve"))),
))

results := batch.Execute(ctx, tx)
defer results.Close()

// Process insert
results.Exec()

// Process update
results.Exec()

// Process select
var user User
results.One(ctx, &user)
```

### 3. Batch with Error Handling

```go
batch := pgx.NewBatchBuilder()

for _, name := range names {
    query := psql.Insert(im.Into("users"), im.Values(psql.Arg(name)))
    if err := batch.AddQuery(query); err != nil {
        return fmt.Errorf("failed to add query: %w", err)
    }
}

results := batch.Execute(ctx, tx)
defer results.Close()

successCount := 0
for i := 0; i < batch.Len(); i++ {
    if res, err := results.Exec(); err != nil {
        log.Printf("Query %d failed: %v", i, err)
        // Optionally rollback transaction
        tx.Rollback(ctx)
        return err
    } else {
        successCount++
    }
}

log.Printf("Successfully executed %d queries", successCount)
```

### 4. Using Raw SQL

When you need to execute SQL that's not easily built with Bob's query builder:

```go
batch := pgx.NewBatchBuilder()

batch.AddRawQuery(`
    INSERT INTO users (name, email, created_at)
    VALUES ($1, $2, NOW())
`, "Frank", "frank@example.com")

batch.AddRawQuery(`
    UPDATE users
    SET last_login = NOW()
    WHERE name = $1
`, "Frank")

results := batch.Execute(ctx, tx)
defer results.Close()

// Process results...
```

## Performance Considerations

### When to Use Batches

Batches are most beneficial when:

- Executing many similar queries (e.g., bulk inserts)
- Performing multiple independent operations
- Minimizing network round-trips is important

### When NOT to Use Batches

Batches may not be ideal when:

- Queries depend on results from previous queries
- You need fine-grained error handling per query
- Working with very large datasets (consider COPY instead)

### COPY for Bulk Inserts

For very large bulk inserts, consider using pgx's COPY protocol directly:

```go
// Access underlying pgx transaction
copyCount, err := tx.CopyFrom(
    ctx,
    pgx.Identifier{"users"},
    []string{"name", "email"},
    pgx.CopyFromRows(rows),
)
```

## Error Handling

### Query Building Errors

Errors during query building are returned immediately:

```go
err := batch.AddQuery(query)
if err != nil {
    // Handle query building error
    return err
}
```

### Execution Errors

Errors during execution are returned when processing results:

```go
results := batch.Execute(ctx, tx)
defer results.Close()

res, err := results.Exec()
if err != nil {
    // Handle execution error
    return err
}
```

### Partial Success

If a query in the middle of a batch fails, subsequent queries may not be executed. Always check errors for each result:

```go
for i := 0; i < batch.Len(); i++ {
    _, err := results.Exec()
    if err != nil {
        log.Printf("Query %d failed: %v", i, err)
        // Decide whether to continue or abort
        break
    }
}
```

## Context and Cancellation

Batch operations respect context cancellation:

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

batch := pgx.NewBatchBuilder()
// Add queries using context
for _, query := range queries {
    if err := batch.AddQueryContext(ctx, query); err != nil {
        return err
    }
}

results := batch.Execute(ctx, tx)
defer results.Close()
```

## Best Practices

1. **Always close BatchResults**: Use `defer results.Close()` immediately after getting results
2. **Process results in order**: Results must be processed in the same order queries were added
3. **Check all errors**: Each query can fail independently
4. **Use transactions**: Wrap batches in transactions for consistency
5. **Limit batch size**: Very large batches can cause memory issues
6. **Use AddQueryContext**: Prefer `AddQueryContext` for proper context propagation

## Examples

See `batch_example_test.go` for complete working examples:

- Basic batch operations
- Mixed query types
- Error handling
- Context usage
- BatchHelper usage

## Integration with Bob

Batch operations work seamlessly with all Bob features:

- **Generated models**: Use generated insert/update/delete methods
- **Type safety**: Full type checking at compile time
- **Relationships**: Combine with Bob's relationship loading
- **Hooks**: Query hooks are respected (executed per query)

## Comparison with Other Approaches

### Multiple Individual Queries

```go
// Without batch (N round-trips)
for _, name := range names {
    _, err := psql.Insert(im.Into("users"), im.Values(psql.Arg(name))).Exec(ctx, tx)
}
```

### With Batch (1 round-trip)

```go
batch := pgx.NewBatchBuilder()
for _, name := range names {
    batch.AddQuery(psql.Insert(im.Into("users"), im.Values(psql.Arg(name))))
}
results := batch.Execute(ctx, tx)
defer results.Close()
```

**Performance**: Batch operations can be 5-10x faster for multiple queries.

## Limitations

- Batch operations are PostgreSQL/pgx specific
- Results must be processed in order
- Cannot reuse a BatchBuilder after execution
- Prepared statements are not automatically used (pgx handles this internally)

## See Also

- [pgx documentation](https://pkg.go.dev/github.com/jackc/pgx/v5)
- [Bob query builder documentation](https://github.com/stephenafamo/bob)
- PostgreSQL [batch operation best practices](https://www.postgresql.org/docs/current/sql-prepare.html)
