package pgtypes

import (
	"database/sql"
	"database/sql/driver"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/lib/pq"
)

var (
	_ sql.Scanner        = (*Array[int])(nil)
	_ driver.Valuer      = Array[int]{}
	_ pgtype.ArraySetter = (*Array[int])(nil)
	_ pgtype.ArrayGetter = Array[int]{}
)

// Array is a one-dimensional Postgres array.
// A nil Array is a SQL NULL, an empty non-nil Array is an empty array.
//
// It implements sql.Scanner and driver.Valuer for use with database/sql drivers
// such as lib/pq and pgx/stdlib, and pgtype.ArraySetter and pgtype.ArrayGetter
// so that it is natively scanned and encoded by pgx.
// Scan accepts both the text array literal and the pgx binary wire format,
// so it also works when wrapped by another sql.Scanner (e.g. null.Val or sql.Null).
type Array[T any] []T

// Dimensions implements the pgtype.ArrayGetter interface.
func (a Array[T]) Dimensions() []pgtype.ArrayDimension {
	if a == nil {
		return nil
	}

	return []pgtype.ArrayDimension{{Length: int32(len(a)), LowerBound: 1}}
}

// Index implements the pgtype.ArrayGetter interface.
func (a Array[T]) Index(i int) any {
	return a[i]
}

// IndexType implements the pgtype.ArrayGetter interface.
func (a Array[T]) IndexType() any {
	var el T
	return el
}

// SetDimensions implements the pgtype.ArraySetter interface.
// Multi-dimensional arrays are flattened.
func (a *Array[T]) SetDimensions(dimensions []pgtype.ArrayDimension) error {
	if dimensions == nil {
		*a = nil
		return nil
	}

	*a = make(Array[T], cardinality(dimensions))
	return nil
}

// ScanIndex implements the pgtype.ArraySetter interface.
func (a Array[T]) ScanIndex(i int) any {
	return &a[i]
}

// ScanIndexType implements the pgtype.ArraySetter interface.
func (a Array[T]) ScanIndexType() any {
	return new(T)
}

// Scan implements the sql.Scanner interface.
func (a *Array[T]) Scan(src any) error {
	return scanArray(src, a, new(T))
}

// Value implements the driver.Valuer interface.
func (a Array[T]) Value() (driver.Value, error) {
	// bytea elements must be hex encoded, which pq.GenericArray does not do
	if b, ok := any(a).(Array[[]byte]); ok {
		return pq.ByteaArray(b).Value()
	}

	return pq.GenericArray{A: []T(a)}.Value()
}
