package pgx_test

import (
	"context"
	"fmt"
	"log"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/im"
	"github.com/stephenafamo/bob/dialect/psql/sm"
	"github.com/stephenafamo/bob/dialect/psql/um"
	"github.com/stephenafamo/bob/drivers/pgx"
	"github.com/stephenafamo/bob/orm/omit"
	"github.com/stephenafamo/scan"
)

// Example: Demonstrates how to use batch operations with Bob-generated models
// This example simulates working with generated models by manually defining the types

// User represents a generated model (normally from gen/models/users.go)
type User struct {
	ID    int    `db:"id,pk"`
	Name  string `db:"name"`
	Email string `db:"email"`
}

// UserSlice is the generated slice type
type UserSlice []*User

// UserSetter is the generated setter type with optional fields
type UserSetter struct {
	ID    omit.Val[int]    `db:"id"`
	Name  omit.Val[string] `db:"name"`
	Email omit.Val[string] `db:"email"`
}

// Apply implements the orm.Setter interface for UserSetter
// This would be generated in gen/models/users.go
func (s *UserSetter) Apply(q *psql.InsertQuery) {
	expressions := []bob.Expression{}

	if s.ID.IsSet() {
		expressions = append(expressions, psql.Arg(s.ID.GetOrZero()))
	} else {
		expressions = append(expressions, psql.Raw("DEFAULT"))
	}

	if s.Name.IsSet() {
		expressions = append(expressions, psql.Arg(s.Name.GetOrZero()))
	} else {
		expressions = append(expressions, psql.Raw("DEFAULT"))
	}

	if s.Email.IsSet() {
		expressions = append(expressions, psql.Arg(s.Email.GetOrZero()))
	} else {
		expressions = append(expressions, psql.Raw("DEFAULT"))
	}

	q.Apply(im.Values(expressions...))
}

// Users is the generated table variable (normally from gen/models/users.go)
// In real code this would be: var Users = psql.NewTablex[*User, UserSlice, *UserSetter](...)
var Users = psql.NewTablex[*User, UserSlice, *UserSetter]("public", "users")

// ExampleBatchInsert demonstrates batch insert with generated models
func ExampleBatchInsert() {
	ctx := context.Background()
	// db would be your *pgx.Pool
	var db bob.Executor

	qb := pgx.NewQueuedBatch()

	// Prepare result storage
	var users UserSlice

	// Insert multiple users using the generated table
	names := []string{"Alice", "Bob", "Charlie"}
	for _, name := range names {
		var user User
		insertQ := Users.Insert(
			&UserSetter{Name: omit.From(name)},
			im.Returning("*"),
		)

		// Queue insert - result will be populated on Execute
		err := pgx.QueueInsertRowReturning(qb, ctx, insertQ,
			scan.StructMapper[User](), &user)
		if err != nil {
			log.Fatal(err)
		}
		users = append(users, &user)
	}

	// Execute batch - single round trip!
	if err := qb.Execute(ctx, db); err != nil {
		log.Fatal(err)
	}

	// All users now have their IDs populated
	for _, user := range users {
		fmt.Printf("Inserted user: ID=%d, Name=%s\n", user.ID, user.Name)
	}
}

// ExampleBatchBulkInsert demonstrates bulk insert with slice
func ExampleBatchBulkInsert() {
	ctx := context.Background()
	var db bob.Executor

	qb := pgx.NewQueuedBatch()

	// Insert multiple users in a single INSERT statement
	insertQ := Users.Insert(
		&UserSetter{Name: omit.From("Alice"), Email: omit.From("alice@example.com")},
		&UserSetter{Name: omit.From("Bob"), Email: omit.From("bob@example.com")},
		&UserSetter{Name: omit.From("Charlie"), Email: omit.From("charlie@example.com")},
		im.Returning("*"),
	)

	var users UserSlice
	err := pgx.QueueInsertReturning(qb, ctx, insertQ,
		scan.StructMapper[*User](), &users)
	if err != nil {
		log.Fatal(err)
	}

	// Execute batch
	if err := qb.Execute(ctx, db); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Inserted %d users in bulk\n", len(users))
}

