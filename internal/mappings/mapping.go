package mappings

import (
	"reflect"
	"strings"

	"github.com/stephenafamo/scan"
)

type Mapping struct {
	All           []string
	NonGenerated  []string
	AutoIncrement []string
}

// GetMappings adds Bob's write-column flags to Scan's resolved fields.
func GetMappings[T any]() Mapping {
	typ := reflect.TypeFor[T]()
	fields, err := scan.StructMapperFields(typ)
	if err != nil {
		return Mapping{} // A non-struct model has no column metadata.
	}
	if typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}

	c := Mapping{All: make([]string, 0, len(fields)), NonGenerated: make([]string, 0, len(fields))}
	for _, field := range fields {
		if field.Name == "" || len(field.Index) == 0 {
			continue
		}
		c.All = append(c.All, field.Name)
		_, opts, _ := strings.Cut(typ.FieldByIndex(field.Index).Tag.Get("db"), ",")
		var generated bool
		for opt := range strings.SplitSeq(opts, ",") {
			switch opt {
			case "generated":
				generated = true
			case "autoincr":
				c.AutoIncrement = append(c.AutoIncrement, field.Name)
			}
		}
		if !generated {
			c.NonGenerated = append(c.NonGenerated, field.Name)
		}
	}

	return c
}
