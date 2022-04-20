package expr

import (
	"io"

	"github.com/stephenafamo/typesql/query"
)

type Conflict struct {
	Do string // DO NOTHING | DO UPDATE
	ConflictTarget
	ConflictUpdateSet
}

func (c *Conflict) SetConflict(conflict Conflict) {
	*c = conflict
}

func (c Conflict) WriteSQL(w io.Writer, d query.Dialect, start int) ([]any, error) {
	w.Write([]byte("ON CONFLICT"))

	args, err := query.ExpressIf(w, d, start, c.Target, true, " ", "")
	if err != nil {
		return nil, err
	}

	w.Write([]byte(" DO "))
	w.Write([]byte(c.Do))

	setArgs, err := query.ExpressIf(w, d, start+len(args), c.ConflictUpdateSet, true, " ", "")
	if err != nil {
		return nil, err
	}
	args = append(args, setArgs...)

	return args, nil
}

type ConflictTarget struct {
	Target any
	Where  []any
}

func (c ConflictTarget) WriteSQL(w io.Writer, d query.Dialect, start int) ([]any, error) {
	args, err := query.ExpressIf(w, d, start, c.Target, c.Target != nil, "", "")
	if err != nil {
		return nil, err
	}

	whereArgs, err := query.ExpressSlice(w, d, start+len(args), c.Where, " WHERE ", " AND ", "")
	if err != nil {
		return nil, err
	}
	args = append(args, whereArgs...)

	return args, nil
}

type ConflictUpdateSet struct {
	Set   []any
	Where []any
}

func (c ConflictUpdateSet) WriteSQL(w io.Writer, d query.Dialect, start int) ([]any, error) {
	args, err := query.ExpressSlice(w, d, start, c.Set, "SET\n", ",\n", "")
	if err != nil {
		return nil, err
	}

	whereArgs, err := query.ExpressSlice(w, d, start+len(args), c.Where, "\nWHERE ", " AND ", "")
	if err != nil {
		return nil, err
	}
	args = append(args, whereArgs...)

	return args, nil
}