// ExampleBatchUpdate demonstrates batch updates with RETURNING
func ExampleBatchUpdate() {
	ctx := context.Background()
	var db bob.Executor

	qb := pgx.NewQueuedBatch()

	// Update multiple users individually
	userIDs := []int{1, 2, 3}
	var updatedUsers UserSlice

	// Simulate Users.Columns for type-safe column references
	type userColumns struct {
		ID    psql.Expression
		Name  psql.Expression
		Email psql.Expression
	}
	cols := userColumns{
		ID:    psql.Quote("id"),
		Name:  psql.Quote("name"),
		Email: psql.Quote("email"),
	}

	for _, id := range userIDs {
		updateQ := Users.Update(
			um.Set("last_login", "NOW()"),
			um.Where(cols.ID.EQ(psql.Arg(id))),
			um.Returning("*"),
		)

		var user User
		err := pgx.QueueUpdateRowReturning(qb, ctx, updateQ,
			scan.StructMapper[User](), &user)
		if err != nil {
			log.Fatal(err)
		}
		updatedUsers = append(updatedUsers, &user)
	}

	if err := qb.Execute(ctx, db); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Updated %d users\n", len(updatedUsers))
}

// ExampleBatchMixedOperations demonstrates mixing inserts, updates, and selects
func ExampleBatchMixedOperations() {
	ctx := context.Background()
	var db bob.Executor

	qb := pgx.NewQueuedBatch()

	// Column references
	cols := struct {
		ID    psql.Expression
		Name  psql.Expression
		Email psql.Expression
	}{
		ID:    psql.Quote("id"),
		Name:  psql.Quote("name"),
		Email: psql.Quote("email"),
	}

	// 1. Insert a new user
	var newUser User
	insertQ := Users.Insert(
		&UserSetter{
			Name:  omit.From("David"),
			Email: omit.From("david@example.com"),
		},
		im.Returning("*"),
	)
	pgx.QueueInsertRowReturning(qb, ctx, insertQ,
		scan.StructMapper[User](), &newUser)

	// 2. Update an existing user
	updateQ := Users.Update(
		um.Set("email", "newemail@example.com"),
		um.Where(cols.Name.EQ(psql.Arg("Alice"))),
	)
	pgx.QueueExecRow(qb, ctx, updateQ) // Validates exactly 1 row updated

	// 3. Query users
	var allUsers UserSlice
	selectQ := Users.Query(
		sm.Columns(cols.ID, cols.Name, cols.Email),
		sm.OrderBy(cols.Name),
	)
	pgx.QueueSelectAll(qb, ctx, selectQ,
		scan.StructMapper[*User](), &allUsers)

	// Execute all operations in one batch
	if err := qb.Execute(ctx, db); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("New user: %+v\n", newUser)
	fmt.Printf("Total users: %d\n", len(allUsers))
}

// ExampleBatchDelete demonstrates batch deletes
func ExampleBatchDelete() {
	ctx := context.Background()
	var db bob.Executor

	qb := pgx.NewQueuedBatch()

	// Delete multiple users by ID
	idsToDelete := []int{10, 11, 12}

	cols := struct{ ID psql.Expression }{ID: psql.Quote("id")}

	for _, id := range idsToDelete {
		deleteQ := Users.Delete(
			sm.Where(cols.ID.EQ(psql.Arg(id))),
		)
		// QueueExecRow ensures exactly 1 row is deleted
		pgx.QueueExecRow(qb, ctx, deleteQ)
	}

	if err := qb.Execute(ctx, db); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Deleted %d users\n", len(idsToDelete))
}

