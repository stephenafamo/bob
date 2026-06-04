package bob

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"testing"
)

var (
	_ Preparer[StdPrepared] = DB{}
	_ Executor              = DB{}
	_ Transactor[Tx]        = DB{}
)

var (
	_ Preparer[StdPrepared] = Conn{}
	_ Executor              = Conn{}
	_ Transactor[Tx]        = Conn{}
)

var (
	_ Preparer[StdPrepared]  = Tx{}
	_ Executor               = Tx{}
	_ Transaction            = Tx{}
	_ txForStmt[StdPrepared] = Tx{}
)

// ─── mock driver ─────────────────────────────────────────────────────────────

type mockConnector struct{ rollbackErr error }

func (c mockConnector) Connect(_ context.Context) (driver.Conn, error) {
	return mockConn{rollbackErr: c.rollbackErr}, nil
}
func (c mockConnector) Driver() driver.Driver { return nil }

type mockConn struct{ rollbackErr error }

func (c mockConn) Prepare(_ string) (driver.Stmt, error) { return nil, nil }
func (c mockConn) Close() error                           { return nil }
func (c mockConn) Begin() (driver.Tx, error)              { return mockTx{rollbackErr: c.rollbackErr}, nil }

type mockTx struct{ rollbackErr error }

func (t mockTx) Commit() error   { return nil }
func (t mockTx) Rollback() error { return t.rollbackErr }

func newMockDB(rollbackErr error) DB {
	return NewDB(sql.OpenDB(mockConnector{rollbackErr: rollbackErr}))
}

// ─── RunInTx panic tests ──────────────────────────────────────────────────────

// TestRunInTxPanicPropagates verifies that a panic inside fn is re-raised after
// the transaction has been rolled back.
func TestRunInTxPanicPropagates(t *testing.T) {
	t.Parallel()

	panicValue := errors.New("forced panic")

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic to propagate, but it did not")
		}
		if r != panicValue {
			t.Fatalf("got panic value %v, want %v", r, panicValue)
		}
	}()

	_ = newMockDB(nil).RunInTx(context.Background(), nil, func(_ context.Context, _ Executor) error {
		panic(panicValue)
	})

	t.Fatal("RunInTx should not return normally after a panic")
}

// TestRunInTxPanicErrorJoinsRollbackErr verifies that when fn panics with an
// error value and rollback also fails, both errors are joined and re-panicked.
func TestRunInTxPanicErrorJoinsRollbackErr(t *testing.T) {
	t.Parallel()

	panicErr := errors.New("forced panic")
	rollbackErr := errors.New("rollback failed")

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic to propagate, but it did not")
		}
		joined, ok := r.(error)
		if !ok {
			t.Fatalf("expected panic value to be an error, got %T", r)
		}
		if !errors.Is(joined, panicErr) {
			t.Errorf("joined error should wrap panicErr: %v", joined)
		}
		if !errors.Is(joined, rollbackErr) {
			t.Errorf("joined error should wrap rollbackErr: %v", joined)
		}
	}()

	_ = newMockDB(rollbackErr).RunInTx(context.Background(), nil, func(_ context.Context, _ Executor) error {
		panic(panicErr)
	})

	t.Fatal("RunInTx should not return normally after a panic")
}

// TestRunInTxPanicNonErrorPreservesValue verifies that when fn panics with a
// non-error value, the original panic value is preserved (even if rollback fails).
func TestRunInTxPanicNonErrorPreservesValue(t *testing.T) {
	t.Parallel()

	const panicValue = "non-error panic"

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic to propagate, but it did not")
		}
		if r != panicValue {
			t.Fatalf("got panic value %v, want %v", r, panicValue)
		}
	}()

	_ = newMockDB(nil).RunInTx(context.Background(), nil, func(_ context.Context, _ Executor) error {
		panic(panicValue)
	})

	t.Fatal("RunInTx should not return normally after a panic")
}
