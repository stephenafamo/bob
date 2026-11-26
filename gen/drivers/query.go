package drivers

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/gen/language"
	"github.com/stephenafamo/bob/internal"
	"github.com/stephenafamo/bob/orm"
	"github.com/volatiletech/strmangle"
)

type TemplateInclude interface {
	IncludeInTemplate(language.Importer) string
}

type QueryFolder struct {
	Path  string      `json:"path"`
	Files []QueryFile `json:"files"`
}

func (q QueryFolder) Types() []string {
	var types []string
	for _, file := range q.Files {
		for _, query := range file.Queries {
			for _, col := range query.Columns {
				types = append(types, col.TypeName)
			}
			for _, arg := range query.Args {
				types = append(types, arg.Types()...)
			}
		}
	}

	slices.Sort(types)
	return slices.Compact(types)
}

type QueryFile struct {
	Path    string  `json:"path"`
	Queries []Query `json:"queries"`
}

func (q QueryFile) BaseName() string {
	if q.Path == "" {
		return ""
	}

	base := filepath.Base(q.Path)
	if base == "" {
		return ""
	}

	return base[:len(base)-4]
}

func (q QueryFile) Formatted() string {
	if len(q.Queries) == 0 {
		return ""
	}

	var sb strings.Builder
	for i, query := range q.Queries {
		if i > 0 {
			sb.WriteString("\n\n") // 2 extra new line between queries
		}
		fmt.Fprintf(&sb, "-- %s\n%s;", query.Name, query.SQL)
	}

	return sb.String()
}

func (q QueryFile) QueryPosition(i, headerLen int) string {
	if i >= len(q.Queries) {
		return "-1:-1"
	}

	position := headerLen
	for index, query := range q.Queries {
		if index > 0 {
			position += 3 // semi-colon and 2 new lines between queries
		}

		position += len(query.Name) + 4 // 2 dashes and a space before, and a newline after

		if index == i {
			return fmt.Sprintf("%d:%d", position, position+len(query.SQL))
		}

		position += len(query.SQL)
	}

	return fmt.Sprintf("%d:%d", position, position+len(q.Queries[i].SQL))
}

type Query struct {
	Name   string        `json:"name"`
	SQL    string        `json:"raw"`
	Type   bob.QueryType `json:"type"`
	Config QueryConfig   `json:"config"`

	Columns QueryCols       `json:"columns"`
	Args    []QueryArg      `json:"args"`
	Mods    TemplateInclude `json:"mods"`
}

func (q Query) ArgsByPosition() []orm.ArgWithPosition {
	var args []orm.ArgWithPosition

	for _, arg := range q.Args {
		for _, pos := range arg.Positions {
			args = append(args, orm.ArgWithPosition{
				Name:  arg.Col.Name,
				Start: pos[0],
				Stop:  pos[1],
			})
		}
	}

	slices.SortFunc(args, func(i, j orm.ArgWithPosition) int {
		if i.Start != j.Start {
			return i.Start - j.Start
		}

		return i.Stop - j.Stop
	})

	return args
}

func (q Query) MarshalJSON() ([]byte, error) {
	tmp := struct {
		Type bob.QueryType `json:"type"`
		Name string        `json:"name"`
		SQL  string        `json:"raw"`

		Config QueryConfig `json:"config"`

		Columns []QueryCol `json:"columns"`
		Args    []QueryArg `json:"args"`
		Mods    string     `json:"mods"`
	}{
		Type:    q.Type,
		Name:    q.Name,
		SQL:     q.SQL,
		Config:  q.Config,
		Columns: q.Columns,
		Args:    q.Args,
	}

	if q.Mods != nil {
		tmp.Mods = q.Mods.IncludeInTemplate(dummyImporter{})
	}

	return json.Marshal(tmp)
}

func (q Query) HasNonMultipleArgs() bool {
	for _, arg := range q.Args {
		if !arg.CanBeMultiple {
			return true
		}
	}

	return false
}

func (q Query) HasMultipleArgs() bool {
	for _, arg := range q.Args {
		if arg.CanBeMultiple {
			return true
		}
	}

	return false
}

type QueryConfig struct {
	ResultTypeOne     string `json:"result_type_one"`
	ResultTypeAll     string `json:"result_type_all"`
	ResultTransformer string `json:"result_type_transformer"`
}

