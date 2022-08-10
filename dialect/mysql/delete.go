package mysql

import (
	"io"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/clause"
	"github.com/stephenafamo/bob/mods"
)

func Delete(queryMods ...bob.Mod[*DeleteQuery]) bob.BaseQuery[*DeleteQuery] {
	q := &DeleteQuery{}
	for _, mod := range queryMods {
		mod.Apply(q)
	}

	return bob.BaseQuery[*DeleteQuery]{
		Expression: q,
		Dialect:    dialect,
	}
}

// Trying to represent the query structure as documented in
// https://dev.mysql.com/doc/refman/8.0/en/delete.html
type DeleteQuery struct {
	hints

	clause.With
	modifiers[string]
	partitions
	tables []clause.Table
	clause.From
	clause.Where
	clause.OrderBy
	clause.Limit
}

func (d DeleteQuery) WriteSQL(w io.Writer, dl bob.Dialect, start int) ([]any, error) {
	var args []any

	withArgs, err := bob.ExpressIf(w, dl, start+len(args), d.With,
		len(d.With.CTEs) > 0, "\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, withArgs...)

	w.Write([]byte("DELETE "))

	// no optimizer hint args
	_, err = bob.ExpressIf(w, dl, start+len(args), d.hints,
		len(d.hints.hints) > 0, "\n", "\n")
	if err != nil {
		return nil, err
	}

	// no modifiers args
	_, err = bob.ExpressIf(w, dl, start+len(args), d.modifiers,
		len(d.modifiers.modifiers) > 0, "", " ")
	if err != nil {
		return nil, err
	}

	tableArgs, err := bob.ExpressSlice(w, dl, start+len(args), d.tables, "FROM ", ", ", "")
	if err != nil {
		return nil, err
	}
	args = append(args, tableArgs...)

	usingArgs, err := bob.ExpressIf(w, dl, start+len(args), d.From,
		d.From.Table != nil, "\nUSING ", "")
	if err != nil {
		return nil, err
	}
	args = append(args, usingArgs...)

	whereArgs, err := bob.ExpressIf(w, dl, start+len(args), d.Where,
		len(d.Where.Conditions) > 0, "\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, whereArgs...)

	orderArgs, err := bob.ExpressIf(w, dl, start+len(args), d.OrderBy,
		len(d.OrderBy.Expressions) > 0, "\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, orderArgs...)

	_, err = bob.ExpressIf(w, dl, start+len(args), d.Limit,
		d.Limit.Count != nil, "\n", "")
	if err != nil {
		return nil, err
	}

	return args, nil
}

//nolint:gochecknoglobals
var DeleteQM = deleteQM{}

type deleteQM struct {
	hintMod[*DeleteQuery] // for optimizer hints
	withMod[*DeleteQuery]
	fromItemMod[*DeleteQuery]
	joinMod[*clause.From]
}

func (deleteQM) LowPriority() bob.Mod[*DeleteQuery] {
	return mods.QueryModFunc[*DeleteQuery](func(i *DeleteQuery) {
		i.AppendModifier("LOW_PRIORITY")
	})
}

func (deleteQM) Quick() bob.Mod[*DeleteQuery] {
	return mods.QueryModFunc[*DeleteQuery](func(i *DeleteQuery) {
		i.AppendModifier("QUICK")
	})
}

func (deleteQM) Ignore() bob.Mod[*DeleteQuery] {
	return mods.QueryModFunc[*DeleteQuery](func(i *DeleteQuery) {
		i.AppendModifier("IGNORE")
	})
}

func (qm deleteQM) From(name any) bob.Mod[*DeleteQuery] {
	return mods.QueryModFunc[*DeleteQuery](func(u *DeleteQuery) {
		u.tables = append(u.tables, clause.Table{
			Expression: name,
		})
	})
}

func (qm deleteQM) FromAs(name any, alias string) bob.Mod[*DeleteQuery] {
	return mods.QueryModFunc[*DeleteQuery](func(u *DeleteQuery) {
		u.tables = append(u.tables, clause.Table{
			Expression: name,
			Alias:      alias,
		})
	})
}

func (qm deleteQM) Using(table any) bob.Mod[*DeleteQuery] {
	return mods.QueryModFunc[*DeleteQuery](func(q *DeleteQuery) {
		q.SetTable(table)
	})
}

func (qm deleteQM) Where(e bob.Expression) bob.Mod[*DeleteQuery] {
	return mods.Where[*DeleteQuery]{e}
}

func (qm deleteQM) WhereClause(clause string, args ...any) bob.Mod[*DeleteQuery] {
	return mods.Where[*DeleteQuery]{Raw(clause, args...)}
}

func (deleteQM) OrderBy(e any) orderBy[*DeleteQuery] {
	return orderBy[*DeleteQuery](func() clause.OrderDef {
		return clause.OrderDef{
			Expression: e,
		}
	})
}

func (deleteQM) Limit(count int64) bob.Mod[*DeleteQuery] {
	return mods.Limit[*DeleteQuery]{
		Count: count,
	}
}
