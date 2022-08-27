package dialect

import (
	"io"
)

//nolint:gochecknoglobals
var Dialect dialect

type dialect struct{}

func (d dialect) WriteArg(w io.Writer, position int) {
	w.Write([]byte("?"))
}

func (d dialect) WriteQuoted(w io.Writer, s string) {
	w.Write([]byte("`"))
	w.Write([]byte(s))
	w.Write([]byte("`"))
}
