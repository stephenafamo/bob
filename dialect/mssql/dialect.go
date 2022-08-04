package mssql

import (
	"io"
	"strconv"
)

//nolint:gochecknoglobals
var dialect Dialect

type Dialect struct{}

func (d Dialect) WriteArg(w io.Writer, position int) {
	w.Write([]byte("@p"))
	w.Write([]byte(strconv.Itoa(position)))
}

func (d Dialect) WriteNamedArg(w io.Writer, name string) {
	w.Write([]byte("@"))
	w.Write([]byte(name))
}

func (d Dialect) WriteQuoted(w io.Writer, s string) {
	w.Write([]byte("["))
	w.Write([]byte(s))
	w.Write([]byte("]"))
}
