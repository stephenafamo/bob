package internal

import (
	"reflect"
	"regexp"
	"strings"

	"github.com/stephenafamo/bob/orm"
)

type Columns struct {
	All       []string
	PKs       []string
	Generated []string
}

func (c Columns) Get(table ...string) orm.Columns {
	cols := make([]string, len(c.All))
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

func GetColumns(typ reflect.Type) Columns {
	var c Columns

	if typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}

	if typ.Kind() != reflect.Struct {
		return c
	}

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

		if field.Anonymous {
			fieldType := field.Type
			if fieldType.Kind() == reflect.Pointer {
				fieldType = fieldType.Elem()
			}

			if fieldType.Kind() == reflect.Struct {
				newCols := GetColumns(fieldType)
				c.All = append(c.All, newCols.All...)
				c.PKs = append(c.PKs, newCols.PKs...)
				c.Generated = append(c.Generated, newCols.Generated...)
				continue
			}
		}

		c.All = append(c.All, props.Name)
		if props.IsPK {
			c.PKs = append(c.PKs, props.Name)
		}
		if props.IsGenerated {
			c.Generated = append(c.Generated, props.Name)
		}
	}

	return c
}

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
