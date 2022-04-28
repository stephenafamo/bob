package expr

import (
	"io"

	"github.com/stephenafamo/bob/query"
)

type FromItems struct {
	Items []FromItem
}

func (f *FromItems) AppendFromItem(item FromItem) {
	f.Items = append(f.Items, item)
}

func (f FromItems) WriteSQL(w io.Writer, d query.Dialect, start int) ([]any, error) {
	return query.ExpressSlice(w, d, start, f.Items, "", ",\n", "")
}

/*
https://www.postgresql.org/docs/current/sql-select.html#SQL-WITH

where from_item can be one of:

    [ ONLY ] table_name [ * ] [ [ AS ] alias [ ( column_alias [, ...] ) ] ]
                [ TABLESAMPLE sampling_method ( argument [, ...] ) [ REPEATABLE ( seed ) ] ]
    [ LATERAL ] ( select ) [ AS ] alias [ ( column_alias [, ...] ) ]
    with_query_name [ [ AS ] alias [ ( column_alias [, ...] ) ] ]
    [ LATERAL ] function_name ( [ argument [, ...] ] )
                [ WITH ORDINALITY ] [ [ AS ] alias [ ( column_alias [, ...] ) ] ]
    [ LATERAL ] function_name ( [ argument [, ...] ] ) [ AS ] alias ( column_definition [, ...] )
    [ LATERAL ] function_name ( [ argument [, ...] ] ) AS ( column_definition [, ...] )
    [ LATERAL ] ROWS FROM( function_name ( [ argument [, ...] ] ) [ AS ( column_definition [, ...] ) ] [, ...] )
                [ WITH ORDINALITY ] [ [ AS ] alias [ ( column_alias [, ...] ) ] ]
    from_item [ NATURAL ] join_type from_item [ ON join_condition | USING ( join_column [, ...] ) [ AS join_using_alias ] ]
*/

type FromItem struct {
	Table any
	Funcs []any

	// Aliases
	Alias   string
	Columns []string

	// Modifiers for the query
	Only           bool
	Lateral        bool
	WithOrdinality bool

	// Joins
	Joins []Join
}

func (f *FromItem) SetTableAlias(alias string, columns ...string) {
	f.Alias = alias
	f.Columns = columns
}

func (f *FromItem) AppendFunction(function Function) {
	f.Funcs = append(f.Funcs, function)
}

func (f *FromItem) AppendJoin(j Join) {
	f.Joins = append(f.Joins, j)
}

func (f FromItem) WriteSQL(w io.Writer, d query.Dialect, start int) ([]any, error) {
	if f.Table == nil && f.Funcs == nil {
		return nil, nil
	}

	if f.Table == nil {
		f.Table = Functions(f.Funcs)
	}

	if f.Only {
		w.Write([]byte("ONLY "))
	}

	if f.Lateral {
		w.Write([]byte("LATERAL "))
	}

	args, err := query.Express(w, d, start, f.Table)
	if err != nil {
		return nil, err
	}

	if f.WithOrdinality {
		w.Write([]byte("WITH ORDINALITY "))
	}

	if f.Alias != "" {
		w.Write([]byte(" AS "))
		d.WriteQuoted(w, f.Alias)
	}

	if len(f.Columns) > 0 {
		w.Write([]byte("("))
		for k, cAlias := range f.Columns {
			if k != 0 {
				w.Write([]byte(", "))
			}

			d.WriteQuoted(w, cAlias)
		}
		w.Write([]byte(")"))
	}

	joinArgs, err := query.ExpressSlice(w, d, start+len(args), f.Joins, "\n", "\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, joinArgs...)

	return args, nil
}
