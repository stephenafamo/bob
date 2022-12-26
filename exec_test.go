package bob

import (
	"context"
	"database/sql"

	"github.com/stephenafamo/scan"
)

type NoopExecutor struct{}

func (n NoopExecutor) QueryContext(ctx context.Context, query string, args ...any) (scan.Rows, error) {
	return nil, nil
}

func (n NoopExecutor) ExecContext(_ context.Context, _ string, _ ...any) (sql.Result, error) {
	return nil, nil
}
