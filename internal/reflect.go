package internal

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/stephenafamo/bob/expr"
	"github.com/stephenafamo/bob/orm"
)

var (
	matchFirstCapRe = regexp.MustCompile("(.)([A-Z][a-z]+)")
	matchAllCapRe   = regexp.MustCompile("([a-z0-9])([A-Z])")
)

// snakeCaseFieldFunc is a NameMapperFunc that maps struct field to snake case.
func snakeCase(str string) string {
	snake := matchFirstCapRe.ReplaceAllString(str, "${1}_${2}")
	snake = matchAllCapRe.ReplaceAllString(snake, "${1}_${2}")
	return strings.ToLower(snake)
}

//nolint:gochecknoglobals
var settableTyp = reflect.TypeOf((*interface{ IsSet() bool })(nil)).Elem()

type Mapping struct {
	All          []string
	PKs          []string
	NonPKs       []string
	Generated    []string
	NonGenerated []string

	CreatedAtCol int
	UpdatedAtCol int
	DeletedAtCol int
}

func (c Mapping) Columns(table ...string) orm.Columns {
	cols := make([]string, 0, len(c.All))
	for _, col := range c.All {
		if col == "" {
			continue
		}

		cols = append(cols, col)
	}

	copy(cols, c.All)

	return orm.NewColumns(cols).WithParent(table...)
}

type colProperties struct {
	Name        string
	IsPK        bool
	IsGenerated bool
}

func getColProperties(tag string) colProperties {
	var p colProperties
	if tag == "" {
		return p
	}

	parts := strings.Split(tag, ",")
	p.Name = parts[0]

	for _, part := range parts[1:] {
		switch part {
		case "pk":
			p.IsPK = true
		case "generated":
			p.IsGenerated = true
		}
	}

	return p
}

func GetMappings(typ reflect.Type) Mapping {
	c := Mapping{}

	if typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}

	if typ.Kind() != reflect.Struct {
		return c
	}

	c.All = make([]string, typ.NumField())
	c.PKs = make([]string, typ.NumField())
	c.NonPKs = make([]string, typ.NumField())
	c.Generated = make([]string, typ.NumField())
	c.NonGenerated = make([]string, typ.NumField())

	// Go through the struct fields and populate the map.
	// Recursively go into any child structs, adding a prefix where necessary
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)

		// Don't consider unexported fields
		if !field.IsExported() {
			continue
		}

		// Skip columns that have the tag "-"
		tag := field.Tag.Get("db")
		if tag == "-" {
			continue
		}

		if tag == "" {
			tag = snakeCase(field.Name)
		}

		props := getColProperties(tag)

		c.All[field.Index[0]] = props.Name
		if props.IsPK {
			c.PKs[field.Index[0]] = props.Name
		} else {
			c.NonPKs[field.Index[0]] = props.Name
		}
		if props.IsGenerated {
			c.Generated[field.Index[0]] = props.Name
		} else {
			c.NonGenerated[field.Index[0]] = props.Name
		}
	}

	return c
}

// Get the values for non generated columns
func GetColumnValues[T any](mapping Mapping, filter []string, objs ...T) ([]string, [][]any, error) {
	if len(objs) == 0 {
		return nil, nil, nil
	}

	allvalues := make([][]any, 0, len(objs))

	refVal1 := reflect.ValueOf(objs[0])
	cols, vals1, err := getObjColsVals(mapping, filter, refVal1)
	if err != nil {
		return nil, nil, fmt.Errorf("get column list: %w", err)
	}

	allvalues = append(allvalues, vals1)

	colMap := map[string]struct{}{}
	for _, c := range cols {
		colMap[c] = struct{}{}
	}

	for index, obj := range objs[1:] {
		refVal := reflect.ValueOf(obj)
		values, err := getObjVals(mapping, colMap, refVal)
		if err != nil {
			return nil, nil, fmt.Errorf("row %d: %w", index+2, err)
		}

		allvalues = append(allvalues, values)
	}

	return cols, allvalues, nil
}

func getObjColsVals(mapping Mapping, filter []string, val reflect.Value) ([]string, []any, error) {
	cols := make([]string, 0, len(mapping.NonGenerated))
	values := make([]any, 0, len(mapping.NonGenerated))

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
		if field.Type().Implements(settableTyp) {
			shoudSet = field.MethodByName("IsSet").Call(nil)[0].Interface().(bool)
		}

		if !shoudSet {
			continue
		}

		cols = append(cols, name)
		values = append(values, expr.Arg(field.Interface()))
	}

	return cols, values, nil
}

func getObjVals(mapping Mapping, colMap map[string]struct{}, val reflect.Value) ([]any, error) {
	if val.Kind() == reflect.Pointer {
		if val.IsNil() {
			return nil, errors.New("object is nil")
		}
		val = val.Elem()
	}

	values := make([]any, 0, len(colMap))

	for index, name := range mapping.NonGenerated {
		if name == "" {
			continue
		}

		if _, ok := colMap[name]; !ok {
			continue
		}

		field := val.Field(index)
		values = append(values, expr.Arg(field.Interface()))
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
