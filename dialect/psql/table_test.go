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

var userTable = NewTable[*User, *UserSetter, bob.Expression]("", "users", expr.ColsForStruct[User](""), expr.ColsForStruct[User]("users"))

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