// ExampleBatchWithValidation demonstrates validation patterns
func ExampleBatchWithValidation() {
	ctx := context.Background()
	var db bob.Executor

	type RegistrationRequest struct {
		Name  string
		Email string
	}

	requests := []RegistrationRequest{
		{Name: "Alice", Email: "alice@example.com"},
		{Name: "Bob", Email: "bob@example.com"},
	}

	qb := pgx.NewQueuedBatch()
	var users UserSlice

	cols := struct{ Email psql.Expression }{Email: psql.Quote("email")}

	for _, req := range requests {
		// Check if email exists
		existsQ := Users.Query(
			sm.Columns("COUNT(*)"),
			sm.Where(cols.Email.EQ(psql.Arg(req.Email))),
		)
		var count int
		pgx.QueueSelectRow(qb, ctx, existsQ,
			scan.SingleColumnMapper[int](&count), &count)

		// Insert user
		var user User
		insertQ := Users.Insert(
			&UserSetter{
				Name:  omit.From(req.Name),
				Email: omit.From(req.Email),
			},
			im.Returning("*"),
		)
		pgx.QueueInsertRowReturning(qb, ctx, insertQ,
			scan.StructMapper[User](), &user)

		users = append(users, &user)
	}

	if err := qb.Execute(ctx, db); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Registered %d users\n", len(users))
}

// ExampleBatchPerformanceComparison shows the performance difference
func ExampleBatchPerformanceComparison() {
	ctx := context.Background()
	var db bob.Executor

	names := []string{"User1", "User2", "User3", "User4", "User5"}

	// BAD: Without batch - 5 round trips
	fmt.Println("Without batch (5 round trips):")
	for _, name := range names {
		_, err := Users.Insert(&UserSetter{
			Name: omit.From(name),
		}).One(ctx, db)
		if err != nil {
			log.Fatal(err)
		}
	}

	// GOOD: With batch - 1 round trip
	fmt.Println("With batch (1 round trip):")
	qb := pgx.NewQueuedBatch()
	var users UserSlice

	for _, name := range names {
		var user User
		insertQ := Users.Insert(
			&UserSetter{Name: omit.From(name)},
			im.Returning("*"),
		)
		pgx.QueueInsertRowReturning(qb, ctx, insertQ,
			scan.StructMapper[User](), &user)
		users = append(users, &user)
	}

	if err := qb.Execute(ctx, db); err != nil {
		log.Fatal(err)
	}

	fmt.Println("Batch approach is ~5x faster!")
}

// ExampleBatchErrorHandling demonstrates error handling patterns
func ExampleBatchErrorHandling() {
	ctx := context.Background()
	var db bob.Executor

	qb := pgx.NewQueuedBatch()

	// Add queries
	for i := 1; i <= 3; i++ {
		insertQ := Users.Insert(&UserSetter{
			Name: omit.From(fmt.Sprintf("User%d", i)),
		})

		if err := pgx.QueueExec(qb, ctx, insertQ); err != nil {
			// Error building query
			log.Printf("Failed to queue insert %d: %v", i, err)
			return
		}
	}

	// Execute batch
	if err := qb.Execute(ctx, db); err != nil {
		// Error executing batch
		log.Printf("Batch execution failed: %v", err)
		return
	}

	fmt.Println("All inserts succeeded")
}

// ExampleBatchTransactionComparison shows when to use transactions
func ExampleBatchTransactionComparison() {
	ctx := context.Background()
	var pool *pgx.Pool

	// Case 1: Batch alone (single atomic operation)
	fmt.Println("Batch without transaction:")
	batch := pgx.NewBatchBuilder()
	// ... add queries
	results := batch.Execute(ctx, pool) // Direct on pool - atomic
	results.Close()

	// Case 2: Batch within transaction (multiple steps need atomicity)
	fmt.Println("Batch within transaction:")
	tx, _ := pool.Begin(ctx)
	defer tx.Rollback(ctx)

	// Batch 1
	batch1 := pgx.NewBatchBuilder()
	// ... add queries
	batch1.Execute(ctx, tx) // Execute on tx

	// Some other operation
	_, _ = Users.Query(sm.Limit(1)).One(ctx, tx)

	// Batch 2
	batch2 := pgx.NewBatchBuilder()
	// ... add queries
	batch2.Execute(ctx, tx)

	tx.Commit(ctx)
}
