package drivers

import (
	"fmt"
	"strings"

	"github.com/stephenafamo/bob/gen/language"
)

func (q Query) HasNestedReturns() bool {
	for _, col := range q.Columns {
		if strings.Contains(col.DBName, ".") || strings.Contains(col.DBName, "__") {
			return true
		}
	}
	return false
}

// NestedColumns returns a list of nested columns in the query.
// Nesting is determined by the presence of a dot in the column's DBName.
// For example, if a column's DBName is "user.address.street", it will be nested under "user" and "address".
func (q Query) NestedColumns() nestedSlice {
	final := make(nestedSlice, 0, len(q.Columns))

	for colIndex, col := range q.Columns {
		names := strings.Split(col.Name, ".")            // Split by . to handle names like "user.address.street" which indicates to-many nesting
		names = strings.Split(names[len(names)-1], "__") // Split by __ to handle names like "user__address__street" which indicates to-one nesting
		col.Name = names[len(names)-1]                   // Use the last part of the name as the column name

		toManyParts := strings.Split(col.DBName, ".")

		current := &final
		for manyPartIndex, manyPart := range toManyParts {
			toOneParts := strings.Split(manyPart, "__")
			for onePartIndex, part := range toOneParts {
				// If this is the last part, we need to assign the column index and column itself
				isLastPart := (onePartIndex == len(toOneParts)-1) && (manyPartIndex == len(toManyParts)-1)
				if isLastPart {
					*current = append(*current, nested{Index: colIndex, Col: col})
					continue
				}

				// Only merge matching containers. A scalar result can have
				// the same name as a nested result at this level.
				isSingle := onePartIndex < len(toOneParts)-1
				var foundNode *nested
				for i := range *current {
					if node := &(*current)[i]; len(node.Children) > 0 && node.Col.Name == part && node.Single == isSingle {
						foundNode = node
						break
					}
				}

				if foundNode != nil {
					// Node was found, so we just move deeper into its children for the next iteration
					current = &foundNode.Children
					continue
				}

				// Node was not found, create a new one
				*current = append(*current, nested{Single: isSingle, Col: QueryCol{Name: part, DBName: part}})
				// Move deeper into the children of the newly created node
				current = &(*current)[len(*current)-1].Children
			}
		}
	}

	final.FixNames()

	return final
}

type nested struct {
	Index    int
	Single   bool // Indicates if this is a single value (to-one) or a slice (to-many)
	Col      QueryCol
	Children nestedSlice
}

func (n nested) Type(currPkg string, i language.Importer, types Types, typeName string) string {
	if len(n.Children) == 0 {
		return n.Col.Type(currPkg, i, types)
	}

	return fmt.Sprintf("%s_%s", typeName, n.Col.Name)
}

func (n nested) Assign(selfName, rowName string, cols []QueryCol) string {
	if len(n.Children) == 0 {
		return fmt.Sprintf("%s.%s = %s.%s",
			selfName, n.Col.Name,
			rowName, cols[n.Index].Name)
	}
	return ""
}

func (n nested) Compare(currPkg string, i language.Importer, types Types, selfName, rowName string, cols []QueryCol) string {
	if len(n.Children) > 0 {
		return ""
	}

	lhs := fmt.Sprintf("%s.%s", selfName, n.Col.Name)
	rhs := fmt.Sprintf("%s.%s", rowName, cols[n.Index].Name)

	nullable := *n.Col.Nullable
	cmpExpr := types.GetCompareExpr(currPkg, i, n.Col.TypeName, nullable, nullable)
	cmpExpr = strings.ReplaceAll(cmpExpr, "AAA", lhs)
	cmpExpr = strings.ReplaceAll(cmpExpr, "BBB", rhs)

	return cmpExpr
}

func (n nested) NotNull(currPkg string, types Types, rowName string, cols []QueryCol) string {
	if len(n.Children) > 0 {
		return ""
	}
	varName := fmt.Sprintf("%s.%s", rowName, cols[n.Index].Name)
	return types.GetNullTypeValid(currPkg, n.Col.TypeName, varName)
}

type nestedSlice []nested

func (n nestedSlice) FixNames() nestedSlice {
	cols := n.columns().WithNames()
	for i := range n {
		n[i].Col.Name = cols[i].Name
		n[i].Children.FixNames()
	}

	return n
}

func (n nestedSlice) columns() QueryCols {
	cols := make(QueryCols, len(n))
	for i := range n {
		cols[i] = n[i].Col
	}
	return cols
}

