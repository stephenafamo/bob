package pgtypes

import (
	"database/sql/driver"

	"github.com/lib/pq"
)

type EnumArray[T ~string] []T

// Scan implements the sql.Scanner interface.
func (e *EnumArray[T]) Scan(src any) error {
	var arr pq.StringArray
	if err := arr.Scan(src); err != nil {
		return err
	}

	slice := make([]T, len(arr))
	for i, s := range arr {
		slice[i] = T(s)
	}

	*e = slice
	return nil
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
