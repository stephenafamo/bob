package dialect

import (
	"io"
	"strconv"
)

//nolint:gochecknoglobals
var Dialect dialect

type dialect struct{}

func (d dialect) WriteArg(w io.StringWriter, position int) {
	w.WriteString("$")
	w.WriteString(strconv.Itoa(position))
}

func (d dialect) WriteQuoted(w io.StringWriter, s string) {
	w.WriteString(`"`)
	w.WriteString(s)
	w.WriteString(`"`)
}
