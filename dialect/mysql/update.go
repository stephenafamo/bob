package mysql

import (
	"io"

	"github.com/stephenafamo/bob"
	clause "github.com/stephenafamo/bob/clause"
	"github.com/stephenafamo/bob/dialect/mysql/dialect"
)

func Update(queryMods ...bob.Mod[*UpdateQuery]) bob.BaseQuery[*UpdateQuery] {
	q := &UpdateQuery{}
	for _, mod := range queryMods {
		mod.Apply(q)
	}

	return bob.BaseQuery[*UpdateQuery]{
		Expression: q,
		Dialect:    dialect.Dialect,
	}
}

// Trying to represent the select query structure as documented in
// https://www.postgresql.org/docs/current/sql-update.html
type UpdateQuery struct {
	hints
	modifiers[any]

	clause.With
	clause.Set
	clause.From
	clause.Where
	clause.OrderBy
	clause.Limit
}

func (u UpdateQuery) WriteSQL(w io.Writer, d bob.Dialect, start int) ([]any, error) {
	var args []any

	withArgs, err := bob.ExpressIf(w, d, start+len(args), u.With,
		len(u.With.CTEs) > 0, "\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, withArgs...)

	w.Write([]byte("UPDATE"))

	// no optimizer hint args
	_, err = bob.ExpressIf(w, d, start+len(args), u.hints,
		len(u.hints.hints) > 0, "\n", "\n")
	if err != nil {
		return nil, err
	}

	// no modifiers args
	_, err = bob.ExpressIf(w, d, start+len(args), u.modifiers,
		len(u.modifiers.modifiers) > 0, " ", "")
	if err != nil {
		return nil, err
	}

	fromArgs, err := bob.ExpressIf(w, d, start+len(args), u.From,
		u.From.Table != nil, " ", "")
	if err != nil {
		return nil, err
	}
	args = append(args, fromArgs...)

	setArgs, err := bob.ExpressIf(w, d, start+len(args), u.Set, true, " ", "")
	if err != nil {
		return nil, err
	}
	args = append(args, setArgs...)

	whereArgs, err := bob.ExpressIf(w, d, start+len(args), u.Where,
		len(u.Where.Conditions) > 0, "\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, whereArgs...)

	orderArgs, err := bob.ExpressIf(w, d, start+len(args), u.OrderBy,
		len(u.OrderBy.Expressions) > 0, "\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, orderArgs...)

	_, err = bob.ExpressIf(w, d, start+len(args), u.Limit,
		u.Limit.Count != nil, "\n", "")
	if err != nil {
		return nil, err
	}

	return args, nil
}
