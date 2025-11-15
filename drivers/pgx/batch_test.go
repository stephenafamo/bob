package pgx_test

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/im"
	"github.com/stephenafamo/bob/dialect/psql/sm"
	"github.com/stephenafamo/bob/dialect/psql/um"
	"github.com/stephenafamo/bob/drivers/pgx"
	"github.com/stephenafamo/scan"
)

func TestBatchBuilder(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()

	// Setup test database connection
	pool, err := pgxpool.New(ctx, getTestDSN())
	if err != nil {
		t.Skipf("skipping test: cannot connect to database: %v", err)
	}
	defer pool.Close()

	db := pgx.NewPool(pool)

	// Create test table
	setupTestTable(t, db)
	defer cleanupTestTable(t, db)

	t.Run("AddQuery and Execute", func(t *testing.T) {
		tx, err := db.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer tx.Rollback(ctx)

		batch := pgx.NewBatchBuilder()

		// Add multiple insert queries
		insertQuery1 := psql.Insert(
			im.Into("test_users"),
			im.Values(psql.Arg("Alice", 30)),
		)
		insertQuery2 := psql.Insert(
			im.Into("test_users"),
			im.Values(psql.Arg("Bob", 25)),
		)
		insertQuery3 := psql.Insert(
			im.Into("test_users"),
			im.Values(psql.Arg("Charlie", 35)),
		)

		if err := batch.AddQuery(insertQuery1); err != nil {
			t.Fatal(err)
		}
		if err := batch.AddQuery(insertQuery2); err != nil {
			t.Fatal(err)
		}
		if err := batch.AddQuery(insertQuery3); err != nil {
			t.Fatal(err)
		}

		if batch.Len() != 3 {
			t.Fatalf("expected batch length 3, got %d", batch.Len())
		}

		// Execute batch
		results := batch.Execute(ctx, tx)
		defer results.Close()

		// Check results
		for i := 0; i < 3; i++ {
			res, err := results.Exec()
			if err != nil {
				t.Fatalf("exec %d failed: %v", i, err)
			}
			rows, err := res.RowsAffected()
			if err != nil {
				t.Fatalf("failed to get rows affected: %v", err)
			}
			if rows != 1 {
				t.Fatalf("expected 1 row affected, got %d", rows)
			}
		}

		if err := tx.Commit(ctx); err != nil {
			t.Fatal(err)
		}

		// Verify data was inserted
		countQuery := psql.Select(sm.Columns(psql.Quote("count(*)")), sm.From("test_users"))
		count, err := bob.One(ctx, db, countQuery, scan.SingleColumnMapper[int64])
		if err != nil {
			t.Fatal(err)
		}
		if count != 3 {
			t.Fatalf("expected 3 rows, got %d", count)
		}
	})

	t.Run("AddRawQuery", func(t *testing.T) {
		tx, err := db.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer tx.Rollback(ctx)

		batch := pgx.NewBatchBuilder()
		batch.AddRawQuery("INSERT INTO test_users (name, age) VALUES ($1, $2)", "Dave", 40)
		batch.AddRawQuery("INSERT INTO test_users (name, age) VALUES ($1, $2)", "Eve", 28)

		if batch.Len() != 2 {
			t.Fatalf("expected batch length 2, got %d", batch.Len())
		}

		results := batch.Execute(ctx, tx)
		defer results.Close()

		for i := 0; i < 2; i++ {
			_, err := results.Exec()
			if err != nil {
				t.Fatalf("exec %d failed: %v", i, err)
			}
		}

		if err := tx.Commit(ctx); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("Query results", func(t *testing.T) {
		tx, err := db.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer tx.Rollback(ctx)

		// Insert test data first
		insertQuery := psql.Insert(
			im.Into("test_users"),
			im.Values(psql.Arg("Frank", 32)),
		)
		_, err = insertQuery.Exec(ctx, tx)
		if err != nil {
			t.Fatal(err)
		}

		batch := pgx.NewBatchBuilder()

		// Add select queries
		selectQuery1 := psql.Select(
			sm.Columns("name", "age"),
			sm.From("test_users"),
			sm.Where(psql.Quote("name").EQ(psql.Arg("Frank"))),
		)
		selectQuery2 := psql.Select(
			sm.Columns("count(*)"),
			sm.From("test_users"),
		)

		if err := batch.AddQuery(selectQuery1); err != nil {
			t.Fatal(err)
		}
		if err := batch.AddQuery(selectQuery2); err != nil {
			t.Fatal(err)
		}

		results := batch.Execute(ctx, tx)
		defer results.Close()

		// First query - get user
		type User struct {
			Name string `db:"name"`
			Age  int    `db:"age"`
		}
		user, err := pgx.ScanOne(ctx, results, scan.StructMapper[User]())
		if err != nil {
			t.Fatal(err)
		}
		if user.Name != "Frank" || user.Age != 32 {
			t.Fatalf("unexpected user: %+v", user)
		}

		// Second query - get count
		count, err := pgx.ScanOne(ctx, results, scan.SingleColumnMapper[int64])
		if err != nil {
			t.Fatal(err)
		}
		if count != 1 {
			t.Fatalf("expected count 1, got %d", count)
		}

		if err := tx.Commit(ctx); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("Mixed operations", func(t *testing.T) {
		tx, err := db.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer tx.Rollback(ctx)

		batch := pgx.NewBatchBuilder()

		// Insert
		insertQuery := psql.Insert(
			im.Into("test_users"),
			im.Values(psql.Arg("George", 45)),
		)
		batch.AddQuery(insertQuery)

		// Update
		updateQuery := psql.Update(
			um.Table("test_users"),
			um.SetCol("age").ToArg(46),
			um.Where(psql.Quote("name").EQ(psql.Arg("George"))),
		)
		batch.AddQuery(updateQuery)

		// Select
		selectQuery := psql.Select(
			sm.Columns("name", "age"),
			sm.From("test_users"),
			sm.Where(psql.Quote("name").EQ(psql.Arg("George"))),
		)
		batch.AddQuery(selectQuery)

		results := batch.Execute(ctx, tx)
		defer results.Close()

		// Process insert result
		_, err = results.Exec()
		if err != nil {
			t.Fatal(err)
		}

		// Process update result
		_, err = results.Exec()
		if err != nil {
			t.Fatal(err)
		}

		// Process select result
		type User struct {
			Name string `db:"name"`
			Age  int    `db:"age"`
		}
		user, err := pgx.ScanOne(ctx, results, scan.StructMapper[User]())
		if err != nil {
			t.Fatal(err)
		}
		if user.Age != 46 {
			t.Fatalf("expected age 46, got %d", user.Age)
		}

		if err := tx.Commit(ctx); err != nil {
			t.Fatal(err)
		}
	})
}

func TestBatchHelper(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()

	pool, err := pgxpool.New(ctx, getTestDSN())
	if err != nil {
		t.Skipf("skipping test: cannot connect to database: %v", err)
	}
	defer pool.Close()

	db := pgx.NewPool(pool)

	setupTestTable(t, db)
	defer cleanupTestTable(t, db)

	t.Run("ExecQueries", func(t *testing.T) {
		tx, err := db.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer tx.Rollback(ctx)

		helper := pgx.NewBatchHelper(ctx, tx)

		insertQuery1 := psql.Insert(
			im.Into("test_users"),
			im.Values(psql.Arg("Henry", 50)),
		)
		insertQuery2 := psql.Insert(
			im.Into("test_users"),
			im.Values(psql.Arg("Iris", 29)),
		)

		results, err := helper.ExecQueries(insertQuery1, insertQuery2)
		if err != nil {
			t.Fatal(err)
		}

		if len(results) != 2 {
			t.Fatalf("expected 2 results, got %d", len(results))
		}

		for i, res := range results {
			rows, err := res.RowsAffected()
			if err != nil {
				t.Fatalf("failed to get rows affected for result %d: %v", i, err)
			}
			if rows != 1 {
				t.Fatalf("expected 1 row affected for result %d, got %d", i, rows)
			}
		}

		if err := tx.Commit(ctx); err != nil {
			t.Fatal(err)
		}
	})

}

// Helper functions for test setup
func getTestDSN() string {
	// Try to get from environment, otherwise use default
	dsn := "postgres://postgres:postgres@localhost:5432/bob_test?sslmode=disable"
	return dsn
}

func setupTestTable(t *testing.T, db pgx.Pool) {
	ctx := context.Background()
	_, err := db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS test_users (
			id SERIAL PRIMARY KEY,
			name TEXT NOT NULL,
			age INTEGER NOT NULL
		)
	`)
	if err != nil {
		t.Fatal(err)
	}

	// Clean any existing data
	_, err = db.ExecContext(ctx, "TRUNCATE TABLE test_users")
	if err != nil {
		t.Fatal(err)
	}
}

func cleanupTestTable(t *testing.T, db pgx.Pool) {
	ctx := context.Background()
	_, err := db.ExecContext(ctx, "DROP TABLE IF EXISTS test_users")
	if err != nil {
		t.Logf("cleanup failed: %v", err)
	}
}
