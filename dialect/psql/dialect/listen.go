package dialect

import (
	"context"
	"io"

	"github.com/stephenafamo/bob"
)

// Trying to represent the listen query structure as documented in
// https://www.postgresql.org/docs/current/sql-listen.html
type ListenQuery struct {
	Channel string
}

func (l ListenQuery) WriteSQL(_ context.Context, w io.StringWriter, dl bob.Dialect, _ int) ([]any, error) {
	w.WriteString("LISTEN ")
	dl.WriteQuoted(w, l.Channel)
	return nil, nil
}
