package dialect

import (
	"io"
)

//nolint:gochecknoglobals
var (
	Dialect      dialect
	questionMark = []byte("?")
	backtick     = []byte("`")
)

type dialect struct{}

func (d dialect) WriteArg(w io.Writer, position int) {
	w.Write(questionMark)
}

func (d dialect) WriteQuoted(w io.Writer, s string) {
	w.Write(backtick)
	w.Write([]byte(s))
	w.Write(backtick)
}