func (q QueryConfig) Merge(other QueryConfig) QueryConfig {
	if other.ResultTypeOne != "" {
		q.ResultTypeOne = other.ResultTypeOne
	}

	if other.ResultTypeAll != "" {
		q.ResultTypeAll = other.ResultTypeAll
	}

	if other.ResultTransformer != "" {
		q.ResultTransformer = other.ResultTransformer
	}

	return q
}

type QueryCol struct {
	Name       string   `json:"name"`
	DBName     string   `json:"db_name"`
	Nullable   *bool    `json:"nullable"`
	TypeName   string   `json:"type"`
	TypeLimits []string `json:"type_limits"`
}

func (q QueryCol) Merge(others ...QueryCol) QueryCol {
	for _, other := range others {
		if other.Name != "" {
			q.Name = other.Name
		}

		if other.TypeName != "" {
			q.TypeName = other.TypeName
		}

		if other.Nullable != nil {
			q.Nullable = other.Nullable
		}
	}

	return q
}

type QueryCols []QueryCol

func (q QueryCols) WithNames() QueryCols {
	newCols := slices.Clone(q)
	names := make([]string, len(q))
	for i := range newCols {
		names[i] = strmangle.TitleCase(q[i].Name)
	}
	makeNamesDistinct(names, "")
	for i := range newCols {
		newCols[i].Name = names[i]
	}

	return newCols
}

func (q QueryCols) NameAt(i int) string {
	earlyDuplicates := 0
	for _, col := range q[:i] {
		if col.Name == q[i].Name {
			earlyDuplicates++
		}
	}

	if earlyDuplicates > 0 {
		return fmt.Sprintf("%s_%d", q[i].Name, earlyDuplicates+1)
	}

	return q[i].Name
}

type QueryArg struct {
	Col       QueryCol   `json:"col"`
	Children  []QueryArg `json:"children"`
	Positions [][2]int   `json:"positions"`

	CanBeMultiple bool `json:"can_be_multiple"`
}

func (q QueryArg) Types() []string {
	if len(q.Children) == 0 {
		return []string{q.Col.TypeName}
	}

	types := make([]string, 0, len(q.Children))
	for _, child := range q.Children {
		types = append(types, child.Types()...)
	}

	return types
}

func (c QueryCol) Type(currPkg string, i language.Importer, types Types) string {
	return c.getType(currPkg, i, types, true)
}

func (c QueryCol) TypeNoImport(currPkg string, i language.Importer, types Types) string {
	return c.getType(currPkg, i, types, false)
}

func (c QueryCol) getType(currPkg string, i language.Importer, types Types, doImport bool) string {
	if c.Nullable == nil {
		panic(fmt.Sprintf("Column %s has no nullable value defined", c.Name))
	}
	if !*c.Nullable {
		typ, def := types.GetNameAndDef(currPkg, c.TypeName)
		if doImport {
			i.ImportList(def.Imports)
		}
		return typ
	}

	typ, imports := types.GetNullTypeWithImports(currPkg, c.TypeName)
	if doImport {
		i.ImportList(imports)
	}
	return typ.Name
}

func (c QueryArg) RandomExpr(currPkg string, i language.Importer, types Types) string {
	typ := c.TypeDef(currPkg, i, types, false)
	var sb strings.Builder

	if c.CanBeMultiple {
		fmt.Fprintf(&sb, "[]%s{", typ)
	} else if len(c.Children) > 0 {
		sb.WriteString(typ)
	}

	if len(c.Children) == 0 {
		if c.Col.Nullable != nil && *c.Col.Nullable {
			colTyp, _ := types.GetNameAndDef(currPkg, c.Col.TypeName)
			nullTyp := types.GetNullType(currPkg, c.Col.TypeName)
			i.ImportList(nullTyp.CreateExprImports)
			normalized := internal.TypesReplacer.Replace(c.Col.TypeName)
			return strings.NewReplacer(
				"SRC", fmt.Sprintf("random_%s(nil)", normalized),
				"BASETYPE", colTyp,
				"NULLTYPE", nullTyp.Name,
				"NULLVAL", "true",
			).Replace(nullTyp.CreateExpr)
		} else {
			normalized := internal.TypesReplacer.Replace(c.Col.TypeName)
			fmt.Fprintf(&sb, "random_%s(nil)", normalized)
		}
	} else {
		sb.WriteString("{")
		for _, child := range c.Children {
			sb.WriteString(strmangle.TitleCase(child.Col.Name))
			sb.WriteString(": ")
			sb.WriteString(child.RandomExpr(currPkg, i, types))
			sb.WriteString(",\n")
		}
		sb.WriteString("}")
	}

	if c.CanBeMultiple {
		sb.WriteString("}")
	}

	return sb.String()
}