func (n nestedSlice) Nullable() bool {
	for _, child := range n {
		if len(child.Children) > 0 {
			continue
		}
		if child.Col.Nullable != nil && !*child.Col.Nullable {
			return false
		}
	}
	return true
}

func (n nestedSlice) Types(currPkg string, i language.Importer, types Types, typeName string) []string {
	return n.TypesWithTags(currPkg, i, types, typeName, QueryResultTags{})
}

func (n nestedSlice) TypesWithTags(currPkg string, i language.Importer, types Types, typeName string, tags QueryResultTags) []string {
	allTypes := []string{""}

	var self strings.Builder

	fmt.Fprintf(&self, "type %s = struct{\n", typeName)
	typeName = strings.TrimSuffix(typeName, "_")

	fieldTags := tags.FieldTags(n.columns())
	for idx, child := range n {
		childType := child.Type(currPkg, i, types, typeName)

		if len(child.Children) > 0 {
			allTypes = append(allTypes, child.Children.TypesWithTags(
				currPkg, i, types, childType, tags,
			)...)

			switch {
			case child.Single && child.Children.AllNullable():
				childType = "*" + childType
			case !child.Single:
				childType = "[]" + childType
			}
		}

		fmt.Fprintf(&self, "%s %s", child.Col.Name, childType)
		if fieldTags[idx] != "" {
			fmt.Fprintf(&self, " `%s`", fieldTags[idx])
		}
		self.WriteByte('\n')
	}

	self.WriteString("}")

	allTypes[0] = self.String()
	return allTypes
}

func (n nestedSlice) AllNullable() bool {
	for _, child := range n {
		if len(child.Children) > 0 {
			continue
		}
		if child.Col.Nullable != nil && !*child.Col.Nullable {
			return false
		}
	}

	return true
}

func (n nestedSlice) FistNotNull() nested {
	for _, child := range n {
		if len(child.Children) > 0 {
			continue
		}
		if child.Col.Nullable != nil && !*child.Col.Nullable {
			return child
		}
	}

	return nested{}
}

func (n nestedSlice) Assign(currPkg string, i language.Importer, types Types, selfName, rowName string, cols []QueryCol) string {
	assigns := make([]string, 0, len(n))
	for _, child := range n {
		assigns = append(assigns, child.Assign(selfName, rowName, cols))
	}

	return strings.Join(assigns, "\n")
}

func (n nestedSlice) Compare(currPkg string, i language.Importer, types Types, selfName, rowName string, cols []QueryCol) string {
	assigns := make([]string, 0, len(n))
	for _, child := range n {
		assign := child.Compare(currPkg, i, types, selfName, rowName, cols)
		if assign == "" {
			continue
		}

		assigns = append(assigns, assign)
	}

	return strings.Join(assigns, " &&\n")
}

func (n nestedSlice) NotNull(currPkg string, types Types, rowName string, cols []QueryCol) string {
	assigns := make([]string, 0, len(n))
	for _, child := range n {
		assign := child.NotNull(currPkg, types, rowName, cols)
		if assign == "" {
			continue
		}

		assigns = append(assigns, assign)
	}

	return strings.Join(assigns, " ||\n")
}

// TransformRows groups the scanned rows in first-seen order. Only the root
// collection shares an index; child collections still belong to each parent.
func (n nestedSlice) TransformRows(currPkg string, i language.Importer, types Types, cols []QueryCol, typeName string) string {
	decl, lookup := n.parentIndex(currPkg, i, types, cols)
	return decl + "\nfor _, row := range scanned {\n" +
		n.transform(currPkg, i, types, cols, false, typeName, "final", "index", lookup) + "\n}\n"
}

