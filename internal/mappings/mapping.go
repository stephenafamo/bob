package mappings

import (
	"reflect"
	"regexp"
	"strings"
)

var (
	matchFirstCapRe = regexp.MustCompile("(.)([A-Z][a-z]+)")
	matchAllCapRe   = regexp.MustCompile("([a-z0-9])([A-Z])")
)

type colProperties struct {
	Name          string
	IsPK          bool
	IsGenerated   bool
	AutoIncrement bool
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
		case "autoincr":
			p.AutoIncrement = true
		}
	}

	return p
}

type Mapping struct {
	All           []string
	PKs           []string
	NonPKs        []string
	Generated     []string
	NonGenerated  []string
	AutoIncrement []string
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
	c.AutoIncrement = make([]string, typ.NumField())

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
		if props.AutoIncrement {
			c.AutoIncrement[field.Index[0]] = props.Name
		}
	}

	return c
}

// snakeCaseFieldFunc is a NameMapperFunc that maps struct field to snake case.
func snakeCase(str string) string {
	snake := matchFirstCapRe.ReplaceAllString(str, "${1}_${2}")
	snake = matchAllCapRe.ReplaceAllString(snake, "${1}_${2}")
	return strings.ToLower(snake)
}
