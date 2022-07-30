package mysql

import (
	"fmt"
	"io"

	"github.com/stephenafamo/bob/clause"
	"github.com/stephenafamo/bob/mods"
	"github.com/stephenafamo/bob/query"
)

func Insert(queryMods ...query.Mod[*insertQuery]) query.BaseQuery[*insertQuery] {
	q := &insertQuery{}
	for _, mod := range queryMods {
		mod.Apply(q)
	}

	return query.BaseQuery[*insertQuery]{
		Expression: q,
		Dialect:    dialect,
	}
}

// Trying to represent the query structure as documented in
// https://dev.mysql.com/doc/refman/8.0/en/insert.html
type insertQuery struct {
	hints
	modifiers[string]
	partitions
	clause.Values

	table              any
	columns            []string
	rowAlias           string
	columnAlias        []string
	sets               []set
	duplicateKeyUpdate []set
}

func (i insertQuery) WriteSQL(w io.Writer, d query.Dialect, start int) ([]any, error) {
	var args []any
	var err error

	w.Write([]byte("INSERT "))

	// no optimizer hint args
	_, err = query.ExpressIf(w, d, start+len(args), i.hints,
		len(i.hints.hints) > 0, "\n", "\n")
	if err != nil {
		return nil, err
	}

	// no modifiers args
	_, err = query.ExpressIf(w, d, start+len(args), i.modifiers,
		len(i.modifiers.modifiers) > 0, "", " ")
	if err != nil {
		return nil, err
	}

	// no expected table args
	_, err = query.ExpressIf(w, d, start+len(args), i.table, true, "INTO ", " ")
	if err != nil {
		return nil, err
	}

	// no partition args
	_, err = query.ExpressIf(w, d, start+len(args), i.partitions,
		len(i.partitions.partitions) > 0, "", " ")
	if err != nil {
		return nil, err
	}

	// No columns args
	if len(i.columns) > 0 {
		w.Write([]byte(" ("))
		for k, cAlias := range i.columns {
			if k != 0 {
				w.Write([]byte(", "))
			}

			d.WriteQuoted(w, cAlias)
		}
		w.Write([]byte(")"))
	}

	// Either this or the values will get expressed
	valArgs, err := query.ExpressSlice(w, d, start+len(args), i.sets, "\nSET ", "\n", " ")
	if err != nil {
		return nil, err
	}
	args = append(args, valArgs...)

	// Either this or SET will get expressed
	setArgs, err := query.ExpressIf(w, d, start+len(args), i.Values, len(i.sets) == 0, "\n", " ")
	if err != nil {
		return nil, err
	}
	args = append(args, setArgs...)

	// The aliases
	if i.rowAlias != "" {
		_, err = fmt.Fprintf(w, "\nAS %s", i.rowAlias)
		if err != nil {
			return nil, err
		}

		if len(i.columnAlias) > 0 {
			w.Write([]byte("("))
			for k, cAlias := range i.columnAlias {
				if k != 0 {
					w.Write([]byte(", "))
				}

				d.WriteQuoted(w, cAlias)
			}
			w.Write([]byte(")"))
		}
	}

	// Either this or the values will get expressed
	updateArgs, err := query.ExpressSlice(w, d, start+len(args), i.duplicateKeyUpdate,
		"\nON DUPLICATE KEY UPDATE\n", "\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, updateArgs...)

	w.Write([]byte("\n"))
	return args, nil
}

func (i *insertQuery) addSet(s set) {
	i.sets = append(i.sets, s)
}

type InsertQM struct {
	setMod[*insertQuery]
	hintMod[*insertQuery]      // for optimizer hints
	partitionMod[*insertQuery] // for partitions
}

func (qm InsertQM) Into(name any, columns ...string) query.Mod[*insertQuery] {
	return mods.QueryModFunc[*insertQuery](func(i *insertQuery) {
		i.table = name
		i.columns = columns
	})
}

func (qm InsertQM) LowPriority() query.Mod[*insertQuery] {
	return mods.QueryModFunc[*insertQuery](func(i *insertQuery) {
		i.AppendModifier("LOW_PRIORITY")
	})
}

func (qm InsertQM) HighPriority() query.Mod[*insertQuery] {
	return mods.QueryModFunc[*insertQuery](func(i *insertQuery) {
		i.AppendModifier("HIGH_PRIORITY")
	})
}

func (qm InsertQM) Ignore() query.Mod[*insertQuery] {
	return mods.QueryModFunc[*insertQuery](func(i *insertQuery) {
		i.AppendModifier("IGNORE")
	})
}

func (qm InsertQM) Values(clauses ...any) query.Mod[*insertQuery] {
	return mods.Values[*insertQuery](clauses)
}

// Insert from a query
// If Go allows type parameters on methods, limit this to select, table and raw
func (qm InsertQM) Query(q query.BaseQuery[*selectQuery]) query.Mod[*insertQuery] {
	return mods.QueryModFunc[*insertQuery](func(i *insertQuery) {
		i.Query = q
	})
}

// Insert with Set a = b
func (qm InsertQM) Set(col string, val any) setMod[*insertQuery] {
	return setMod[*insertQuery]{
		col: col,
		val: val,
	}
}

func (qm InsertQM) As(rowAlias string, colAlias ...string) query.Mod[*insertQuery] {
	return mods.QueryModFunc[*insertQuery](func(i *insertQuery) {
		i.rowAlias = rowAlias
		i.columnAlias = colAlias
	})
}

func (qm InsertQM) OnDuplicateKeyUpdate(sets ...setMod[*insertQuery]) query.Mod[*insertQuery] {
	return mods.QueryModFunc[*insertQuery](func(i *insertQuery) {
		for _, s := range sets {
			i.duplicateKeyUpdate = append(i.duplicateKeyUpdate, set(s))
		}
	})
}
