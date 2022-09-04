package bob

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/stephenafamo/scan"
)

func DebugExecutor(q Executor) Executor {
	return debugQueryer{w: q}
}

type debugQueryer struct {
	w Executor
}

func (d debugQueryer) print(query string, args ...any) {
	fmt.Println(query)
	for i, arg := range args {
		fmt.Printf("%d: %#v\n", i, arg)
	}
	fmt.Printf("\n")
}

func (d debugQueryer) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	d.print(query, args...)
	return d.w.ExecContext(ctx, query, args...)
}

func (d debugQueryer) QueryContext(ctx context.Context, query string, args ...any) (scan.Rows, error) {
	d.print(query, args...)
	return d.w.QueryContext(ctx, query, args...)
}
