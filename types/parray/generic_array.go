package parray

import (
	"database/sql/driver"

	"github.com/lib/pq"
)

type GenericArray[T any] []T

// Scan implements the sql.Scanner interface.
func (e *GenericArray[T]) Scan(src any) error {
	return pq.GenericArray{A: e}.Scan(src)
}

// Value implements the driver.Valuer interface.
func (e GenericArray[T]) Value() (driver.Value, error) {
	return pq.GenericArray{A: e}.Value()
}
