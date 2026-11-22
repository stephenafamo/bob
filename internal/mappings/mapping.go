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
	seen := make(map[string]bool)
	visitFields[T](func(name, _ string, topLevel bool) {
		if topLevel && !seen[name] {
			names = append(names, name)
			seen[name] = true
		}
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

// WriteColumns applies model tag options to the declared SQL columns, in
// insert order. A Scan field can be a descendant of a single physical column
// when the caller supplies a custom scanner.
func WriteColumns[T any](names []string) Mapping {
	options := make(map[string]string)
	visitFields[T](func(name, opts string, _ bool) {
		if opts != "" {
			options[name] = opts
		}
	})

	var c Mapping
	for _, name := range names {
		var generated bool
		for opt := range strings.SplitSeq(options[name], ",") {
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
	}
	return c
}

// Visit each Scan field and its named ancestors. Start from Scan's resolved
// name so untagged fields and flattened embeds follow the mapper's rules.
func visitFields[T any](visit func(name, opts string, topLevel bool)) {
	typ := reflect.TypeFor[T]()
	fields, _ := scan.StructMapperFields(typ)
	if typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}

	for _, field := range fields {
		name := field.Name
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
			visit(name, opts, parent == "")
			name = parent
		}
	}
}
