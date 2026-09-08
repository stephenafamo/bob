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
	if b, ok := src.([]byte); ok && isBinaryArray(b) {
		return scanArray(src, a, new(T))
	}

	// lib/pq has faster parsers for the primitive element types
	switch v := any(a).(type) {
	case *Array[string]:
		return scanPQ[pq.StringArray](src, v)
	case *Array[bool]:
		return scanPQ[pq.BoolArray](src, v)
	case *Array[int32]:
		return scanPQ[pq.Int32Array](src, v)
	case *Array[int64]:
		return scanPQ[pq.Int64Array](src, v)
	case *Array[float32]:
		return scanPQ[pq.Float32Array](src, v)
	case *Array[float64]:
		return scanPQ[pq.Float64Array](src, v)
	case *Array[[]byte]:
		return scanPQ[pq.ByteaArray](src, v)
	}

	return scanArray(src, a, new(T))
}

// scanPQ scans a text array literal into dst using one of lib/pq's array types.
// lib/pq does not understand explicit bounds or multiple dimensions,
// so anything it rejects is handed to the pgx parser instead.
func scanPQ[PQ ~[]E, E any, P interface {
	*PQ
	sql.Scanner
}](src any, dst *Array[E],
) error {
	var v PQ
	if err := P(&v).Scan(src); err != nil {
		return scanArray(src, dst, new(E))
	}

	*dst = Array[E](v)
	return nil
}

// Value implements the driver.Valuer interface.
func (a Array[T]) Value() (driver.Value, error) {
	// lib/pq has faster encoders for the primitive element types,
	// and bytea elements must be hex encoded, which pq.GenericArray does not do
	switch v := any(a).(type) {
	case Array[string]:
		return pq.StringArray(v).Value()
	case Array[bool]:
		return pq.BoolArray(v).Value()
	case Array[int32]:
		return pq.Int32Array(v).Value()
	case Array[int64]:
		return pq.Int64Array(v).Value()
	case Array[float32]:
		return pq.Float32Array(v).Value()
	case Array[float64]:
		return pq.Float64Array(v).Value()
	case Array[[]byte]:
		return pq.ByteaArray(v).Value()
	}

	return pq.GenericArray{A: []T(a)}.Value()
}
