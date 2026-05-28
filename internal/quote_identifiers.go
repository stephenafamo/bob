package internal

import (
	"context"
	"io"

	"github.com/stephenafamo/bob"
)

// quotedIdent is a single SQL identifier segment for bob.ExpressSlice.
type quotedIdent string

func (q quotedIdent) WriteSQL(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
	if q == "" {
		return nil, nil
	}

	d.WriteQuoted(w, string(q))

	return nil, nil
}

// QuoteIdentifiers maps each name to a quoted identifier expression for bob.ExpressSlice.
func QuoteIdentifiers(identifiers []string) []bob.Expression {
	if len(identifiers) == 0 {
		return nil
	}

	out := make([]bob.Expression, len(identifiers))
	for i, ident := range identifiers {
		out[i] = quotedIdent(ident)
	}

	return out
}
