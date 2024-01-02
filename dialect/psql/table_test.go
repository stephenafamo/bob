package psql

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-txdb"
	"github.com/aarondl/opt/omit"
	"github.com/jackc/pgx/v5"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql/dialect"
	"github.com/stephenafamo/bob/dialect/psql/um"
	"github.com/stephenafamo/bob/orm"
	"github.com/stephenafamo/scan"
)

var dsn = os.Getenv("PSQL_DIALECT_TEST_DSN")

func TestMain(m *testing.M) {
	if dsn == "" {
		fmt.Printf("No environment variable PSQL_DIALECT_TEST_DSN")
		os.Exit(1)
	}
	// somehow create the DB
	config, err := pgx.ParseConfig(dsn)
	if err != nil {
		fmt.Printf("could not parse dsn: %v", err)
		os.Exit(1)
	}

	if !strings.Contains(config.Database, "droppable") {
		fmt.Printf("database name %q must contain %q to ensure that data is not lost", config.Database, "droppable")
		os.Exit(1)
	}

	txdb.Register("txdb", "pgx", dsn)

	os.Exit(m.Run())
}

type User struct {
	ID    int64  `db:"id,pk"`
	Name  string `db:"name"`
	Email string `db:"email"`
}

func (u *User) PrimaryKeyVals() bob.Expression {
	return Arg(u.ID)
}

type UserSetter struct {
	ID    omit.Val[int64]  `db:"id,pk"`
	Name  omit.Val[string] `db:"name"`
	Email omit.Val[string] `db:"email"`

	orm.Setter[*User, *dialect.InsertQuery, *dialect.UpdateQuery]
}

func (s UserSetter) Overwrite(t *User) {
	if !s.ID.IsUnset() {
		t.ID, _ = s.ID.Get()
	}

	if !s.Name.IsUnset() {
		t.Name, _ = s.Name.Get()
	}

	if !s.Email.IsUnset() {
		t.Email, _ = s.Email.Get()
	}
}

func (s UserSetter) Apply(q *dialect.UpdateQuery) {
	if !s.ID.IsUnset() {
		um.SetCol("id").ToArg(s.ID).Apply(q)
	}

	if !s.Name.IsUnset() {
		um.SetCol("name").ToArg(s.Name).Apply(q)
	}

	if !s.Email.IsUnset() {
		um.SetCol("email").ToArg(s.Email).Apply(q)
	}
}

var userTable = NewTable[*User, *UserSetter]("", "users")

func TestUpdate(t *testing.T) {
	ctx := context.Background()
	db, err := bob.Open("txdb", "TestUpdate")
	if err != nil {
		t.Fatalf("could not open database connection: %v", err)
	}
	defer db.Close()

	_, err = db.ExecContext(ctx, `CREATE TABLE users (
		id INTEGER,
		name TEXT NOT NULL,
		email TEXT UNIQUE NOT NULL
	)`)
	if err != nil {
		t.Fatalf("could not create users table: %v", err)
	}

	q := "INSERT INTO users (id, name, email) values ($1, '', '') RETURNING *"
	user, err := scan.One(ctx, db, scan.StructMapper[*User](), q, 1)
	if err != nil {
		t.Fatalf("could not insert user: %v", err)
	}

	if *user != (User{ID: 1}) {
		t.Fatalf("unexpected inserted user: %v", err)
	}

	err = userTable.Update(ctx, db, &UserSetter{
		Name:  omit.From("Stephen"),
		Email: omit.From("stephen@exapmle.com"),
	}, user)
	if err != nil {
		t.Errorf("error updating: %v", err)
	}

	q = "SELECT * FROM users WHERE id = $1"
	user, err = scan.One(ctx, db, scan.StructMapper[*User](), q, 1)
	if err != nil {
		t.Fatalf("could not get user: %v", err)
	}

	if *user != (User{
		ID:    1,
		Name:  "Stephen",
		Email: "stephen@exapmle.com",
	}) {
		t.Fatalf("unexpected retrieved user: %v", err)
	}
}
