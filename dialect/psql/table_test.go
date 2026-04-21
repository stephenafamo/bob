package psql

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"testing"

	_ "github.com/lib/pq"
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql/dialect"
	"github.com/stephenafamo/bob/dialect/psql/dm"
	"github.com/stephenafamo/bob/dialect/psql/im"
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

func TestTableUpdateDefaultsReturningAllColumns(t *testing.T) {
	q := userTable.Update(
		um.SetCol("name").ToArg("Stephen"),
		um.Where(Quote("id").EQ(Arg(1))),
	)

	sql, args, err := q.Build(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	expectedSQL := "UPDATE \"users\" AS \"users\" SET\n\"name\" = $1\nWHERE (\"id\" = $2)\nRETURNING \"users\".\"id\" AS \"id\", \"users\".\"name\" AS \"name\", \"users\".\"email\" AS \"email\""
	if sql != expectedSQL {
		t.Fatalf("unexpected SQL: %#v", sql)
	}
	if len(args) != 2 {
		t.Fatalf("unexpected arg count: %d", len(args))
	}
}

func TestTableUpdateExplicitReturningOverridesDefault(t *testing.T) {
	q := userTable.Update(
		um.SetCol("name").ToArg("Stephen"),
		um.Where(Quote("id").EQ(Arg(1))),
		um.Returning("id"),
	)

	sql, args, err := q.Build(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	expectedSQL := "UPDATE \"users\" AS \"users\" SET\n\"name\" = $1\nWHERE (\"id\" = $2)\nRETURNING id"
	if sql != expectedSQL {
		t.Fatalf("unexpected SQL: %#v", sql)
	}
	if len(args) != 2 {
		t.Fatalf("unexpected arg count: %d", len(args))
	}
}

func TestTableUpdateAdditionalExplicitReturningAppends(t *testing.T) {
	base := userTable.Update(
		um.SetCol("name").ToArg("Stephen"),
		um.Where(Quote("id").EQ(Arg(1))),
	)

	q := base.With(um.Returning("id")).With(um.Returning("email"))

	sql, args, err := q.Build(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	expectedSQL := "UPDATE \"users\" AS \"users\" SET\n\"name\" = $1\nWHERE (\"id\" = $2)\nRETURNING id, email"
	if sql != expectedSQL {
		t.Fatalf("unexpected SQL: %#v", sql)
	}
	if len(args) != 2 {
		t.Fatalf("unexpected arg count: %d", len(args))
	}
}

func TestTableInsertDefaultsReturningAllColumns(t *testing.T) {
	q := userTable.Insert(
		im.Rows([]bob.Expression{Arg(int64(1)), Arg("Stephen"), Arg("stephen@example.com")}),
	)

	sql, args, err := q.Build(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	expectedSQL := "INSERT INTO \"users\" AS \"users\"(\"id\", \"name\", \"email\")\nVALUES ($1, $2, $3)\nRETURNING \"users\".\"id\" AS \"id\", \"users\".\"name\" AS \"name\", \"users\".\"email\" AS \"email\"\n"
	if sql != expectedSQL {
		t.Fatalf("unexpected SQL: %#v", sql)
	}
	if len(args) != 3 {
		t.Fatalf("unexpected arg count: %d", len(args))
	}
}

func TestTableInsertExplicitReturningOverridesDefault(t *testing.T) {
	q := userTable.Insert(
		im.Rows([]bob.Expression{Arg(int64(1)), Arg("Stephen"), Arg("stephen@example.com")}),
		im.Returning("id"),
	)

	sql, args, err := q.Build(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	expectedSQL := "INSERT INTO \"users\" AS \"users\"(\"id\", \"name\", \"email\")\nVALUES ($1, $2, $3)\nRETURNING id\n"
	if sql != expectedSQL {
		t.Fatalf("unexpected SQL: %#v", sql)
	}
	if len(args) != 3 {
		t.Fatalf("unexpected arg count: %d", len(args))
	}
}

func TestTableInsertAdditionalExplicitReturningAppends(t *testing.T) {
	base := userTable.Insert(
		im.Rows([]bob.Expression{Arg(int64(1)), Arg("Stephen"), Arg("stephen@example.com")}),
	)

	q := base.With(im.Returning("id")).With(im.Returning("email"))

	sql, args, err := q.Build(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	expectedSQL := "INSERT INTO \"users\" AS \"users\"(\"id\", \"name\", \"email\")\nVALUES ($1, $2, $3)\nRETURNING id, email\n"
	if sql != expectedSQL {
		t.Fatalf("unexpected SQL: %#v", sql)
	}
	if len(args) != 3 {
		t.Fatalf("unexpected arg count: %d", len(args))
	}
}

func TestTableDeleteDefaultsReturningAllColumns(t *testing.T) {
	q := userTable.Delete(
		dm.Where(Quote("id").EQ(Arg(1))),
	)

	sql, args, err := q.Build(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	expectedSQL := "DELETE FROM \"users\" AS \"users\"\nWHERE (\"id\" = $1)\nRETURNING \"users\".\"id\" AS \"id\", \"users\".\"name\" AS \"name\", \"users\".\"email\" AS \"email\""
	if sql != expectedSQL {
		t.Fatalf("unexpected SQL: %#v", sql)
	}
	if len(args) != 1 {
		t.Fatalf("unexpected arg count: %d", len(args))
	}
}

func TestTableDeleteExplicitReturningOverridesDefault(t *testing.T) {
	q := userTable.Delete(
		dm.Where(Quote("id").EQ(Arg(1))),
		dm.Returning("id"),
	)

	sql, args, err := q.Build(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	expectedSQL := "DELETE FROM \"users\" AS \"users\"\nWHERE (\"id\" = $1)\nRETURNING id"
	if sql != expectedSQL {
		t.Fatalf("unexpected SQL: %#v", sql)
	}
	if len(args) != 1 {
		t.Fatalf("unexpected arg count: %d", len(args))
	}
}

func TestTableDeleteAdditionalExplicitReturningAppends(t *testing.T) {
	base := userTable.Delete(
		dm.Where(Quote("id").EQ(Arg(1))),
	)

	q := base.With(dm.Returning("id")).With(dm.Returning("email"))

	sql, args, err := q.Build(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	expectedSQL := "DELETE FROM \"users\" AS \"users\"\nWHERE (\"id\" = $1)\nRETURNING id, email"
	if sql != expectedSQL {
		t.Fatalf("unexpected SQL: %#v", sql)
	}
	if len(args) != 1 {
		t.Fatalf("unexpected arg count: %d", len(args))
	}
}

func TestTableUpdateApplyDoesNotMutateOriginal(t *testing.T) {
	base := userTable.Update(
		um.SetCol("name").ToArg("Stephen"),
	)

	derived := base.Apply(
		um.Where(Quote("id").EQ(Arg(1))),
	)

	baseSQL, _, err := base.Build(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	derivedSQL, _, err := derived.Build(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	expectedBase := "UPDATE \"users\" AS \"users\" SET\n\"name\" = $1\nRETURNING \"users\".\"id\" AS \"id\", \"users\".\"name\" AS \"name\", \"users\".\"email\" AS \"email\""
	if baseSQL != expectedBase {
		t.Fatalf("unexpected base SQL: %#v", baseSQL)
	}

	expectedDerived := "UPDATE \"users\" AS \"users\" SET\n\"name\" = $1\nWHERE (\"id\" = $2)\nRETURNING \"users\".\"id\" AS \"id\", \"users\".\"name\" AS \"name\", \"users\".\"email\" AS \"email\""
	if derivedSQL != expectedDerived {
		t.Fatalf("unexpected derived SQL: %#v", derivedSQL)
	}
}

func TestTableInsertWithDoesNotMutateOriginal(t *testing.T) {
	base := userTable.Insert(
		im.Rows([]bob.Expression{Arg(int64(1)), Arg("Stephen"), Arg("stephen@example.com")}),
	)

	derived := base.With(im.Returning("id"))

	baseSQL, _, err := base.Build(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	derivedSQL, _, err := derived.Build(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	expectedBase := "INSERT INTO \"users\" AS \"users\"(\"id\", \"name\", \"email\")\nVALUES ($1, $2, $3)\nRETURNING \"users\".\"id\" AS \"id\", \"users\".\"name\" AS \"name\", \"users\".\"email\" AS \"email\"\n"
	if baseSQL != expectedBase {
		t.Fatalf("unexpected base SQL: %#v", baseSQL)
	}

	expectedDerived := "INSERT INTO \"users\" AS \"users\"(\"id\", \"name\", \"email\")\nVALUES ($1, $2, $3)\nRETURNING id\n"
	if derivedSQL != expectedDerived {
		t.Fatalf("unexpected derived SQL: %#v", derivedSQL)
	}
}

func TestTableDeleteApplyDoesNotMutateOriginal(t *testing.T) {
	base := userTable.Delete(
		dm.Where(Quote("email").EQ(Arg("stephen@example.com"))),
	)

	derived := base.Apply(
		dm.Returning("id"),
	)

	baseSQL, _, err := base.Build(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	derivedSQL, _, err := derived.Build(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	expectedBase := "DELETE FROM \"users\" AS \"users\"\nWHERE (\"email\" = $1)\nRETURNING \"users\".\"id\" AS \"id\", \"users\".\"name\" AS \"name\", \"users\".\"email\" AS \"email\""
	if baseSQL != expectedBase {
		t.Fatalf("unexpected base SQL: %#v", baseSQL)
	}

	expectedDerived := "DELETE FROM \"users\" AS \"users\"\nWHERE (\"email\" = $1)\nRETURNING id"
	if derivedSQL != expectedDerived {
		t.Fatalf("unexpected derived SQL: %#v", derivedSQL)
	}
}

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
		mm.WhenMatched().ThenUpdate(
			mm.SetCol("name").To(Quote("u", "name")),
			mm.SetCol("email").To(Quote("u", "email")),
		),
		mm.WhenNotMatched().ThenInsert(
			mm.Columns("id", "name", "email"),
			mm.Values(Quote("u", "id"), Quote("u", "name"), Quote("u", "email")),
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

func TestTableMergeWithVersion(t *testing.T) {
	// Use the existing userTable from the test file

	t.Run("version 17+ adds RETURNING automatically", func(t *testing.T) {
		ctx := context.Background()
		ctx = SetVersion(ctx, 17)

		mergeQuery := userTable.Merge(
			mm.Using("source").As("s").On(
				Quote("s", "id").EQ(Quote("users", "id")),
			),
			mm.WhenMatched().ThenUpdate(
				mm.SetCol("name").To(Quote("s", "name")),
			),
		)

		sql, args, err := bob.Build(ctx, mergeQuery)
		if err != nil {
			t.Fatalf("error: %v", err)
		}

		// Should contain RETURNING because version is 17+
		if !strings.Contains(sql, "RETURNING") {
			t.Errorf("expected RETURNING clause for version 17+, got: %s", sql)
		}
		if len(args) != 0 {
			t.Errorf("expected no args, got %v", args)
		}
	})

	t.Run("version below 17 does not add RETURNING automatically", func(t *testing.T) {
		ctx := context.Background()
		ctx = SetVersion(ctx, 16)

		mergeQuery := userTable.Merge(
			mm.Using("source").As("s").On(
				Quote("s", "id").EQ(Quote("users", "id")),
			),
			mm.WhenMatched().ThenUpdate(
				mm.SetCol("name").To(Quote("s", "name")),
			),
		)

		sql, args, err := bob.Build(ctx, mergeQuery)
		if err != nil {
			t.Fatalf("error: %v", err)
		}

		// Should NOT contain RETURNING because version is below 17
		if strings.Contains(sql, "RETURNING") {
			t.Errorf("expected no RETURNING clause for version 16, got: %s", sql)
		}
		if len(args) != 0 {
			t.Errorf("expected no args, got %v", args)
		}
	})

	t.Run("no version set does not add RETURNING automatically", func(t *testing.T) {
		ctx := context.Background()
		// No version set

		mergeQuery := userTable.Merge(
			mm.Using("source").As("s").On(
				Quote("s", "id").EQ(Quote("users", "id")),
			),
			mm.WhenMatched().ThenUpdate(
				mm.SetCol("name").To(Quote("s", "name")),
			),
		)

		sql, args, err := bob.Build(ctx, mergeQuery)
		if err != nil {
			t.Fatalf("error: %v", err)
		}

		// Should NOT contain RETURNING because no version set
		if strings.Contains(sql, "RETURNING") {
			t.Errorf("expected no RETURNING clause when version not set, got: %s", sql)
		}
		if len(args) != 0 {
			t.Errorf("expected no args, got %v", args)
		}
	})
}
