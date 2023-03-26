package dialect

import (
	"io"
	"strconv"
)

//nolint:gochecknoglobals
var (
	Dialect     dialect
	dollar      = []byte("$")
	doubleQuote = []byte(`"`)
)

type dialect struct{}

func (d dialect) WriteArg(w io.Writer, position int) {
	w.Write(dollar)
	w.Write([]byte(strconv.Itoa(position)))
}

func (d dialect) WriteQuoted(w io.Writer, s string) {
	w.Write(doubleQuote)
	w.Write([]byte(s))
	w.Write(doubleQuote)
}
