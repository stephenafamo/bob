package drivers

import (
	"fmt"
	"slices"
	"strings"

	"github.com/stephenafamo/bob/gen/language"
	"github.com/stephenafamo/bob/orm"
)

type Tables[C, I any] []Table[C, I]

// GetTable by name. Panics if not found (for use in templates mostly).
func (tables Tables[C, I]) Get(name string) Table[C, I] {
	for _, table := range tables {
		if table.Key == name {
			return table
		}
	}

	panic(fmt.Sprintf("could not find table name: %s", name))
}

func (tables Tables[C, I]) GetColumn(table string, column string) Column {
	for _, t := range tables {
		if t.Key != table {
			continue
		}

		return t.GetColumn(column)
	}

	panic("unknown table " + table)
}

func (tables Tables[C, I]) ColumnSetter(currPkg string, i language.Importer, types Types, table, column, val, nullVal string) string {
	for _, t := range tables {
		if t.Key != table {
			continue
		}

		col := t.GetColumn(column)
		if !col.Nullable {
			return val
		}

		colTyp, _ := types.GetNameAndDef(currPkg, col.Type)
		nullType := types.GetNullType(currPkg, col.Type)

		i.ImportList(nullType.CreateExprImports)
		return strings.NewReplacer(
			"SRC", val,
			"BASETYPE", colTyp,
			"NULLTYPE", nullType.Name,
			"NULLVAL", nullVal,
		).Replace(nullType.CreateExpr)
	}

	panic("unknown table " + table)
}

func (tables Tables[C, I]) ColumnGetter(currPkg string, i language.Importer, types Types, table, column, varName string) string {
	for _, t := range tables {
		if t.Key != table {
			continue
		}

		col := t.GetColumn(column)
		if !col.Nullable {
			return varName
		}

		colTyp, _ := types.GetNameAndDef(currPkg, col.Type)
		nullType := types.GetNullType(currPkg, col.Type)

		i.ImportList(nullType.UseExprImports)
		return strings.NewReplacer(
			"SRC", varName,
			"BASETYPE", colTyp,
			"NULLTYPE", nullType.Name,
			"NULLVAL", "true",
		).Replace(nullType.UseExpr)
	}

	panic("unknown table " + table)
}

// The source is never optional, but the destination can be
func (tables Tables[C, I]) ColumnAssigner(
	currentPkg string, i language.Importer, types Types, aliases Aliases,
	destTName, srcTName string,
	destColName, srcColName string,
	varName string, destOpt bool,
) string {
	src := fmt.Sprintf("%s.%s", varName, aliases[srcTName].Columns[srcColName])
	srcTable := tables.Get(srcTName)
	srcCol := srcTable.GetColumn(srcColName)

	destTable := tables.Get(destTName)
	destCol := destTable.GetColumn(destColName)

	nullType := types.GetNullType(currentPkg, destCol.Type)
	typeReplacer := strings.NewReplacer(
		"BASETYPE", destCol.Type,
		"NULLTYPE", nullType.Name,
		"NULLVAL", "true",
		"SRC", src,
	)

	//-----------------------------------------------------------------------
	// This switch handles the cases when we
	// don't need the nullable type information
	//-----------------------------------------------------------------------
	switch {
	case destOpt:
		return types.ToOptional(currentPkg, i, srcCol.Type, src, destCol.Nullable, srcCol.Nullable)

	// Same nullability, dest IS NOT optional
	case !destOpt && (srcCol.Nullable == destCol.Nullable):
		return src

	case !destOpt && destCol.Nullable && !srcCol.Nullable:
		i.ImportList(nullType.CreateExprImports)
		return typeReplacer.Replace(nullType.CreateExpr)

	case !destOpt && !destCol.Nullable && srcCol.Nullable:
		i.ImportList(nullType.UseExprImports)
		return typeReplacer.Replace(nullType.UseExpr)

	default:
		panic(fmt.Sprintf("unknown column assign case: %s.%s -> %s.%s", destTName, destColName, srcTName, srcColName))
	}
	//=======================================================================
}

func (tables Tables[C, I]) NeededBridgeRels(r orm.Relationship) []struct {
	Table    string
	Position int
	Many     bool
} {
	ma := []struct {
		Table    string
		Position int
		Many     bool
	}{}

	for _, side := range r.ValuedSides() {
		if side.TableName == r.Local() {
			continue
		}
		if side.TableName == r.Foreign() {
			continue
		}
		if side.TableName == "" {
			continue
		}

		sideTable := tables.Get(side.TableName)
		if sideTable.IsJoinTableForRel(r, side.Position) {
			continue
		}

		shouldAdd := false

		table := tables.Get(side.TableName)
		for _, col := range table.Columns {
			if col.Generated {
				continue
			}
			if slices.Contains(side.Columns(), col.Name) {
				continue
			}

			shouldAdd = true
			break
		}

		if !shouldAdd {
			continue
		}

		ma = append(ma, struct {
			Table    string
			Position int
			Many     bool
		}{
			Table:    side.TableName,
			Position: side.Position,
			Many:     r.NeedsMany(side.Position),
		})

	}

	return ma
}

