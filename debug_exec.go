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
	return debugExecutor{w: w, exec: exec}
}

type debugExecutor struct {
	w    io.Writer
	exec Executor
}

func (d debugExecutor) print(query string, args ...any) {
	fmt.Fprintln(d.w, query)
	for i, arg := range args {
		fmt.Fprintf(d.w, "%d: %#v\n", i, arg)
	}
	fmt.Fprintf(d.w, "\n")
}

func (d debugExecutor) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	d.print(query, args...)
	return d.exec.ExecContext(ctx, query, args...)
}

func (d debugExecutor) QueryContext(ctx context.Context, query string, args ...any) (scan.Rows, error) {
	d.print(query, args...)
	return d.exec.QueryContext(ctx, query, args...)
}
