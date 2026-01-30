package psql

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"syscall"
	"testing"

	_ "github.com/lib/pq"
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql/dialect"
	"github.com/stephenafamo/bob/dialect/psql/mm"
	"github.com/stephenafamo/bob/dialect/psql/um"
	"github.com/stephenafamo/bob/expr"
	"github.com/stephenafamo/bob/internal"
	"github.com/stephenafamo/bob/orm"
	"github.com/stephenafamo/scan"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

var testDB bob.DB

func TestMain(m *testing.M) {
	code := 1
	defer func() {
		os.Exit(code)
	}()

	ctx, cancel := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT,
		syscall.SIGTERM,
	)
	defer cancel()

	postgresContainer, err := postgres.Run(
		ctx, "pgvector/pgvector:0.8.0-pg16",
		postgres.BasicWaitStrategies(),
		testcontainers.WithLogger(log.New(io.Discard, "", log.LstdFlags)),
	)
	if err != nil {
		fmt.Printf("could not start postgres container: %v\n", err)
		return
	}
	defer func() {
		if err := testcontainers.TerminateContainer(postgresContainer); err != nil {
			log.Printf("failed to terminate container: %s", err)
		}
	}()

	dsn, err := postgresContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		fmt.Printf("could not get connection string: %v\n", err)
		return
	}

	testDB, err = bob.Open("postgres", dsn)
	if err != nil {
		fmt.Printf("could not connect to db: %v\n", err)
		return
	}
	defer testDB.Close()

	code = m.Run()
}

type User struct {
	ID    int64  `db:"id,pk"`
	Name  string `db:"name"`
	Email string `db:"email"`
}

type UserSetter struct {
	ID    *int64  `db:"id,pk"`
	Name  *string `db:"name"`
	Email *string `db:"email"`

	orm.Setter[*User, *dialect.InsertQuery, *dialect.UpdateQuery]
}

func (s UserSetter) Overwrite(t *User) {
	if s.ID != nil {
		t.ID = *s.ID
	}

	if s.Name != nil {
		t.Name = *s.Name
	}

	if s.Email != nil {
		t.Email = *s.Email
	}
}

func (s UserSetter) UpdateMod() bob.Mod[*dialect.UpdateQuery] {
	return um.Set(s.Expressions()...)
}

func (s UserSetter) Expressions(prefix ...string) []bob.Expression {
	exprs := make([]bob.Expression, 0, 3)

	if s.ID != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			Quote(append(prefix, "id")...),
			Arg(s.ID),
		}})
	}

	if s.Email != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			Quote(append(prefix, "name")...),
			Arg(s.Name),
		}})
	}

	if s.Email != nil {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			Quote(append(prefix, "email")...),
			Arg(s.Email),
		}})
	}

	return exprs
}

var userTable = NewTable[*User, *UserSetter, bob.Expression]("", "users", expr.ColsForStruct[User]("users"))

func TestUpdate(t *testing.T) {
	ctx := t.Context()

	tx, err := testDB.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("could not begin transaction: %v", err)
		return
	}

	_, err = tx.ExecContext(ctx, `CREATE TABLE users (
		id INTEGER,
		name TEXT NOT NULL,
		email TEXT UNIQUE NOT NULL
	)`)
	if err != nil {
		t.Fatalf("could not create users table: %v", err)
	}

	q := "INSERT INTO users (id, name, email) values ($1, '', '') RETURNING *"
	user, err := scan.One(ctx, tx, scan.StructMapper[*User](), q, 1)
	if err != nil {
		t.Fatalf("could not insert user: %v", err)
	}

	if *user != (User{ID: 1}) {
		t.Fatalf("unexpected inserted user: %v", err)
	}

	_, err = userTable.Update(UserSetter{
		Name:  internal.Pointer("Stephen"),
		Email: internal.Pointer("stephen@example.com"),
	}.UpdateMod(), um.Where(Quote("id").EQ(Arg(user.ID)))).Exec(ctx, tx)
	if err != nil {
		t.Errorf("error updating: %v", err)
	}

	q = "SELECT * FROM users WHERE id = $1"
	user, err = scan.One(ctx, tx, scan.StructMapper[*User](), q, 1)
	if err != nil {
		t.Fatalf("could not get user: %v", err)
	}

	if *user != (User{
		ID:    1,
		Name:  "Stephen",
		Email: "stephen@example.com",
	}) {
		t.Fatalf("unexpected retrieved user: %#v: %v", *user, err)
	}
}

