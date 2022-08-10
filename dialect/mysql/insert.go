package mysql

import (
	"fmt"
	"io"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/clause"
	"github.com/stephenafamo/bob/mods"
)

func Insert(queryMods ...bob.Mod[*InsertQuery]) bob.BaseQuery[*InsertQuery] {
	q := &InsertQuery{}
	for _, mod := range queryMods {
		mod.Apply(q)
	}

	return bob.BaseQuery[*InsertQuery]{
		Expression: q,
		Dialect:    dialect,
	}
}

// Trying to represent the query structure as documented in
// https://dev.mysql.com/doc/refman/8.0/en/insert.html
type InsertQuery struct {
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

func (i InsertQuery) WriteSQL(w io.Writer, d bob.Dialect, start int) ([]any, error) {
	var args []any
	var err error

	w.Write([]byte("INSERT "))

	// no optimizer hint args
	_, err = bob.ExpressIf(w, d, start+len(args), i.hints,
		len(i.hints.hints) > 0, "\n", "\n")
	if err != nil {
		return nil, err
	}

	// no modifiers args
	_, err = bob.ExpressIf(w, d, start+len(args), i.modifiers,
		len(i.modifiers.modifiers) > 0, "", " ")
	if err != nil {
		return nil, err
	}

	// no expected table args
	_, err = bob.ExpressIf(w, d, start+len(args), i.table, true, "INTO ", " ")
	if err != nil {
		return nil, err
	}

	// no partition args
	_, err = bob.ExpressIf(w, d, start+len(args), i.partitions,
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
	valArgs, err := bob.ExpressSlice(w, d, start+len(args), i.sets, "\nSET ", "\n", " ")
	if err != nil {
		return nil, err
	}
	args = append(args, valArgs...)

	// Either this or SET will get expressed
	setArgs, err := bob.ExpressIf(w, d, start+len(args), i.Values, len(i.sets) == 0, "\n", " ")
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
	updateArgs, err := bob.ExpressSlice(w, d, start+len(args), i.duplicateKeyUpdate,
		"\nON DUPLICATE KEY UPDATE\n", "\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, updateArgs...)

	w.Write([]byte("\n"))
	return args, nil
}

func (i *InsertQuery) addSet(s set) {
	i.sets = append(i.sets, s)
}

//nolint:gochecknoglobals
var InsertQM = insertQM{}

type insertQM struct {
	setMod[*InsertQuery]
	hintMod[*InsertQuery]      // for optimizer hints
	partitionMod[*InsertQuery] // for partitions
}

func (qm insertQM) Into(name any, columns ...string) bob.Mod[*InsertQuery] {
	return mods.QueryModFunc[*InsertQuery](func(i *InsertQuery) {
		i.table = name
		i.columns = columns
	})
}

func (qm insertQM) LowPriority() bob.Mod[*InsertQuery] {
	return mods.QueryModFunc[*InsertQuery](func(i *InsertQuery) {
		i.AppendModifier("LOW_PRIORITY")
	})
}

func (qm insertQM) HighPriority() bob.Mod[*InsertQuery] {
	return mods.QueryModFunc[*InsertQuery](func(i *InsertQuery) {
		i.AppendModifier("HIGH_PRIORITY")
	})
}

func (qm insertQM) Ignore() bob.Mod[*InsertQuery] {
	return mods.QueryModFunc[*InsertQuery](func(i *InsertQuery) {
		i.AppendModifier("IGNORE")
	})
}

func (qm insertQM) Values(clauses ...any) bob.Mod[*InsertQuery] {
	return mods.Values[*InsertQuery](clauses)
}

// Insert from a query
// If Go allows type parameters on methods, limit this to select, table and raw
func (qm insertQM) Query(q bob.Query) bob.Mod[*InsertQuery] {
	return mods.QueryModFunc[*InsertQuery](func(i *InsertQuery) {
		i.Query = q
	})
}

// Insert with Set a = b
func (qm insertQM) Set(col string, val any) setMod[*InsertQuery] {
	return setMod[*InsertQuery]{
		col: col,
		val: val,
	}
}

func (qm insertQM) As(rowAlias string, colAlias ...string) bob.Mod[*InsertQuery] {
	return mods.QueryModFunc[*InsertQuery](func(i *InsertQuery) {
		i.rowAlias = rowAlias
		i.columnAlias = colAlias
	})
}

func (qm insertQM) OnDuplicateKeyUpdate(sets ...setMod[*InsertQuery]) bob.Mod[*InsertQuery] {
	return mods.QueryModFunc[*InsertQuery](func(i *InsertQuery) {
		for _, s := range sets {
			i.duplicateKeyUpdate = append(i.duplicateKeyUpdate, set(s))
		}
	})
}