// parentIndex emits a key equivalent to Compare: all direct scalar fields,
// and no nested fields. A NULL in any key component never matches another row.
func (n nestedSlice) parentIndex(currPkg string, i language.Importer, types Types, cols []QueryCol) (string, string) {
	var scalars []nested
	for _, child := range n {
		if len(child.Children) != 0 {
			continue
		}
		col := child.Col
		if !types.CanCompareWithEquals(currPkg, col.TypeName) {
			return "", ""
		}
		typ := types.GetWithoutImporting(currPkg, col.TypeName)
		if typ == "any" || typ == "interface{}" {
			return "", "" // the dynamic value need not be hashable
		}
		scalars = append(scalars, child)
	}
	if len(scalars) == 0 {
		return "", ""
	}

	var fields, values, valid []string
	for _, child := range scalars {
		col := child.Col
		field := "row." + cols[child.Index].Name
		fields = append(fields, col.Name+" "+types.Get(currPkg, i, col.TypeName))
		values = append(values, types.UnwrapNullExpr(currPkg, i, col.TypeName, field, *col.Nullable))
		if *col.Nullable {
			valid = append(valid, types.GetNullTypeValid(currPkg, col.TypeName, field))
		}
	}
	decl := "type parentKey struct {\n" + strings.Join(fields, "\n") + "\n}\nparents := make(map[parentKey]int)\n"
	lookup := `key := parentKey{` + strings.Join(values, ", ") + `}
		if pos, ok := parents[key]; ok {
			index = pos
		} else {
			parents[key] = len(final)
		}
	`
	if len(valid) != 0 {
		lookup = "if " + strings.Join(valid, " && ") + " {\n" + lookup + "}\n"
	}
	return decl, lookup
}

func (n nestedSlice) Transform(
	currPkg string,
	i language.Importer,
	types Types,
	cols []QueryCol,
	isSingle bool,
	typeName, collectedRowsVar, indexName string,
) string {
	return n.transform(currPkg, i, types, cols, isSingle, typeName, collectedRowsVar, indexName, "")
}

func (n nestedSlice) transform(
	currPkg string, i language.Importer, types Types, cols []QueryCol,
	isSingle bool, typeName, collectedRowsVar, indexName, lookup string,
) string {
	nullable := n.Nullable()
	transformation := &strings.Builder{}

	if nullable {
		fmt.Fprintf(transformation, "\nif %s {", n.NotNull(currPkg, types, "row", cols))
	}

	switch {
	case isSingle && n.AllNullable():
		fmt.Fprintf(
			transformation, `if %s == nil {
			  var fresh %s
			  %s
			  %s = &fresh
			}
			`,
			collectedRowsVar,
			typeName,
			n.Assign(currPkg, i, types, "fresh", "row", cols),
			collectedRowsVar,
		)

	case isSingle: // isSingle == true && !n.AllNullable()
		col := n.FistNotNull().Col
		nnTypName, _ := types.GetNameAndDef(currPkg, col.TypeName)
		lhs := fmt.Sprintf("%s.%s", collectedRowsVar, col.Name)
		rhs := fmt.Sprintf("zero%s%s", col.Name, indexName)
		cmpExpr := types.GetCompareExpr(currPkg, i, col.TypeName, *col.Nullable, *col.Nullable)
		cmpExpr = strings.ReplaceAll(cmpExpr, "AAA", lhs)
		cmpExpr = strings.ReplaceAll(cmpExpr, "BBB", rhs)

		fmt.Fprintf(
			transformation, `
			var %s %s
			if %s {
			  fresh := %s{}
			  %s
			  %s = fresh
			}
			`,
			rhs, nnTypName, cmpExpr, typeName,
			n.Assign(currPkg, i, types, "fresh", "row", cols),
			collectedRowsVar,
		)

	default: // isSingle == false
		fmt.Fprintf(transformation, "\n%s := -1\n", indexName)
		if lookup != "" {
			transformation.WriteString(lookup)
		} else {
			fmt.Fprintf(transformation, `for i, existing := range %s {
          if %s {
            %s = i
            break
          }
        }
		`, collectedRowsVar, n.Compare(currPkg, i, types, "existing", "row", cols), indexName)
		}

		fmt.Fprintf(transformation, `if %s == -1 {
          fresh := %s{}
          %s
          %s = append(%s, fresh)
          %s = len(%s) - 1
        }
		`,
			indexName, typeName,
			n.Assign(currPkg, i, types, "fresh", "row", cols),
			collectedRowsVar, collectedRowsVar, indexName, collectedRowsVar)
	}

	typeName = strings.TrimSuffix(typeName, "_")
	switch {
	case !isSingle:
		collectedRowsVar = fmt.Sprintf("%s[%s]", collectedRowsVar, indexName)
	}

	for _, child := range n {
		if len(child.Children) == 0 {
			continue
		}
		childType := fmt.Sprintf("%s_%s", typeName, child.Col.Name)
		childVar := fmt.Sprintf("%s.%s", collectedRowsVar, child.Col.Name)
		transformation.WriteString(child.Children.Transform(
			currPkg, i, types, cols,
			child.Single, childType, childVar,
			child.Col.Name+indexName,
		))
	}

	if nullable {
		transformation.WriteString("}\n")
	}

	return transformation.String()
}
