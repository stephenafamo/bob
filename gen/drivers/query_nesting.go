package drivers

import (
	"fmt"
	"strings"

	"github.com/stephenafamo/bob/gen/language"
	"github.com/volatiletech/strmangle"
)

// NestedColumns returns a list of nested columns in the query.
// Nesting is determined by the presence of a dot in the column's DBName.
// For example, if a column's DBName is "user.address.street", it will be nested under "user" and "address".
func (q Query) NestedColumns() nestedSlice {
	final := make(nestedSlice, 0, len(q.Columns))

	for colIndex, col := range q.Columns {
		names := strings.Split(col.Name, ".")
		col.Name = names[len(names)-1] // Use the last part of the name as the column name

		parts := strings.Split(col.DBName, ".")

		current := &final
		for partIndex, part := range parts {
			// If this is the last part, we need to assign the column index and column itself
			if partIndex == len(parts)-1 {
				*current = append(*current, nested{Index: colIndex, Col: col})
				continue
			}

			// Find the node for the current part in the *current slice
			var foundNode *nested
			for i := range *current {
				if (*current)[i].Col.Name == part {
					foundNode = &(*current)[i]
					break
				}
			}

			if foundNode != nil {
				// Node was found, so we just move deeper into its children for the next iteration
				current = &foundNode.Children
			} else {
				// Node was not found, create a new one
				*current = append(*current, nested{Col: QueryCol{Name: part}})
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

func (n nested) Compare(currPkg string, types Types, selfName, rowName string, cols []QueryCol) string {
	if len(n.Children) > 0 {
		return ""
	}

	lhs := fmt.Sprintf("%s.%s", selfName, n.Col.Name)
	rhs := fmt.Sprintf("%s.%s", rowName, cols[n.Index].Name)
	_, typDef := types.GetNameAndDef(currPkg, n.Col.TypeName)
	cmpExpr := typDef.CompareExpr
	if cmpExpr == "" {
		cmpExpr = "AAA == BBB"
	}

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
	// Fix duplicate names in the slice
	names := make(map[string]int, len(n))
	for i := range n {
		name := strmangle.TitleCase(n[i].Col.Name)
		index := names[name]
		names[name] = index + 1
		if index > 0 {
			name = fmt.Sprintf("%s%d", name, index+1)
		}

		n[i].Col.Name = name
		n[i].Children.FixNames()
	}

	return n
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
	allTypes := []string{""}

	var self strings.Builder
	fmt.Fprintf(&self, "type %s = struct{\n", typeName)
	for _, child := range n {
		childType := child.Type(currPkg, i, types, typeName)

		if len(child.Children) > 0 {
			allTypes = append(allTypes, child.Children.Types(
				currPkg, i, types, childType,
			)...)
			childType = "[]" + childType
		}

		fmt.Fprintf(&self, "%s %s\n",
			child.Col.Name,
			childType,
		)

	}

	self.WriteString("}")

	allTypes[0] = self.String()
	return allTypes
}

func (n nestedSlice) Assign(currPkg string, i language.Importer, types Types, selfName, rowName string, cols []QueryCol) string {
	assigns := make([]string, 0, len(n))
	for _, child := range n {
		assigns = append(assigns, child.Assign(selfName, rowName, cols))
	}

	return strings.Join(assigns, "\n")
}

func (n nestedSlice) Compare(currPkg string, types Types, selfName, rowName string, cols []QueryCol) string {
	assigns := make([]string, 0, len(n))
	for _, child := range n {
		assign := child.Compare(currPkg, types, selfName, rowName, cols)
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

func (n nestedSlice) Transform(currPkg string, i language.Importer, types Types, cols []QueryCol, typeName string, collectedRowsVar, indexName string) string {
	nullable := n.Nullable()
	transformation := &strings.Builder{}

	if nullable {
		fmt.Fprintf(transformation, "\nif %s {", n.NotNull(currPkg, types, "row", cols))
	}

	fmt.Fprintf(transformation, `
        %s := -1
        for i, existing := range %s {
          if %s {
            %s = i
            break
          }
        }

        if %s == -1 {
          fresh := %s{}
          %s
          %s = append(%s, fresh)
          %s = len(%s) - 1
        }

		`,
		indexName, collectedRowsVar,
		n.Compare(currPkg, types, "existing", "row", cols),
		indexName, indexName, typeName,
		n.Assign(currPkg, i, types, "fresh", "row", cols),
		collectedRowsVar, collectedRowsVar, indexName, collectedRowsVar)

	for _, child := range n {
		if len(child.Children) == 0 {
			continue
		}
		childType := fmt.Sprintf("%s_%s", typeName, child.Col.Name)
		transformation.WriteString(child.Children.Transform(
			currPkg, i, types, cols, childType,
			fmt.Sprintf("%s[%s].%s", collectedRowsVar, indexName, child.Col.Name),
			child.Col.Name+indexName,
		))
	}

	if nullable {
		transformation.WriteString("\n}\n")
	}

	return transformation.String()
}
