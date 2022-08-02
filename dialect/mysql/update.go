package mysql

import (
	"io"

	"github.com/stephenafamo/bob"
	clause "github.com/stephenafamo/bob/clause"
	"github.com/stephenafamo/bob/expr"
	"github.com/stephenafamo/bob/mods"
)

func Update(queryMods ...bob.Mod[*updateQuery]) bob.BaseQuery[*updateQuery] {
	q := &updateQuery{}
	for _, mod := range queryMods {
		mod.Apply(q)
	}

	return bob.BaseQuery[*updateQuery]{
		Expression: q,
		Dialect:    dialect,
	}
}

// Trying to represent the select query structure as documented in
// https://www.postgresql.org/docs/current/sql-update.html
type updateQuery struct {
	hints
	modifiers[any]

	clause.With
	clause.Set
	clause.FromItems
	clause.Where
	clause.OrderBy
	clause.Limit
}

func (u updateQuery) WriteSQL(w io.Writer, d bob.Dialect, start int) ([]any, error) {
	var args []any

	withArgs, err := bob.ExpressIf(w, d, start+len(args), u.With,
		len(u.With.CTEs) > 0, "\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, withArgs...)

	w.Write([]byte("UPDATE "))

	// no optimizer hint args
	_, err = bob.ExpressIf(w, d, start+len(args), u.hints,
		len(u.hints.hints) > 0, "\n", "\n")
	if err != nil {
		return nil, err
	}

	// no modifiers args
	_, err = bob.ExpressIf(w, d, start+len(args), u.modifiers,
		len(u.modifiers.modifiers) > 0, "", " ")
	if err != nil {
		return nil, err
	}

	tableArgs, err := bob.ExpressSlice(w, d, start+len(args), u.FromItems.Items, "\n", ",\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, tableArgs...)

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

type UpdateQM struct {
	hintMod[*updateQuery] // for optimizer hints
	withMod[*updateQuery]

	fromMod mods.FromMod[*updateQuery] // we don't use it as FROM
	fromItemMod
	joinMod[*clause.FromItem]
}

func (UpdateQM) LowPriority() bob.Mod[*deleteQuery] {
	return mods.QueryModFunc[*deleteQuery](func(i *deleteQuery) {
		i.AppendModifier("LOW_PRIORITY")
	})
}

func (UpdateQM) Ignore() bob.Mod[*deleteQuery] {
	return mods.QueryModFunc[*deleteQuery](func(i *deleteQuery) {
		i.AppendModifier("IGNORE")
	})
}

func (u UpdateQM) Table(name any, mods ...bob.Mod[*clause.FromItem]) bob.Mod[*updateQuery] {
	return u.fromMod.From(name, mods...)
}

func (u UpdateQM) TableAs(name any, alias string, mods ...bob.Mod[*clause.FromItem]) bob.Mod[*updateQuery] {
	return u.fromMod.From(name, append(mods, u.As(alias))...)
}

func (UpdateQM) Set(a string, b any) bob.Mod[*updateQuery] {
	return mods.Set[*updateQuery]{expr.OP("=", Quote(a), b)}
}

func (UpdateQM) SetArg(a string, b any) bob.Mod[*updateQuery] {
	return mods.Set[*updateQuery]{expr.OP("=", Quote(a), Arg(b))}
}

func (UpdateQM) Where(e bob.Expression) bob.Mod[*updateQuery] {
	return mods.Where[*updateQuery]{e}
}

func (UpdateQM) WhereClause(clause string, args ...any) bob.Mod[*updateQuery] {
	return mods.Where[*updateQuery]{Raw(clause, args...)}
}

func (UpdateQM) OrderBy(e any) orderBy[*updateQuery] {
	return orderBy[*updateQuery](func() clause.OrderDef {
		return clause.OrderDef{
			Expression: e,
		}
	})
}

func (UpdateQM) Limit(count int64) bob.Mod[*updateQuery] {
	return mods.Limit[*updateQuery]{
		Count: count,
	}
}
