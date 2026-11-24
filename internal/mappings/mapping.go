package mappings

import (
	"reflect"
	"strings"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/expr"
	"github.com/stephenafamo/scan"
)

// SQLColumns uses the declared SQL names even when Scan sees several fields
// inside one column. Without names, the default mapper needs Scan's leaf
// fields; a custom mapper can read each top-level field as a single column.
func SQLColumns[T any](columns bob.Expression, parent string, customScanner bool) expr.ColumnsExpr {
	if c, ok := columns.(interface{ Names() []string }); ok {
		return expr.NewColumnsExpr(c.Names()...).WithParent(parent)
	}
	if !customScanner {
		return expr.ColsForStruct[T](parent)
	}
	var names []string
	modelColumns[T](true, nil, func(name, _ string) {
		names = append(names, name)
	})
	return expr.NewColumnsExpr(names...).WithParent(parent)
}

// A custom scanner can need a specific SQL expression. Without one, use
// the physical columns qualified by the table's alias.
func Projection(columns bob.Expression, physical expr.ColumnsExpr, customScanner bool) bob.Expression {
	if customScanner && columns != nil {
		return columns
	}
	return physical
}

type Mapping struct {
	NonGenerated  []string
	AutoIncrement []string
}

// WriteColumns follows model declaration order, independent of the SELECT
// projection. Setters append their VALUES in this order.
func WriteColumns[T any](customScanner bool, physical []string) Mapping {
	var c Mapping
	var mapped bool
	modelColumns[T](customScanner, physical, func(name, opts string) {
		mapped = true
		var generated bool
		for opt := range strings.SplitSeq(opts, ",") {
			switch opt {
			case "generated":
				generated = true
			case "autoincr":
				c.AutoIncrement = append(c.AutoIncrement, name)
			}
		}
		if !generated {
			c.NonGenerated = append(c.NonGenerated, name)
		}
	})
	if !mapped && customScanner {
		// A scalar model has no field order. Its declared physical
		// columns are the only available positional mapping.
		c.NonGenerated = physical
	}
	return c
}

// The default scanner reads leaves; custom scanners can read a whole
// top-level field as one physical column. Embedded fields stay flattened.
func modelColumns[T any](customScanner bool, physical []string, visit func(name, opts string)) {
	typ := reflect.TypeFor[T]()
	fields, _ := scan.StructMapperFields(typ)
	if typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}

	// A custom scanner may explicitly name nested columns. Expand those
	// parents, even when only one of their children is in the projection.
	expanded := make(map[string]bool)
	if customScanner {
		for _, col := range physical {
			for i := strings.LastIndexByte(col, '.'); i >= 0; i = strings.LastIndexByte(col, '.') {
				col = col[:i]
				expanded[col] = true
			}
		}
	}
	seen := make(map[string]bool)
	for _, field := range fields {
		name := field.Name
		var column, options string
		for path := field.Index; len(path) > 0 && name != ""; path = path[:len(path)-1] {
			f := typ.FieldByIndex(path)
			if f.Anonymous {
				continue // Scan flattens embedded fields.
			}
			tag, opts, _ := strings.Cut(f.Tag.Get("db"), ",")
			var parent string
			if tag != "" && tag != name {
				parent, _ = strings.CutSuffix(name, "."+tag)
			} else if tag == "" {
				if i := strings.LastIndexByte(name, '.'); i >= 0 {
					parent = name[:i]
				}
			}
			if column == "" || !expanded[name] {
				column, options = name, opts
			}
			if !customScanner {
				break
			}
			name = parent
		}
		if column != "" && !seen[column] {
			visit(column, options)
			seen[column] = true
		}
	}
}
