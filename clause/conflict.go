package clause

import (
	"context"
	"io"

	"github.com/twitter-payments/bob"
)

type Conflict struct {
	Do     string // DO NOTHING | DO UPDATE
	Target ConflictTarget
	Set
	Where
}

func (c *Conflict) SetConflict(conflict Conflict) {
	*c = conflict
}

func (c Conflict) WriteSQL(ctx context.Context, w io.Writer, d bob.Dialect, start int) ([]any, error) {
	w.Write([]byte("ON CONFLICT"))

	args, err := bob.ExpressIf(ctx, w, d, start, c.Target, true, "", "")
	if err != nil {
		return nil, err
	}

	w.Write([]byte(" DO "))
	w.Write([]byte(c.Do))

	setArgs, err := bob.ExpressIf(ctx, w, d, start+len(args), c.Set, len(c.Set.Set) > 0, " SET\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, setArgs...)

	whereArgs, err := bob.ExpressIf(ctx, w, d, start+len(args), c.Where,
		len(c.Where.Conditions) > 0, "\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, whereArgs...)

	return args, nil
}

type ConflictTarget struct {
	Constraint string
	Columns    []any
	Where      []any
}

func (c ConflictTarget) WriteSQL(ctx context.Context, w io.Writer, d bob.Dialect, start int) ([]any, error) {
	if c.Constraint != "" {
		return bob.ExpressIf(ctx, w, d, start, c.Constraint, true, " ON CONSTRAINT ", "")
	}

	args, err := bob.ExpressSlice(ctx, w, d, start, c.Columns, " (", ", ", ")")
	if err != nil {
		return nil, err
	}

	whereArgs, err := bob.ExpressSlice(ctx, w, d, start+len(args), c.Where, " WHERE ", " AND ", "")
	if err != nil {
		return nil, err
	}
	args = append(args, whereArgs...)

	return args, nil
}
