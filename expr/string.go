package expr

import (
	"context"
	"io"

	"github.com/stephenafamo/bob"
)

type rawString string

// ShouldOmitParens reports that rawString is expected to be printed as it is.
func (rawString) ShouldOmitParens() bool { return true }

func (s rawString) WriteSQL(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
	w.WriteString("'")
	w.WriteString(string(s))
	w.WriteString("'")

	return nil, nil
}
