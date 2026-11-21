package pgx

import (
	"context"
	"io"
	"log"
	"testing"

	"github.com/aarondl/opt/null"
	"github.com/jackc/pgx/v5"
	"github.com/stephenafamo/bob/types/pgtypes"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

// TestNativePgxTSVector round-trips the types bobgen-psql generates for
// tsvector columns with the native pgx driver: pgtypes.TSVector for NOT NULL
// columns and null.Val[pgtypes.TSVector] for nullable ones. A plain string
// does not work from pgx v5.9, which decodes tsvector with a binary codec.
func TestNativePgxTSVector(t *testing.T) {
	ctx := context.Background()
	container, err := postgres.Run(ctx, "pgvector/pgvector:0.8.0-pg16",
		postgres.BasicWaitStrategies(),
		testcontainers.WithLogger(log.New(io.Discard, "", log.LstdFlags)),
	)
	if err != nil {
		t.Skipf("could not start postgres container: %v", err)
	}
	t.Cleanup(func() {
		if err := testcontainers.TerminateContainer(container); err != nil {
			t.Logf("failed to terminate container: %s", err)
		}
	})

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}

	pgxConn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { pgxConn.Close(ctx) })
	db := NewConn(pgxConn)

	if _, err := db.ExecContext(ctx, `CREATE TABLE tsv (id int PRIMARY KEY, v tsvector NOT NULL, n tsvector)`); err != nil {
		t.Fatal(err)
	}

	var written pgtypes.TSVector
	if err := written.Scan("'cat':3A 'fat':2,4 'rat':5B"); err != nil {
		t.Fatal(err)
	}
	want := written.String()

	if _, err := db.ExecContext(ctx,
		`INSERT INTO tsv (id, v, n) VALUES (1, $1, $2), (2, $1, NULL)`,
		written, null.From(written),
	); err != nil {
		t.Fatalf("insert: %v", err)
	}

	t.Run("NOT NULL", func(t *testing.T) {
		var got pgtypes.TSVector
		queryRow(t, db, `SELECT v FROM tsv WHERE id = 1`, &got)
		if got.String() != want {
			t.Fatalf("got %q, want %q", got.String(), want)
		}
	})

	t.Run("null.Val", func(t *testing.T) {
		var got null.Val[pgtypes.TSVector]
		queryRow(t, db, `SELECT n FROM tsv WHERE id = 1`, &got)
		if got.MustGet().String() != want {
			t.Fatalf("got %q, want %q", got.MustGet().String(), want)
		}
	})

	t.Run("null.Val NULL", func(t *testing.T) {
		var got null.Val[pgtypes.TSVector]
		queryRow(t, db, `SELECT n FROM tsv WHERE id = 2`, &got)
		if !got.IsNull() {
			t.Fatalf("expected NULL, got %q", got.MustGet().String())
		}
	})
}

func queryRow(t *testing.T, db Conn, query string, dest any) {
	t.Helper()

	rows, err := db.QueryContext(context.Background(), query)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	if !rows.Next() {
		t.Fatalf("no rows: %v", rows.Err())
	}
	if err := rows.Scan(dest); err != nil {
		t.Fatal(err)
	}
}