func TestMerge(t *testing.T) {
	ctx := t.Context()

	tx, err := testDB.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("could not begin transaction: %v", err)
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Create users table
	_, err = tx.ExecContext(ctx, `CREATE TABLE users (
		id INTEGER PRIMARY KEY,
		name TEXT NOT NULL,
		email TEXT UNIQUE NOT NULL
	)`)
	if err != nil {
		t.Fatalf("could not create users table: %v", err)
	}

	// Create source table for merge
	_, err = tx.ExecContext(ctx, `CREATE TABLE user_updates (
		id INTEGER PRIMARY KEY,
		name TEXT NOT NULL,
		email TEXT NOT NULL
	)`)
	if err != nil {
		t.Fatalf("could not create user_updates table: %v", err)
	}

	// Insert initial user
	_, err = tx.ExecContext(ctx, `INSERT INTO users (id, name, email) VALUES (1, 'Alice', 'alice@example.com')`)
	if err != nil {
		t.Fatalf("could not insert user: %v", err)
	}

	// Insert updates (one existing, one new)
	_, err = tx.ExecContext(ctx, `INSERT INTO user_updates (id, name, email) VALUES 
		(1, 'Alice Smith', 'alice.smith@example.com'),
		(2, 'Bob', 'bob@example.com')`)
	if err != nil {
		t.Fatalf("could not insert user_updates: %v", err)
	}

	// Execute MERGE using table's Merge method
	mergeQuery := userTable.Merge(
		mm.Using("user_updates").As("u").On(
			Quote("u", "id").EQ(Quote("users", "id")),
		),
		mm.WhenMatched(
			mm.ThenUpdate(
				mm.SetCol("name").ToExpr(Quote("u", "name")),
				mm.SetCol("email").ToExpr(Quote("u", "email")),
			),
		),
		mm.WhenNotMatched(
			mm.ThenInsert(
				mm.Columns("id", "name", "email"),
				mm.Values(Quote("u", "id"), Quote("u", "name"), Quote("u", "email")),
			),
		),
	)

	// Get the SQL for debugging
	sql, args, err := bob.Build(ctx, mergeQuery)
	if err != nil {
		t.Fatalf("could not build merge query: %v", err)
	}
	t.Logf("MERGE SQL: %s", sql)
	t.Logf("MERGE Args: %v", args)

	// Execute the merge
	_, err = mergeQuery.Exec(ctx, tx)
	if err != nil {
		t.Fatalf("could not execute merge: %v", err)
	}

	// Verify user 1 was updated
	q := "SELECT * FROM users WHERE id = $1"
	user, err := scan.One(ctx, tx, scan.StructMapper[*User](), q, 1)
	if err != nil {
		t.Fatalf("could not get user 1: %v", err)
	}

	if *user != (User{
		ID:    1,
		Name:  "Alice Smith",
		Email: "alice.smith@example.com",
	}) {
		t.Errorf("unexpected user 1 after merge: %#v", *user)
	}

	// Verify user 2 was inserted
	user, err = scan.One(ctx, tx, scan.StructMapper[*User](), q, 2)
	if err != nil {
		t.Fatalf("could not get user 2: %v", err)
	}

	if *user != (User{
		ID:    2,
		Name:  "Bob",
		Email: "bob@example.com",
	}) {
		t.Errorf("unexpected user 2 after merge: %#v", *user)
	}

	// Verify total count
	var count int
	err = tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil {
		t.Fatalf("could not count users: %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2 users, got %d", count)
	}
}
