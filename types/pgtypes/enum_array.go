package pgtypes

import (
	"database/sql"
	"database/sql/driver"

	"github.com/jackc/pgx/v5/pgtype"
)

var (
	_ sql.Scanner        = (*EnumArray[string])(nil)
	_ driver.Valuer      = EnumArray[string]{}
	_ pgtype.ArraySetter = (*EnumArray[string])(nil)
	_ pgtype.ArrayGetter = EnumArray[string]{}
)

// EnumArray is an [Array] of a Postgres enum type.
type EnumArray[T ~string] = Array[T]
