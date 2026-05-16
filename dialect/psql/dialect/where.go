package dialect

import (
	"context"
	"errors"
	"io"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/clause"
)

var errWhereCurrentOfConflict = errors.New("cannot use both WHERE and WHERE CURRENT OF")

func writeWhereAndCurrentOf(ctx context.Context, w io.StringWriter, d bob.Dialect, start int, where clause.Where, currentOf string) ([]any, error) {
	if len(where.Conditions) > 0 && currentOf != "" {
		return nil, errWhereCurrentOfConflict
	}

	whereArgs, err := bob.ExpressIf(ctx, w, d, start, where, len(where.Conditions) > 0, "\n", "")
	if err != nil {
		return nil, err
	}

	if currentOf != "" {
		w.WriteString("\nWHERE CURRENT OF ")
		w.WriteString(currentOf)
	}

	return whereArgs, nil
}
