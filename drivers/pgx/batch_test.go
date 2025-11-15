package pgx_test

import (
	"context"
	"errors"
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

func TestQueuedBatch(t *testing.T) {
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

	t.Run("QueueSelectRow and Execute", func(t *testing.T) {
		tx, err := db.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer tx.Rollback(ctx)

		// Insert test data
		insertQuery := psql.Insert(
			im.Into("test_users"),
			im.Values(psql.Arg("Laura", 28)),
		)
		_, err = insertQuery.Exec(ctx, tx)
		if err != nil {
			t.Fatal(err)
		}

		// Use QueuedBatch
		qb := pgx.NewQueuedBatch()

		type User struct {
			Name string `db:"name"`
			Age  int    `db:"age"`
		}
		var user User
		var count int64

		selectQuery := psql.Select(
			sm.Columns("name", "age"),
			sm.From("test_users"),
			sm.Where(psql.Quote("name").EQ(psql.Arg("Laura"))),
		)
		countQuery := psql.Select(
			sm.Columns(psql.Quote("count(*)")),
			sm.From("test_users"),
		)

		err = pgx.QueueSelectRow(qb, ctx, selectQuery, scan.StructMapper[User](), &user)
		if err != nil {
			t.Fatal(err)
		}

		err = pgx.QueueSelectRow(qb, ctx, countQuery, scan.SingleColumnMapper[int64], &count)
		if err != nil {
			t.Fatal(err)
		}

		// Execute batch
		err = qb.Execute(ctx, tx)
		if err != nil {
			t.Fatal(err)
		}

		// Verify results
		if user.Name != "Laura" || user.Age != 28 {
			t.Fatalf("unexpected user: %+v", user)
		}
		if count != 1 {
			t.Fatalf("expected count 1, got %d", count)
		}

		tx.Commit(ctx)
	})

	t.Run("QueueSelectAll", func(t *testing.T) {
		tx, err := db.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer tx.Rollback(ctx)

		// Insert test data
		for _, name := range []string{"Mike", "Nancy", "Oscar"} {
			insertQuery := psql.Insert(
				im.Into("test_users"),
				im.Values(psql.Arg(name, 30)),
			)
			_, err = insertQuery.Exec(ctx, tx)
			if err != nil {
				t.Fatal(err)
			}
		}

		qb := pgx.NewQueuedBatch()

		type User struct {
			Name string `db:"name"`
			Age  int    `db:"age"`
		}
		var users []User

		selectQuery := psql.Select(
			sm.Columns("name", "age"),
			sm.From("test_users"),
			sm.OrderBy("name"),
		)

		err = pgx.QueueSelectAll(qb, ctx, selectQuery, scan.StructMapper[User](), &users)
		if err != nil {
			t.Fatal(err)
		}

		err = qb.Execute(ctx, tx)
		if err != nil {
			t.Fatal(err)
		}

		if len(users) != 3 {
			t.Fatalf("expected 3 users, got %d", len(users))
		}
		if users[0].Name != "Mike" {
			t.Fatalf("unexpected first user: %+v", users[0])
		}

		tx.Commit(ctx)
	})

	t.Run("QueueInsertRowReturning", func(t *testing.T) {
		tx, err := db.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer tx.Rollback(ctx)

		qb := pgx.NewQueuedBatch()

		type User struct {
			ID   int    `db:"id"`
			Name string `db:"name"`
			Age  int    `db:"age"`
		}
		var user User

		insertQuery := psql.Insert(
			im.Into("test_users"),
			im.Values(psql.Arg("Paula", 35)),
			im.Returning("id", "name", "age"),
		)

		err = pgx.QueueInsertRowReturning(qb, ctx, insertQuery, scan.StructMapper[User](), &user)
		if err != nil {
			t.Fatal(err)
		}

		err = qb.Execute(ctx, tx)
		if err != nil {
			t.Fatal(err)
		}

		if user.Name != "Paula" || user.Age != 35 || user.ID == 0 {
			t.Fatalf("unexpected user: %+v", user)
		}

		tx.Commit(ctx)
	})

	t.Run("QueueUpdateRowReturning", func(t *testing.T) {
		tx, err := db.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer tx.Rollback(ctx)

		// Insert test data
		insertQuery := psql.Insert(
			im.Into("test_users"),
			im.Values(psql.Arg("Quinn", 40)),
		)
		_, err = insertQuery.Exec(ctx, tx)
		if err != nil {
			t.Fatal(err)
		}

		qb := pgx.NewQueuedBatch()

		type User struct {
			Name string `db:"name"`
			Age  int    `db:"age"`
		}
		var user User

		updateQuery := psql.Update(
			um.Table("test_users"),
			um.SetCol("age").ToArg(41),
			um.Where(psql.Quote("name").EQ(psql.Arg("Quinn"))),
			um.Returning("name", "age"),
		)

		err = pgx.QueueUpdateRowReturning(qb, ctx, updateQuery, scan.StructMapper[User](), &user)
		if err != nil {
			t.Fatal(err)
		}

		err = qb.Execute(ctx, tx)
		if err != nil {
			t.Fatal(err)
		}

		if user.Name != "Quinn" || user.Age != 41 {
			t.Fatalf("unexpected user: %+v", user)
		}

		tx.Commit(ctx)
	})

	t.Run("QueueExecRow validates one row", func(t *testing.T) {
		tx, err := db.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer tx.Rollback(ctx)

		// Insert test data
		insertQuery := psql.Insert(
			im.Into("test_users"),
			im.Values(psql.Arg("Rachel", 45)),
		)
		_, err = insertQuery.Exec(ctx, tx)
		if err != nil {
			t.Fatal(err)
		}

		qb := pgx.NewQueuedBatch()

		updateQuery := psql.Update(
			um.Table("test_users"),
			um.SetCol("age").ToArg(46),
			um.Where(psql.Quote("name").EQ(psql.Arg("Rachel"))),
		)

		err = qb.QueueExecRow(updateQuery)
		if err != nil {
			t.Fatal(err)
		}

		err = qb.Execute(ctx, tx)
		if err != nil {
			t.Fatal(err)
		}

		tx.Commit(ctx)
	})

	t.Run("QueueExecRow fails on no rows", func(t *testing.T) {
		tx, err := db.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer tx.Rollback(ctx)

		qb := pgx.NewQueuedBatch()

		updateQuery := psql.Update(
			um.Table("test_users"),
			um.SetCol("age").ToArg(50),
			um.Where(psql.Quote("name").EQ(psql.Arg("NonExistent"))),
		)

		err = qb.QueueExecRow(updateQuery)
		if err != nil {
			t.Fatal(err)
		}

		err = qb.Execute(ctx, tx)
		if err == nil {
			t.Fatal("expected error for zero rows affected")
		}
		if !errors.Is(err, pgx.ErrNoRowsAffected) {
			t.Fatalf("expected ErrNoRowsAffected, got: %v", err)
		}

		tx.Rollback(ctx)
	})

	t.Run("Mixed QueuedBatch operations", func(t *testing.T) {
		tx, err := db.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer tx.Rollback(ctx)

		qb := pgx.NewQueuedBatch()

		type User struct {
			Name string `db:"name"`
			Age  int    `db:"age"`
		}
		var insertedUser User
		var selectedUsers []User
		var count int64

		// Insert with RETURNING
		insertQuery := psql.Insert(
			im.Into("test_users"),
			im.Values(psql.Arg("Sam", 50)),
			im.Returning("name", "age"),
		)
		err = pgx.QueueInsertRowReturning(qb, ctx, insertQuery, scan.StructMapper[User](), &insertedUser)
		if err != nil {
			t.Fatal(err)
		}

		// Select all
		selectQuery := psql.Select(
			sm.Columns("name", "age"),
			sm.From("test_users"),
		)
		err = pgx.QueueSelectAll(qb, ctx, selectQuery, scan.StructMapper[User](), &selectedUsers)
		if err != nil {
			t.Fatal(err)
		}

		// Count
		countQuery := psql.Select(
			sm.Columns(psql.Quote("count(*)")),
			sm.From("test_users"),
		)
		err = pgx.QueueSelectRow(qb, ctx, countQuery, scan.SingleColumnMapper[int64], &count)
		if err != nil {
			t.Fatal(err)
		}

		// Execute all
		err = qb.Execute(ctx, tx)
		if err != nil {
			t.Fatal(err)
		}

		// Verify
		if insertedUser.Name != "Sam" {
			t.Fatalf("unexpected inserted user: %+v", insertedUser)
		}
		if len(selectedUsers) < 1 {
			t.Fatalf("expected at least 1 user, got %d", len(selectedUsers))
		}
		if count < 1 {
			t.Fatalf("expected count >= 1, got %d", count)
		}

		tx.Commit(ctx)
	})
}

func TestExecRow(t *testing.T) {
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

	t.Run("ExecRow with one row", func(t *testing.T) {
		tx, err := db.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer tx.Rollback(ctx)

		// Insert
		insertQuery := psql.Insert(
			im.Into("test_users"),
			im.Values(psql.Arg("Tom", 25)),
		)
		_, err = insertQuery.Exec(ctx, tx)
		if err != nil {
			t.Fatal(err)
		}

		// Update via batch with ExecRow validation
		batch := pgx.NewBatchBuilder()
		updateQuery := psql.Update(
			um.Table("test_users"),
			um.SetCol("age").ToArg(26),
			um.Where(psql.Quote("name").EQ(psql.Arg("Tom"))),
		)
		batch.AddQuery(updateQuery)

		results := batch.Execute(ctx, tx)
		defer results.Close()

		err = pgx.ExecRow(results)
		if err != nil {
			t.Fatal(err)
		}

		tx.Commit(ctx)
	})

	t.Run("ExecRow fails on no rows", func(t *testing.T) {
		tx, err := db.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer tx.Rollback(ctx)

		batch := pgx.NewBatchBuilder()
		updateQuery := psql.Update(
			um.Table("test_users"),
			um.SetCol("age").ToArg(99),
			um.Where(psql.Quote("name").EQ(psql.Arg("DoesNotExist"))),
		)
		batch.AddQuery(updateQuery)

		results := batch.Execute(ctx, tx)
		defer results.Close()

		err = pgx.ExecRow(results)
		if err == nil {
			t.Fatal("expected error for zero rows affected")
		}
		if !errors.Is(err, pgx.ErrNoRowsAffected) {
			t.Fatalf("expected ErrNoRowsAffected, got: %v", err)
		}

		tx.Rollback(ctx)
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
