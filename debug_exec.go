package bob

import (
	"context"
	"database/sql"
	"database/sql/driver"
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
		val := arg
		if valuer, ok := val.(driver.Valuer); ok {
			val, _ = valuer.Value()
		}
		fmt.Fprintf(w.Writer, "%d: %T: %v\n", i, arg, val)
	}
	fmt.Fprintf(w.Writer, "\n")
}

// Debug wraps an [Executor] and prints the queries and args to os.Stdout
func Debug(exec Executor) Executor {
	return DebugToWriter(exec, nil)
}

// DebugToWriter wraps an existing [Executor] and writes all
// queries and args to the given [io.Writer]
// if w is nil, it fallsback to [os.Stdout]
func DebugToWriter(exec Executor, w io.Writer) Executor {
	if w == nil {
		w = os.Stdout
	}
	return DebugToPrinter(exec, writerPrinter{w})
}

// DebugToPrinter wraps an existing [Executor] and writes all
// queries and args to the given [DebugPrinter]
// if w is nil, it fallsback to writing to [os.Stdout]
func DebugToPrinter(exec Executor, w DebugPrinter) Executor {
	if w == nil {
		w = writerPrinter{os.Stdout}
	}
	return debugExecutor{printer: w, exec: exec}
}

type debugExecutor struct {
	printer DebugPrinter
	exec    Executor
}

func (d debugExecutor) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	d.printer.PrintQuery(query, args...)
	return d.exec.ExecContext(ctx, query, args...)
}

func (d debugExecutor) QueryContext(ctx context.Context, query string, args ...any) (scan.Rows, error) {
	d.printer.PrintQuery(query, args...)
	return d.exec.QueryContext(ctx, query, args...)
}
