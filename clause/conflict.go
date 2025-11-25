package clause

import (
	"context"
	"io"

	"github.com/stephenafamo/bob"
)

type Conflict struct {
	Expression bob.Expression
}

func (c *Conflict) SetConflict(conflict bob.Expression) {
	c.Expression = conflict
}

type ConflictClause struct {
	Do     string // DO NOTHING | DO UPDATE
	Target ConflictTarget
	Set
	Where
}

func (c ConflictClause) WriteSQL(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
	w.WriteString("ON CONFLICT")

	args, err := bob.ExpressIf(ctx, w, d, start, c.Target, true, "", "")
	if err != nil {
		return nil, err
	}

	w.WriteString(" DO ")
	w.WriteString(c.Do)

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

func (c ConflictTarget) WriteSQL(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
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
