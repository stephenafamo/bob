package clause

import (
	"context"
	"errors"
	"io"

	"github.com/stephenafamo/bob"
)

// ErrWhereCurrentOfConflict indicates both WHERE and WHERE CURRENT OF were set.
var ErrWhereCurrentOfConflict = errors.New("cannot use both WHERE and WHERE CURRENT OF")

type Where struct {
	Conditions []any
}

func (wh *Where) AppendWhere(e ...any) {
	wh.Conditions = append(wh.Conditions, e...)
}

func (wh Where) WriteSQL(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
	args, err := bob.ExpressSlice(ctx, w, d, start, wh.Conditions, "WHERE ", " AND ", "")
	if err != nil {
		return nil, err
	}

	return args, nil
}

// WhereCurrentOf represents WHERE CURRENT OF clause.
type WhereCurrentOf struct {
	Cursor string
}

func (w *WhereCurrentOf) SetCurrentOf(cursor string) {
	w.Cursor = cursor
}

func (w WhereCurrentOf) WriteSQL(ctx context.Context, wr io.StringWriter, d bob.Dialect, start int) ([]any, error) {
	if w.Cursor == "" {
		return nil, nil
	}

	wr.WriteString("WHERE CURRENT OF ")
	d.WriteQuoted(wr, w.Cursor)

	return nil, nil
}

// WriteWhereAndCurrentOf writes WHERE and WHERE CURRENT OF clauses.
func WriteWhereAndCurrentOf(ctx context.Context, w io.StringWriter, d bob.Dialect, start int, where Where, currentOf WhereCurrentOf) ([]any, error) {
	if len(where.Conditions) > 0 && currentOf.Cursor != "" {
		return nil, ErrWhereCurrentOfConflict
	}

	whereArgs, err := bob.ExpressIf(ctx, w, d, start, where, len(where.Conditions) > 0, "\n", "")
	if err != nil {
		return nil, err
	}

	currentOfArgs, err := bob.ExpressIf(ctx, w, d, start+len(whereArgs), currentOf,
		currentOf.Cursor != "", "\n", "")
	if err != nil {
		return nil, err
	}

	return append(whereArgs, currentOfArgs...), nil
}
