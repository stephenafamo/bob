package bob

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"os"

	"github.com/stephenafamo/scan"
)

// DebugExecutor wraps an existing [Executor] and writes all
// queries and args to the given [io.Writer]
// if w is nil, it fallsback to [os.Stdout]
func DebugExecutor(exec Executor, w io.Writer) Executor {
	if w == nil {
		w = os.Stdout
	}
	return debugQueryer{w: w, exec: exec}
}

type debugQueryer struct {
	w    io.Writer
	exec Executor
}

func (d debugQueryer) print(query string, args ...any) {
	fmt.Fprintln(d.w, query)
	for i, arg := range args {
		fmt.Fprintf(d.w, "%d: %#v\n", i, arg)
	}
	fmt.Fprintf(d.w, "\n")
}

func (d debugQueryer) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	d.print(query, args...)
	return d.exec.ExecContext(ctx, query, args...)
}

func (d debugQueryer) QueryContext(ctx context.Context, query string, args ...any) (scan.Rows, error) {
	d.print(query, args...)
	return d.exec.QueryContext(ctx, query, args...)
}
