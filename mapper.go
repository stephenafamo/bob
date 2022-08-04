package bob

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
)

type (
	cols    = map[string]int
	visited map[reflect.Type]int
)

func (v visited) copy() visited {
	v2 := make(visited, len(v))
	for t, c := range v {
		v2[t] = c
	}

	return v2
}

type mapping = map[string]mapinfo

type mapinfo struct {
	position []int
	init     [][]int
}

// MapperGen is a function that return the mapping function.
// Any expensive operation, like reflection should be done outside the returned
// function.
// It is called first with the columns from the query to get the mapping function
// which is then used to map every row
// The generator function does not return an error itself to make it less cumbersome
//
// It is recommended to instead return a mapping function that returns an error
// the ErrorMapper is provider for this
type MapperGen[T any] func(cols map[string]int) func(*Values) (T, error)

// The generator function does not return an error itself to make it less cumbersome
// so we return a function that only returns an error instead
// This function makes it easy to return this error
func ErrorMapper[T any](err error, meta ...string) func(*Values) (T, error) {
	err = Error(err, meta...)

	return func(*Values) (T, error) {
		var t T
		return t, err
	}
}

func Error(err error, meta ...string) *mappingError {
	return &mappingError{cause: err, meta: meta}
}

type mappingError struct {
	meta  []string // easy compare
	cause error
}

func (m *mappingError) Unwrap() error {
	return m.cause
}

func (m *mappingError) Error() string {
	return m.cause.Error()
}

func (m *mappingError) Equal(err error) bool {
	var m2 *mappingError
	if !errors.As(err, &m2) {
		return errors.Is(m, err) || errors.Is(err, m)
	}

	if len(m.meta) != len(m2.meta) {
		return false
	}

	// if no meta, the error strings should match exactly
	if len(m.meta) == 0 {
		return m.Error() == m2.Error()
	}

	for k := range m.meta {
		if m.meta[k] != m2.meta[k] {
			return false
		}
	}

	return true
}

// To map to a single value. For queries that return only one column
func SingleValueMapper[T any](c cols) func(*Values) (T, error) {
	if len(c) != 1 {
		err := fmt.Errorf("Expected 1 column but got %d columns", len(c))
		return ErrorMapper[T](err, "wrong column count", "1", strconv.Itoa(len(c)))
	}

	// Get the column name
	var colName string
	for name := range c {
		colName = name
	}

	return func(v *Values) (T, error) {
		return Get[T](v, colName), nil
	}
}

// Maps each row into []any in the order
func SliceMapper[T any](c cols) func(*Values) ([]T, error) {
	return func(v *Values) ([]T, error) {
		row := make([]T, len(c))

		for name, index := range c {
			row[index] = Get[T](v, name)
		}

		return row, nil
	}
}

// Maps all rows into map[string]T
// Most likely used with interface{} to get a map[string]interface{}
func MapMapper[T any](c cols) func(*Values) (map[string]T, error) {
	return func(v *Values) (map[string]T, error) {
		row := make(map[string]T, len(c))

		for name := range c {
			row[name] = Get[T](v, name)
		}

		return row, nil
	}
}
