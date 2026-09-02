package dialect

import (
	"context"
	"io"

	"github.com/stephenafamo/bob"
)

// Trying to represent the notify query structure as documented in
// https://www.postgresql.org/docs/current/sql-notify.html
type NotifyQuery struct {
	Channel string
	Payload string
}

func (n NotifyQuery) WriteSQL(_ context.Context, w io.StringWriter, dl bob.Dialect, _ int) ([]any, error) {
	w.WriteString("NOTIFY ")
	dl.WriteQuoted(w, n.Channel)
	if n.Payload != "" {
		w.WriteString(", '")
		w.WriteString(n.Payload)
		w.WriteString("'")
	}
	return nil, nil
}
