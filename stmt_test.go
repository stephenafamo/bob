package bob

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"testing"

	"github.com/stephenafamo/scan"
)

// fakeRows is a scan.Rows that records whether Close was called.
type fakeRows struct {
	vals    []int
	idx     int
	closed  bool
	scanErr error
}

func (r *fakeRows) Columns() ([]string, error) { return []string{"id"}, nil }
func (r *fakeRows) Next() bool                 { return r.idx < len(r.vals) }
func (r *fakeRows) Err() error                 { return nil }
func (r *fakeRows) Close() error               { r.closed = true; return nil }

func (r *fakeRows) Scan(dest ...any) error {
	if r.scanErr != nil {
		return r.scanErr
	}
	for _, d := range dest {
		if p, ok := d.(*int); ok {
			*p = r.vals[r.idx]
		}
	}
	r.idx++
	return nil
}

// fakeStmt is a PreparedExecutor returning a single fakeRows.
type fakeStmt struct {
	rows *fakeRows
}

func (s fakeStmt) ExecContext(ctx context.Context, args ...any) (sql.Result, error) {
	return nil, nil
}

func (s fakeStmt) QueryContext(ctx context.Context, args ...any) (scan.Rows, error) {
	return s.rows, nil
}

func (s fakeStmt) Close() error { return nil }

// fakePreparer implements Preparer[fakeStmt].
type fakePreparer struct {
	rows *fakeRows
}

func (p fakePreparer) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return nil, nil
}

func (p fakePreparer) QueryContext(ctx context.Context, query string, args ...any) (scan.Rows, error) {
	return p.rows, nil
}

func (p fakePreparer) PrepareContext(ctx context.Context, query string) (fakeStmt, error) {
	return fakeStmt{rows: p.rows}, nil
}

type testStmtQuery struct{}

func (testStmtQuery) WriteSQL(ctx context.Context, w io.StringWriter, d Dialect, start int) ([]any, error) {
	w.WriteString("SELECT 1")
	return nil, nil
}

func (testStmtQuery) WriteQuery(ctx context.Context, w io.StringWriter, start int) ([]any, error) {
	w.WriteString("SELECT 1")
	return nil, nil
}

func (testStmtQuery) Type() QueryType { return QueryTypeSelect }

var testIntMapper = scan.Mapper[int](func(ctx context.Context, cols []string) (scan.BeforeFunc, func(any) (int, error)) {
	return func(r *scan.Row) (any, error) {
			v := new(int)
			r.ScheduleScanByName("id", v)
			return v, nil
		}, func(link any) (int, error) {
			return *(link.(*int)), nil
		}
})

func prepareIntStmt(t *testing.T, rows *fakeRows) QueryStmt[int, int, []int] {
	t.Helper()

	stmt, err := PrepareQueryx[int, fakeStmt, int, []int](
		context.Background(), fakePreparer{rows: rows}, testStmtQuery{}, testIntMapper,
	)
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}

	return stmt
}

func TestQueryStmtOneClosesRows(t *testing.T) {
	rows := &fakeRows{vals: []int{42}}
	stmt := prepareIntStmt(t, rows)

	got, err := stmt.One(context.Background(), 0)
	if err != nil {
		t.Fatalf("one: %v", err)
	}
	if got != 42 {
		t.Fatalf("expected 42, got %d", got)
	}
	if !rows.closed {
		t.Fatal("One did not close rows")
	}
}

func TestQueryStmtOneClosesRowsOnNoRows(t *testing.T) {
	rows := &fakeRows{}
	stmt := prepareIntStmt(t, rows)

	if _, err := stmt.One(context.Background(), 0); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("expected sql.ErrNoRows, got %v", err)
	}
	if !rows.closed {
		t.Fatal("One did not close rows on the no-rows path")
	}
}

func TestQueryStmtAllClosesRows(t *testing.T) {
	rows := &fakeRows{vals: []int{1, 2, 3}}
	stmt := prepareIntStmt(t, rows)

	got, err := stmt.All(context.Background(), 0)
	if err != nil {
		t.Fatalf("all: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(got))
	}
	if !rows.closed {
		t.Fatal("All did not close rows")
	}
}

func TestQueryStmtAllClosesRowsOnScanError(t *testing.T) {
	rows := &fakeRows{vals: []int{1}, scanErr: errors.New("scan failed")}
	stmt := prepareIntStmt(t, rows)

	if _, err := stmt.All(context.Background(), 0); err == nil {
		t.Fatal("expected scan error")
	}
	if !rows.closed {
		t.Fatal("All did not close rows on the scan-error path")
	}
}
