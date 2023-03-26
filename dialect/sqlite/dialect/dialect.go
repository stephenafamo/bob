package dialect

import (
	"io"
	"strconv"
)

//nolint:gochecknoglobals
var (
	Dialect      dialect
	questionMark = []byte("?")
	colon        = []byte(":")
	doubleQuote  = []byte(`"`)
)

type dialect struct{}

func (d dialect) WriteArg(w io.Writer, position int) {
	w.Write(questionMark)
	w.Write([]byte(strconv.Itoa(position)))
}

func (d dialect) WriteNamedArg(w io.Writer, name string) {
	w.Write(colon)
	w.Write([]byte(name))
}

func (d dialect) WriteQuoted(w io.Writer, s string) {
	w.Write(doubleQuote)
	w.Write([]byte(s))
	w.Write(doubleQuote)
}
