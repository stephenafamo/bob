package bob

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"os"

	"github.com/stephenafamo/scan"
)

// DebugPrinter is used to print queries and arguments
type DebugPrinter interface {
	PrintQuery(query string, args ...any)
}

// an implementtion of the [DebugPrinter]
type writerPrinter struct{ io.Writer }

// implements [DebugPrinter]
func (w writerPrinter) PrintQuery(query string, args ...any) {
	fmt.Fprintln(w.Writer, query)
	for i, arg := range args {
		fmt.Fprintf(w.Writer, "%d: %#v\n", i, arg)
	}
	fmt.Fprintf(w.Writer, "\n")
}

// Debug wraps an [Executor] and prints the queries and args to os.Stdout
func Debug(exec Executor) Preparer {
	return DebugToWriter(exec, nil)
}

// DebugToWriter wraps an existing [Executor] and writes all
// queries and args to the given [io.Writer]
// if w is nil, it fallsback to [os.Stdout]
func DebugToWriter(exec Executor, w io.Writer) Preparer {
	if w == nil {
		w = os.Stdout
	}
	return DebugToPrinter(exec, writerPrinter{w})
}

// DebugToPrinter wraps an existing [Executor] and writes all
// queries and args to the given [DebugPrinter]
// if w is nil, it fallsback to writing to [os.Stdout]
func DebugToPrinter(exec Executor, w DebugPrinter) Preparer {
	if w == nil {
		w = writerPrinter{os.Stdout}
	}
	return debugExecutor{printer: w, exec: exec}
}

type debugExecutor struct {
	printer DebugPrinter
	exec    Executor
}

func (d debugExecutor) PrepareContext(ctx context.Context, query string) (Statement, error) {
	p, ok := d.exec.(Preparer)
	if !ok {
		return nil, fmt.Errorf("executor does not implement Preparer")
	}

	stmt, err := p.PrepareContext(ctx, query)
	if err != nil {
		return nil, err
	}

	return debugStmt{printer: d.printer, stmt: stmt, query: query}, nil
}

func (d debugExecutor) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	d.printer.PrintQuery(query, args...)
	return d.exec.ExecContext(ctx, query, args...)
}

func (d debugExecutor) QueryContext(ctx context.Context, query string, args ...any) (scan.Rows, error) {
	d.printer.PrintQuery(query, args...)
	return d.exec.QueryContext(ctx, query, args...)
}

type debugStmt struct {
	printer DebugPrinter
	stmt    Statement
	query   string
}

func (d debugStmt) Close() error {
	return d.stmt.Close()
}

func (d debugStmt) ExecContext(ctx context.Context, args ...any) (sql.Result, error) {
	d.printer.PrintQuery(d.query, args...)
	return d.stmt.ExecContext(ctx, args...)
}

func (d debugStmt) QueryContext(ctx context.Context, args ...any) (scan.Rows, error) {
	d.printer.PrintQuery(d.query, args...)
	return d.stmt.QueryContext(ctx, args...)
}
