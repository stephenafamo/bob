package mssql

import (
	"io"
	"strconv"
)

//nolint:gochecknoglobals
var (
	Dialect             dialect
	atSign              = []byte("@")
	atSignP             = []byte("@p")
	openSquareBrackets  = []byte("[")
	closeSquareBrackets = []byte("]")
)

type dialect struct{}

func (d dialect) WriteArg(w io.Writer, position int) {
	w.Write(atSignP)
	w.Write([]byte(strconv.Itoa(position)))
}

func (d dialect) WriteNamedArg(w io.Writer, name string) {
	w.Write(atSign)
	w.Write([]byte(name))
}

func (d dialect) WriteQuoted(w io.Writer, s string) {
	w.Write(openSquareBrackets)
	w.Write([]byte(s))
	w.Write(closeSquareBrackets)
}
