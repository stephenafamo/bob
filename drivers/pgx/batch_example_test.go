package pgx_test

import (
	"context"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/im"
	"github.com/stephenafamo/bob/dialect/psql/sm"
	"github.com/stephenafamo/bob/drivers/pgx"
	"github.com/stephenafamo/scan"
)

// Example demonstrates basic batch operations with Bob queries
func Example_batchBasic() {
	ctx := context.Background()

	// Setup database connection
	pool, err := pgxpool.New(ctx, "postgres://user:pass@localhost/dbname")
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	db := pgx.NewPool(pool)
	tx, err := db.Begin(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer tx.Rollback(ctx)

	// Create a batch builder
	batch := pgx.NewBatchBuilder()

	// Add multiple insert queries
	for _, name := range []string{"Alice", "Bob", "Charlie"} {
		insertQuery := psql.Insert(
			im.Into("users"),
			im.Values(psql.Arg(name)),
		)
		if err := batch.AddQuery(insertQuery); err != nil {
			log.Fatal(err)
		}
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
		fmt.Printf("Query %d affected %d rows\n", i+1, rows)
	}

	if err := tx.Commit(ctx); err != nil {
		log.Fatal(err)
	}
}

// Example demonstrates batch operations with queries that return data
func Example_batchWithResults() {
	ctx := context.Background()

	pool, err := pgxpool.New(ctx, "postgres://user:pass@localhost/dbname")
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	db := pgx.NewPool(pool)
	tx, err := db.Begin(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer tx.Rollback(ctx)

	// Create batch with mixed operations
	batch := pgx.NewBatchBuilder()

	// Insert a user
	insertQuery := psql.Insert(
		im.Into("users"),
		im.Values(psql.Arg("Dave", "dave@example.com")),
	)
	batch.AddQuery(insertQuery)

	// Select the user back
	selectQuery := psql.Select(
		sm.Columns("id", "name", "email"),
		sm.From("users"),
		sm.Where(psql.Quote("name").EQ(psql.Arg("Dave"))),
	)
	batch.AddQuery(selectQuery)

	// Execute batch
	results := batch.Execute(ctx, tx)
	defer results.Close()

	// Process insert result
	_, err = results.Exec()
	if err != nil {
		log.Fatal(err)
	}

	// Process select result
	type User struct {
		ID    int    `db:"id"`
		Name  string `db:"name"`
		Email string `db:"email"`
	}
	user, err := pgx.ScanOne(ctx, results, scan.StructMapper[User]())
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("User: %+v\n", user)

	tx.Commit(ctx)
}

// Example demonstrates using BatchHelper for convenience
func Example_batchHelper() {
	ctx := context.Background()

	pool, err := pgxpool.New(ctx, "postgres://user:pass@localhost/dbname")
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	db := pgx.NewPool(pool)
	tx, err := db.Begin(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer tx.Rollback(ctx)

	// Create batch helper
	helper := pgx.NewBatchHelper(ctx, tx)

	// Execute multiple insert queries in one batch
	insertQuery1 := psql.Insert(im.Into("users"), im.Values(psql.Arg("Eve", "eve@example.com")))
	insertQuery2 := psql.Insert(im.Into("users"), im.Values(psql.Arg("Frank", "frank@example.com")))
	insertQuery3 := psql.Insert(im.Into("users"), im.Values(psql.Arg("Grace", "grace@example.com")))

	results, err := helper.ExecQueries(insertQuery1, insertQuery2, insertQuery3)
	if err != nil {
		log.Fatal(err)
	}

	for i, res := range results {
		rows, _ := res.RowsAffected()
		fmt.Printf("Insert %d: %d rows affected\n", i+1, rows)
	}

	tx.Commit(ctx)
}

// Example demonstrates batch queries with multiple result sets
func Example_batchQueryAll() {
	ctx := context.Background()

	pool, err := pgxpool.New(ctx, "postgres://user:pass@localhost/dbname")
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	db := pgx.NewPool(pool)
	tx, err := db.Begin(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer tx.Rollback(ctx)

	batch := pgx.NewBatchBuilder()

	type User struct {
		ID   int    `db:"id"`
		Name string `db:"name"`
	}

	// Define multiple select queries
	activeUsersQuery := psql.Select(
		sm.Columns("id", "name"),
		sm.From("users"),
		sm.Where(psql.Quote("active").EQ(psql.Arg(true))),
	)

	inactiveUsersQuery := psql.Select(
		sm.Columns("id", "name"),
		sm.From("users"),
		sm.Where(psql.Quote("active").EQ(psql.Arg(false))),
	)

	// Add queries to batch
	batch.AddQuery(activeUsersQuery)
	batch.AddQuery(inactiveUsersQuery)

	// Execute batch
	results := batch.Execute(ctx, tx)
	defer results.Close()

	// Scan results
	activeUsers, err := pgx.ScanAll(ctx, results, scan.StructMapper[User]())
	if err != nil {
		log.Fatal(err)
	}

	inactiveUsers, err := pgx.ScanAll(ctx, results, scan.StructMapper[User]())
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Active users: %d\n", len(activeUsers))
	fmt.Printf("Inactive users: %d\n", len(inactiveUsers))

	tx.Commit(ctx)
}

// Example demonstrates using raw SQL queries in a batch
func Example_batchRawSQL() {
	ctx := context.Background()

	pool, err := pgxpool.New(ctx, "postgres://user:pass@localhost/dbname")
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	db := pgx.NewPool(pool)
	tx, err := db.Begin(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer tx.Rollback(ctx)

	// Create batch with raw SQL
	batch := pgx.NewBatchBuilder()

	// Add raw SQL queries
	batch.AddRawQuery("INSERT INTO users (name, email) VALUES ($1, $2)", "Henry", "henry@example.com")
	batch.AddRawQuery("INSERT INTO users (name, email) VALUES ($1, $2)", "Iris", "iris@example.com")
	batch.AddRawQuery("UPDATE users SET active = true WHERE name = $1", "Henry")

	// Execute batch
	results := batch.Execute(ctx, tx)
	defer results.Close()

	// Process insert results
	for i := 0; i < 2; i++ {
		res, err := results.Exec()
		if err != nil {
			log.Fatal(err)
		}
		rows, _ := res.RowsAffected()
		fmt.Printf("Insert %d: %d rows\n", i+1, rows)
	}

	// Process update result
	res, err := results.Exec()
	if err != nil {
		log.Fatal(err)
	}
	rows, _ := res.RowsAffected()
	fmt.Printf("Update: %d rows\n", rows)

	tx.Commit(ctx)
}

// Example demonstrates error handling in batch operations
func Example_batchErrorHandling() {
	ctx := context.Background()

	pool, err := pgxpool.New(ctx, "postgres://user:pass@localhost/dbname")
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	db := pgx.NewPool(pool)
	tx, err := db.Begin(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer tx.Rollback(ctx)

	batch := pgx.NewBatchBuilder()

	// Add valid query
	insertQuery1 := psql.Insert(im.Into("users"), im.Values(psql.Arg("Jack")))
	batch.AddQuery(insertQuery1)

	// Add query that might fail (e.g., duplicate key)
	insertQuery2 := psql.Insert(im.Into("users"), im.Values(psql.Arg("Jack")))
	batch.AddQuery(insertQuery2)

	results := batch.Execute(ctx, tx)
	defer results.Close()

	// Process first query
	_, err = results.Exec()
	if err != nil {
		log.Printf("First query failed: %v", err)
	}

	// Process second query - might fail
	_, err = results.Exec()
	if err != nil {
		log.Printf("Second query failed (expected): %v", err)
		// Handle error appropriately
		tx.Rollback(ctx)
		return
	}

	tx.Commit(ctx)
}

// Example demonstrates using batch with context for cancellation
func Example_batchWithContext() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pool, err := pgxpool.New(ctx, "postgres://user:pass@localhost/dbname")
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	db := pgx.NewPool(pool)
	tx, err := db.Begin(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer tx.Rollback(ctx)

	batch := pgx.NewBatchBuilder()

	// Add queries using context
	for i := 0; i < 100; i++ {
		insertQuery := psql.Insert(
			im.Into("users"),
			im.Values(psql.Arg(fmt.Sprintf("User%d", i))),
		)
		if err := batch.AddQueryContext(ctx, insertQuery); err != nil {
			log.Fatal(err)
		}
	}

	// Execute with context (can be cancelled)
	results := batch.Execute(ctx, tx)
	defer results.Close()

	// Process results with potential cancellation
	for i := 0; i < batch.Len(); i++ {
		select {
		case <-ctx.Done():
			log.Println("Operation cancelled")
			return
		default:
			_, err := results.Exec()
			if err != nil {
				log.Fatal(err)
			}
		}
	}

	tx.Commit(ctx)
}
