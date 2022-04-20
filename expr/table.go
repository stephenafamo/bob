package expr

import (
	"io"

	"github.com/stephenafamo/typesql/query"
)

type Table struct {
	Expression any
	Alias      string
	Columns    []string
}

func (t Table) As(alias string, columns ...string) Table {
	t.Alias = alias
	t.Columns = append(t.Columns, columns...)

	return t
}

func (t Table) WriteSQL(w io.Writer, d query.Dialect, start int) ([]any, error) {
	args, err := query.Express(w, d, start, t.Expression)
	if err != nil {
		return nil, err
	}

	if t.Alias != "" {
		w.Write([]byte(" AS "))
		d.WriteQuoted(w, t.Alias)
	}

	if len(t.Columns) > 0 {
		w.Write([]byte("("))
		for k, cAlias := range t.Columns {
			if k != 0 {
				w.Write([]byte(", "))
			}

			d.WriteQuoted(w, cAlias)
		}
		w.Write([]byte(")"))
	}

	return args, nil
}

// Table returns a table definition
// the expression can be a table name, subquery, function, e.t.c.
func T(expression any) Table {
	t := Table{
		Expression: expression,
	}

	return t
}

// TQuery returns a table definition based on a subquery
// the first value in the aliases is used as the table alias
// a 2nd value onward is used as the column aliases
func TQuery(q query.Query, lateral bool) Table {
	t := T(tableSubQuery{
		query:   q,
		lateral: lateral,
	})

	return t
}

type tableSubQuery struct {
	// has to be a select query or values
	query   query.Query
	lateral bool
}

func (t tableSubQuery) WriteSQL(w io.Writer, d query.Dialect, start int) ([]any, error) {
	if t.lateral {
		w.Write([]byte("LATERAL "))
	}

	w.Write([]byte("("))
	args, err := t.query.WriteQuery(w, start)
	if err != nil {
		return nil, err
	}
	w.Write([]byte(")"))

	return args, nil
}

// TFunc returns a table definition based on a function call
// If more than one function is given, the ROWS FROM syntax is used
func TFunc(f ...function) Table {
	t := T(tableFunction{
		functions: f,
	})

	return t
}

type tableFunction struct {
	functions      []function
	withOrdinality bool
	lateral        bool
}

func (t tableFunction) Lateral() tableFunction {
	t.lateral = true
	return t
}

func (t tableFunction) WithOrdinality() tableFunction {
	t.withOrdinality = true
	return t
}

func (t tableFunction) WriteSQL(w io.Writer, d query.Dialect, start int) ([]any, error) {
	if t.lateral {
		w.Write([]byte("LATERAL "))
	}

	if len(t.functions) > 1 {
		w.Write([]byte("ROWS FROM ("))
	}

	args, err := query.ExpressSlice(w, d, start, t.functions, "", ", ", "")
	if err != nil {
		return nil, err
	}

	if len(t.functions) > 1 {
		w.Write([]byte(")"))
	}

	if t.withOrdinality {
		w.Write([]byte("WITH ORDINALITY "))
	}

	return args, nil
}
