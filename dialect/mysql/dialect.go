package mysql

import (
	"io"
)

//nolint:gochecknoglobals
var dialect Dialect

type Dialect struct{}

func (d Dialect) WriteArg(w io.Writer, position int) {
	w.Write([]byte("?"))
}

func (d Dialect) WriteQuoted(w io.Writer, s string) {
	w.Write([]byte("`"))
	w.Write([]byte(s))
	w.Write([]byte("`"))
}
