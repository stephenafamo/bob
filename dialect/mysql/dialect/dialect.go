package dialect

import (
	"io"
)

//nolint:gochecknoglobals
var Dialect dialect

type dialect struct{}

func (d dialect) WriteArg(w io.StringWriter, position int) {
	w.WriteString("?")
}

func (d dialect) WriteQuoted(w io.StringWriter, s string) {
	w.WriteString("`")
	w.WriteString(s)
	w.WriteString("`")
}
