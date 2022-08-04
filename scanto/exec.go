package scanto

import (
	"context"
)

func One[T any](ctx context.Context, exec Queryer, m MapperGen[T], sql string, args ...any) (T, error) {
	var t T

	rows, err := exec.QueryContext(ctx, sql, args...)
	if err != nil {
		return t, err
	}
	defer rows.Close()

	v, err := newValues(rows)
	if err != nil {
		return t, err
	}

	genFunc := m(v.columnsCopy())

	// Record the mapping
	v.recording = true
	if _, err = genFunc(v); err != nil {
		return t, err
	}
	v.recording = false

	rows.Next()
	if err = v.scanRow(rows); err != nil {
		return t, err
	}

	t, err = genFunc(v)
	if err != nil {
		return t, err
	}

	return t, rows.Err()
}

func All[T any](ctx context.Context, exec Queryer, m MapperGen[T], sql string, args ...any) ([]T, error) {
	var results []T

	rows, err := exec.QueryContext(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	v, err := newValues(rows)
	if err != nil {
		return nil, err
	}

	genFunc := m(v.columnsCopy())

	// Record the mapping
	v.recording = true
	if _, err = genFunc(v); err != nil {
		return nil, err
	}
	v.recording = false

	for rows.Next() {
		err = v.scanRow(rows)
		if err != nil {
			return nil, err
		}

		one, err := genFunc(v)
		if err != nil {
			return nil, err
		}

		results = append(results, one)
	}

	return results, rows.Err()
}
