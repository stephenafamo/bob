package pgtypes

import (
	"database/sql"
	"database/sql/driver"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/lib/pq"
)

var (
	_ sql.Scanner        = (*EnumArray[string])(nil)
	_ driver.Valuer      = EnumArray[string]{}
	_ pgtype.ArraySetter = (*EnumArray[string])(nil)
	_ pgtype.ArrayGetter = EnumArray[string]{}
)

// EnumArray is a one-dimensional Postgres array of an enum type.
// A nil EnumArray is a SQL NULL, an empty non-nil EnumArray is an empty array.
//
// Like [Array], it implements sql.Scanner, driver.Valuer,
// pgtype.ArraySetter and pgtype.ArrayGetter.
type EnumArray[T ~string] []T

// Dimensions implements the pgtype.ArrayGetter interface.
func (e EnumArray[T]) Dimensions() []pgtype.ArrayDimension {
	if e == nil {
		return nil
	}

	return []pgtype.ArrayDimension{{Length: int32(len(e)), LowerBound: 1}}
}

// Index implements the pgtype.ArrayGetter interface.
func (e EnumArray[T]) Index(i int) any {
	return e[i]
}

// IndexType implements the pgtype.ArrayGetter interface.
func (e EnumArray[T]) IndexType() any {
	var el T
	return el
}

// SetDimensions implements the pgtype.ArraySetter interface.
// Multi-dimensional arrays are flattened.
func (e *EnumArray[T]) SetDimensions(dimensions []pgtype.ArrayDimension) error {
	if dimensions == nil {
		*e = nil
		return nil
	}

	*e = make(EnumArray[T], cardinality(dimensions))
	return nil
}

// ScanIndex implements the pgtype.ArraySetter interface.
func (e EnumArray[T]) ScanIndex(i int) any {
	return &e[i]
}

// ScanIndexType implements the pgtype.ArraySetter interface.
func (e EnumArray[T]) ScanIndexType() any {
	return new(T)
}

// Scan implements the sql.Scanner interface.
func (e *EnumArray[T]) Scan(src any) error {
	return scanArray(src, e, new(T))
}

// Value implements the driver.Valuer interface.
func (e EnumArray[T]) Value() (driver.Value, error) {
	if e == nil {
		return nil, nil //nolint:nilnil
	}

	arr := make(pq.StringArray, len(e))
	for i, s := range e {
		arr[i] = string(s)
	}

	return arr.Value()
}