func (c QueryArg) Type(currPkg string, i language.Importer, types Types) string {
	if c.CanBeMultiple {
		return "[]" + c.TypeDef(currPkg, i, types, true)
	}
	return c.TypeDef(currPkg, i, types, true)
}

func (c QueryArg) TypeDef(currPkg string, i language.Importer, types Types, doImport bool) string {
	if len(c.Children) == 0 {
		return c.Col.getType(currPkg, i, types, doImport)
	}

	var sb strings.Builder
	sb.WriteString("struct{\n")
	for _, child := range c.Children {
		sb.WriteString(strmangle.TitleCase(child.Col.Name))
		sb.WriteString(" ")
		sb.WriteString(child.Type(currPkg, i, types))
		sb.WriteString("\n")
	}
	sb.WriteString("}")

	return sb.String()
}

func (a QueryArg) ToExpression(i language.Importer, dialect, queryName, varName string) string {
	if len(a.Children) == 0 {
		if a.CanBeMultiple {
			i.Import("github.com/stephenafamo/bob/expr")
			return fmt.Sprintf("expr.ToArgs(%s...)", varName)
		}

		i.Import("github.com/stephenafamo/bob/dialect/" + dialect)
		return fmt.Sprintf("%s.Arg(%s)", dialect, varName)
	}

	if !slices.ContainsFunc(a.Children, func(c QueryArg) bool { return c.CanBeMultiple || len(c.Children) > 0 }) {
		return a.scalarTuples(i, queryName, varName)
	}

	if !a.CanBeMultiple {
		return a.groupExpression(i, dialect, queryName, varName)
	}

	return fmt.Sprintf("genruntime.SQLGroups(len(%s), func(i int) bob.Expression { child := %s[i]; return %s })",
		varName, varName, a.groupExpression(i, dialect, queryName, "child"))
}

// A tuple of scalar parameters has a fixed layout. Emit its literal SQL
// spans once, even when the caller passes thousands of rows.
func (a QueryArg) scalarTuples(i language.Importer, queryName, varName string) string {
	i.Import("github.com/stephenafamo/bob/orm/genruntime")
	n, row := "1", varName
	if a.CanBeMultiple {
		n, row = "len("+varName+")", varName+"[i]"
	}
	fields := make([]string, 0, len(a.Children))
	spans := make([]string, 0, len(a.Children)+1)
	start := a.Positions[0][0]
	for _, child := range a.Children {
		pos := child.Positions[0]
		spans = append(spans, fmt.Sprintf("%sSQL[%d:%d]", queryName, start, pos[0]))
		fields = append(fields, row+"."+strmangle.TitleCase(child.Col.Name))
		start = pos[1]
	}
	spans = append(spans, fmt.Sprintf("%sSQL[%d:%d]", queryName, start, a.Positions[0][1]))
	return fmt.Sprintf("genruntime.SQLTuples(%s, func(args []any, i int) []any { return append(args, %s) }, %s)",
		n, strings.Join(fields, ", "), strings.Join(spans, ", "))
}

func (a QueryArg) groupExpression(i language.Importer, dialect, queryName, varName string) string {
	var sb strings.Builder
	i.Import("github.com/stephenafamo/bob/orm")
	i.Import("github.com/stephenafamo/bob/orm/genruntime")
	start, stop := a.Positions[0][0], a.Positions[0][1]
	// Bind child expressions at write time, including any expanding slices.
	fmt.Fprintf(&sb, "genruntime.SQLGroup(%sSQL[%d:%d], func() []orm.ArgWithPosition { return []orm.ArgWithPosition{\n", queryName, start, stop)
	for _, child := range a.Children {
		childName := strmangle.TitleCase(child.Col.Name)
		fmt.Fprintf(&sb, "{Name: %q, Start: %d, Stop: %d, Expression: %s},\n",
			child.Col.Name, child.Positions[0][0]-start, child.Positions[0][1]-start,
			child.ToExpression(i, dialect, queryName, varName+"."+childName))
	}
	sb.WriteString("} })")
	return sb.String()
}
