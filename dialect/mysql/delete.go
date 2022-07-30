package mysql

import (
	"io"

	"github.com/stephenafamo/bob/clause"
	"github.com/stephenafamo/bob/mods"
	"github.com/stephenafamo/bob/query"
)

func Delete(queryMods ...query.Mod[*deleteQuery]) query.BaseQuery[*deleteQuery] {
	q := &deleteQuery{}
	for _, mod := range queryMods {
		mod.Apply(q)
	}

	return query.BaseQuery[*deleteQuery]{
		Expression: q,
		Dialect:    dialect,
	}
}

// Trying to represent the query structure as documented in
// https://dev.mysql.com/doc/refman/8.0/en/delete.html
type deleteQuery struct {
	clause.With
	modifiers[string]
	partitions
	tables []clause.Table
	clause.FromItems
	clause.Where
	clause.OrderBy
	clause.Limit
}

func (d deleteQuery) WriteSQL(w io.Writer, dl query.Dialect, start int) ([]any, error) {
	var args []any

	withArgs, err := query.ExpressIf(w, dl, start+len(args), d.With,
		len(d.With.CTEs) > 0, "\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, withArgs...)

	w.Write([]byte("DELETE "))

	// no modifiers args
	_, err = query.ExpressIf(w, dl, start+len(args), d.modifiers,
		len(d.modifiers.modifiers) > 0, "", " ")
	if err != nil {
		return nil, err
	}

	tableArgs, err := query.ExpressSlice(w, dl, start+len(args), d.tables, "FROM ", ", ", "")
	if err != nil {
		return nil, err
	}
	args = append(args, tableArgs...)

	usingArgs, err := query.ExpressSlice(w, dl, start+len(args), d.FromItems.Items, "\nUSING ", ",\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, usingArgs...)

	whereArgs, err := query.ExpressIf(w, dl, start+len(args), d.Where,
		len(d.Where.Conditions) > 0, "\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, whereArgs...)

	orderArgs, err := query.ExpressIf(w, dl, start+len(args), d.OrderBy,
		len(d.OrderBy.Expressions) > 0, "\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, orderArgs...)

	_, err = query.ExpressIf(w, dl, start+len(args), d.Limit,
		d.Limit.Count != nil, "\n", "")
	if err != nil {
		return nil, err
	}

	return args, nil
}

type DeleteQM struct {
	withMod[*deleteQuery]
	mods.FromMod[*deleteQuery]
	fromItemMod
	joinMod[*clause.FromItem]
}

func (DeleteQM) LowPriority() query.Mod[*deleteQuery] {
	return mods.QueryModFunc[*deleteQuery](func(i *deleteQuery) {
		i.AppendModifier("LOW_PRIORITY")
	})
}

func (DeleteQM) Quick() query.Mod[*deleteQuery] {
	return mods.QueryModFunc[*deleteQuery](func(i *deleteQuery) {
		i.AppendModifier("QUICK")
	})
}

func (DeleteQM) Ignore() query.Mod[*deleteQuery] {
	return mods.QueryModFunc[*deleteQuery](func(i *deleteQuery) {
		i.AppendModifier("IGNORE")
	})
}

func (qm DeleteQM) From(name any) query.Mod[*deleteQuery] {
	return mods.QueryModFunc[*deleteQuery](func(u *deleteQuery) {
		u.tables = append(u.tables, clause.Table{
			Expression: name,
		})
	})
}

func (qm DeleteQM) FromAs(name any, alias string) query.Mod[*deleteQuery] {
	return mods.QueryModFunc[*deleteQuery](func(u *deleteQuery) {
		u.tables = append(u.tables, clause.Table{
			Expression: name,
			Alias:      alias,
		})
	})
}

func (qm DeleteQM) Using(table any, usingMods ...query.Mod[*clause.FromItem]) query.Mod[*deleteQuery] {
	return qm.FromMod.From(table, usingMods...)
}

func (qm DeleteQM) Where(e query.Expression) query.Mod[*deleteQuery] {
	return mods.Where[*deleteQuery]{e}
}

func (qm DeleteQM) WhereClause(clause string, args ...any) query.Mod[*deleteQuery] {
	return mods.Where[*deleteQuery]{Raw(clause, args...)}
}

func (DeleteQM) OrderBy(e any) orderBy[*deleteQuery] {
	return orderBy[*deleteQuery](func() clause.OrderDef {
		return clause.OrderDef{
			Expression: e,
		}
	})
}

func (DeleteQM) Limit(count int64) query.Mod[*deleteQuery] {
	return mods.Limit[*deleteQuery]{
		Count: count,
	}
}
