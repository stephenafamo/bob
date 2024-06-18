package pgtypes

import (
	"database/sql/driver"

	"github.com/lib/pq"
)

type Array[T any] []T

// Scan implements the sql.Scanner interface.
func (e *Array[T]) Scan(src any) error {
	return pq.GenericArray{A: e}.Scan(src)
}

// Value implements the driver.Valuer interface.
func (e Array[T]) Value() (driver.Value, error) {
	return pq.GenericArray{A: e}.Value()
}
