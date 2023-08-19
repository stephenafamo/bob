package internal

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/expr"
	"github.com/stephenafamo/bob/internal/mappings"
	"github.com/stephenafamo/bob/orm"
)

//nolint:gochecknoglobals
var unsettableTyp = reflect.TypeOf((*interface{ IsUnset() bool })(nil)).Elem()

func MappingCols(m mappings.Mapping, table ...string) orm.Columns {
	// to make sure we don't modify the passed slice
	cols := make([]string, 0, len(m.All))
	for _, col := range m.All {
		if col == "" {
			continue
		}

		cols = append(cols, col)
	}

	copy(cols, m.All)

	return orm.NewColumns(cols...).WithParent(table...)
}

// Get the values for non generated columns
func GetColumnValues[T any](mapping mappings.Mapping, filter []string, objs ...T) ([]string, [][]bob.Expression, error) {
	if len(objs) == 0 {
		return nil, nil, nil
	}

	allvalues := make([][]bob.Expression, 0, len(objs))

	refVal1 := reflect.ValueOf(objs[0])
	cols, vals1, err := getObjColsVals(mapping, filter, refVal1)
	if err != nil {
		return nil, nil, fmt.Errorf("get column list: %w", err)
	}

	allvalues = append(allvalues, vals1)

	for index, obj := range objs[1:] {
		refVal := reflect.ValueOf(obj)
		values, err := getObjVals(mapping, cols, refVal)
		if err != nil {
			return nil, nil, fmt.Errorf("row %d: %w", index+2, err)
		}

		allvalues = append(allvalues, values)
	}

	return cols, allvalues, nil
}

func getObjColsVals(mapping mappings.Mapping, filter []string, val reflect.Value) ([]string, []bob.Expression, error) {
	cols := make([]string, 0, len(mapping.NonGenerated))
	values := make([]bob.Expression, 0, len(mapping.NonGenerated))

	if val.Kind() == reflect.Pointer {
		if val.IsNil() {
			return nil, nil, errors.New("object is nil")
		}
		val = val.Elem()
	}

	hasFilter := len(filter) > 0
	filterMap := sliceToMap(filter)
	for colIndex, name := range mapping.NonGenerated {
		if name == "" {
			continue
		}

		if _, ok := filterMap[name]; !ok && hasFilter {
			continue
		}

		field := val.Field(colIndex)

		shoudSet := true
		if field.Type().Implements(unsettableTyp) {
			shoudSet = !field.MethodByName("IsUnset").Call(nil)[0].Interface().(bool)
		}

		if !shoudSet {
			continue
		}

		cols = append(cols, name)
		values = append(values, expr.Arg(field.Interface()))
	}

	return cols, values, nil
}

func getObjVals(mapping mappings.Mapping, cols []string, val reflect.Value) ([]bob.Expression, error) {
	if val.Kind() == reflect.Pointer {
		if val.IsNil() {
			return nil, errors.New("object is nil")
		}
		val = val.Elem()
	}

	values := make([]bob.Expression, 0, len(cols))

	for index, name := range mapping.NonGenerated {
		if name == "" {
			continue
		}

		for _, c := range cols {
			if name == c {
				field := val.Field(index)
				values = append(values, expr.Arg(field.Interface()))
			}
		}
	}

	return values, nil
}

func sliceToMap[T comparable](s []T) map[T]int {
	m := make(map[T]int, len(s))
	for k, v := range s {
		m[v] = k
	}
	return m
}
