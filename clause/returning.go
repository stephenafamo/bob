package clause

import (
	"context"
	"io"

	"github.com/stephenafamo/bob"
)

type Returning struct {
	OldAlias    string
	NewAlias    string
	Expressions []any
}

func (r *Returning) HasReturning() bool {
	return len(r.Expressions) > 0
}

func (r *Returning) SetOldAlias(alias string) {
	r.OldAlias = alias
}

func (r *Returning) SetNewAlias(alias string) {
	r.NewAlias = alias
}

func (r *Returning) AppendReturning(columns ...any) {
	r.Expressions = append(r.Expressions, columns...)
}

func (r Returning) WriteSQL(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
	if len(r.Expressions) == 0 {
		return nil, nil
	}

	w.WriteString("RETURNING ")

	if r.OldAlias != "" || r.NewAlias != "" {
		w.WriteString("WITH (")

		if r.OldAlias != "" {
			w.WriteString("OLD AS ")
			d.WriteQuoted(w, r.OldAlias)
		}

		if r.NewAlias != "" {
			if r.OldAlias != "" {
				w.WriteString(", ")
			}

			w.WriteString("NEW AS ")
			d.WriteQuoted(w, r.NewAlias)
		}

		w.WriteString(") ")
	}

	return bob.ExpressSlice(ctx, w, d, start, r.Expressions, "", ", ", "")
}
