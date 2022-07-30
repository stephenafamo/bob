package mysql

import (
	"io"

	"github.com/stephenafamo/bob/clause"
	"github.com/stephenafamo/bob/query"
)

func Table(queryMods ...query.Mod[*tableQuery]) query.BaseQuery[*tableQuery] {
	q := &tableQuery{}
	for _, mod := range queryMods {
		mod.Apply(q)
	}

	return query.BaseQuery[*tableQuery]{
		Expression: q,
		Dialect:    dialect,
	}
}

type tableQuery struct {
	name string
	clause.OrderBy
	clause.Limit
	clause.Offset
}

func (t tableQuery) WriteSQL(w io.Writer, d query.Dialect, start int) ([]any, error) {
	if t.name == "" {
		return nil, nil
	}

	var args []any

	orderArgs, err := query.ExpressIf(w, d, start+len(args), t.OrderBy,
		len(t.OrderBy.Expressions) > 0, " ", "")
	if err != nil {
		return nil, err
	}
	args = append(args, orderArgs...)

	_, err = query.ExpressIf(w, d, start+len(args), t.Limit,
		t.Limit.Count != nil, " ", "")
	if err != nil {
		return nil, err
	}

	_, err = query.ExpressIf(w, d, start+len(args), t.Offset,
		t.Offset.Count != nil, " ", "")
	if err != nil {
		return nil, err
	}

	w.Write([]byte("\n"))
	return args, nil
}

//TODO: Add Table statement mods
