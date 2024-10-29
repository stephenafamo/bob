package psql

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/DATA-DOG/go-txdb"
	"github.com/aarondl/opt/omit"
	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql/dialect"
	"github.com/stephenafamo/bob/dialect/psql/um"
	"github.com/stephenafamo/bob/expr"
	helpers "github.com/stephenafamo/bob/gen/bobgen-helpers"
	"github.com/stephenafamo/bob/orm"
	"github.com/stephenafamo/scan"
)

func TestMain(m *testing.M) {
	port, err := helpers.GetFreePort()
	if err != nil {
		fmt.Printf("could not get a free port: %v\n", err)
		os.Exit(1)
	}

	dbConfig := embeddedpostgres.
		DefaultConfig().
		RuntimePath(filepath.Join(os.TempDir(), "psql_driver")).
		Port(uint32(port)).
		Logger(&bytes.Buffer{})
	dsn := dbConfig.GetConnectionURL() + "?sslmode=disable"

	postgres := embeddedpostgres.NewDatabase(dbConfig)
	if err := postgres.Start(); err != nil {
		fmt.Printf("starting embedded postgres: %v\n", err)
		os.Exit(1)
	}

	txdb.Register("txdb", "pgx", dsn)
	code := m.Run()
	if err := postgres.Stop(); err != nil {
		fmt.Printf("could not stop postgres on port %d: %v\n", port, err)
		os.Exit(1)
	}

	os.Exit(code)
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

func (s UserSetter) UpdateMod() bob.Mod[*dialect.UpdateQuery] {
	return um.Set(s.Expressions()...)
}

func (s UserSetter) Expressions(prefix ...string) []bob.Expression {
	exprs := make([]bob.Expression, 0, 3)

	if !s.ID.IsUnset() {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			Quote(append(prefix, "id")...),
			Arg(s.ID),
		}})
	}

	if !s.Email.IsUnset() {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			Quote(append(prefix, "name")...),
			Arg(s.Name),
		}})
	}

	if !s.Email.IsUnset() {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			Quote(append(prefix, "email")...),
			Arg(s.Email),
		}})
	}

	return exprs
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

	_, err = userTable.Update(UserSetter{
		Name:  omit.From("Stephen"),
		Email: omit.From("stephen@example.com"),
	}.UpdateMod(), um.Where(Quote("id").EQ(Arg(user.ID)))).Exec(ctx, db)
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
		Email: "stephen@example.com",
	}) {
		t.Fatalf("unexpected retrieved user: %#v: %v", *user, err)
	}
}