func (tables Tables[C, I]) RelArgs(aliases Aliases, r orm.Relationship) string {
	ma := []string{}
	for _, need := range tables.NeededBridgeRels(r) {
		ma = append(ma, fmt.Sprintf(
			"%s%d,", aliases[need.Table].DownSingular, need.Position,
		))
	}

	return strings.Join(ma, "")
}

func (tables Tables[C, I]) RelDependencies(aliases Aliases, r orm.Relationship, preSuf ...string) string {
	var prefix, suffix string
	if len(preSuf) > 0 {
		prefix = preSuf[0]
	}
	if len(preSuf) > 1 {
		suffix = preSuf[1]
	}
	ma := []string{}
	for _, need := range tables.NeededBridgeRels(r) {
		alias := aliases[need.Table]
		ma = append(ma, fmt.Sprintf(
			"%s *%s%s%s,", alias.DownSingular, alias.UpSingular, prefix, suffix,
		))
	}

	return strings.Join(ma, "")
}

func (tables Tables[C, I]) RelDependenciesPos(aliases Aliases, r orm.Relationship) string {
	needed := tables.NeededBridgeRels(r)
	ma := make([]string, len(needed))

	for i, need := range needed {
		alias := aliases[need.Table]
		if need.Many {
			ma[i] = fmt.Sprintf(
				"%s%d %sSlice,", alias.DownPlural, need.Position, alias.UpSingular,
			)
		} else {
			ma[i] = fmt.Sprintf(
				"%s%d *%s,", alias.DownSingular, need.Position, alias.UpSingular,
			)
		}
	}

	return strings.Join(ma, "")
}

func (tables Tables[C, I]) RelDependenciesTyp(aliases Aliases, r orm.Relationship) string {
	ma := []string{}

	for _, need := range tables.NeededBridgeRels(r) {
		alias := aliases[need.Table]
		ma = append(ma, fmt.Sprintf("%s *%sTemplate", alias.DownSingular, alias.UpSingular))
	}

	return strings.Join(ma, "\n")
}

func (tables Tables[C, I]) RelDependenciesTypSet(aliases Aliases, r orm.Relationship) string {
	ma := []string{}

	for _, need := range tables.NeededBridgeRels(r) {
		alias := aliases[need.Table]
		ma = append(ma, fmt.Sprintf("%s: %s,", alias.DownSingular, alias.DownSingular))
	}

	return strings.Join(ma, "\n")
}

func (tables Tables[C, I]) SetFactoryDeps(currPkg string, i language.Importer, types Types, aliases Aliases, r orm.Relationship, inLoop bool) string {
	local := r.Local()
	foreign := r.Foreign()
	ksides := r.ValuedSides()

	ret := make([]string, 0, len(ksides))
	for _, kside := range ksides {
		switch kside.TableName {
		case local, foreign:
		default:
			continue
		}

		mret := make([]string, 0, len(kside.Mapped))

		for _, mapp := range kside.Mapped {
			switch mapp.ExternalTable {
			case local, foreign:
			default:
				continue
			}

			oalias := aliases[kside.TableName]
			objVarName := getVarName(aliases, kside.TableName, kside.Start, kside.End, false)

			if mapp.Value != [2]string{} {
				oGetter := tables.ColumnGetter(currPkg, i, types, kside.TableName, mapp.Column, objVarName+"."+oalias.Column(mapp.Column))

				if kside.TableName == r.Local() {
					i.Import("github.com/stephenafamo/bob/orm")
					mret = append(mret, fmt.Sprintf(`if %s != %s {
								return &orm.RelationshipChainError{
									Table1: %q, Column1: %q, Value: %q,
								}
							}`,
						oGetter, mapp.Value[1],
						kside.TableName, mapp.Column, mapp.Value[1],
					))
					continue
				}

				mret = append(mret, fmt.Sprintf(`%s.%s = %s //h`,
					objVarName,
					oalias.Columns[mapp.Column],
					mapp.Value[1],
				))
				continue
			}

			extObjVarName := getVarName(aliases, mapp.ExternalTable, mapp.ExternalStart, mapp.ExternalEnd, false)

			oSetter := tables.ColumnAssigner(
				currPkg, i, types, aliases,
				kside.TableName, mapp.ExternalTable,
				mapp.Column, mapp.ExternalColumn,
				extObjVarName, false)

			mret = append(mret, fmt.Sprintf(`%s.%s = %s //h2`,
				objVarName,
				oalias.Columns[mapp.Column],
				oSetter,
			))
		}

		ret = append(ret, strings.Join(mret, "\n"))
	}

	return strings.Join(ret, "\n")
}

func (tables Tables[C, I]) RelIsView(rel orm.Relationship) bool {
	for _, s := range rel.Sides {
		t := tables.Get(s.To)
		if t.Constraints.Primary == nil {
			return true
		}
	}

	return false
}

func getVarName(aliases Aliases, tableName string, local, foreign, many bool) string {
	switch {
	case foreign:
		if many {
			return "rels"
		}
		return "rel"

	case local:
		return "o"

	default:
		alias := aliases[tableName]
		if many {
			return alias.DownPlural
		}
		return alias.DownSingular
	}
}

type dummyImporter struct{}

func (dummyImporter) Import(...string) string    { return "" }
func (dummyImporter) ImportList([]string) string { return "" }
func (dummyImporter) ToList() []string           { return nil }
