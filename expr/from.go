package expr

import (
	"io"

	"github.com/stephenafamo/typesql/query"
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
	Joins []Join
}

func (f *FromItem) SetTable(table Table) {
	f.Table = table
}

func (f *FromItem) AppendJoin(j Join) {
	f.Joins = append(f.Joins, j)
}

func (f FromItem) WriteSQL(w io.Writer, d query.Dialect, start int) ([]any, error) {
	if f.Table == nil {
		return nil, nil
	}

	args, err := query.Express(w, d, start, f.Table)
	if err != nil {
		return nil, err
	}

	joinArgs, err := query.ExpressSlice(w, d, start+len(args), f.Joins, "\n", "\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, joinArgs...)

	return args, nil
}
