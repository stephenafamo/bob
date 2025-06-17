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
	types := []string{}
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

func (q QueryFile) QueryPosition(i int, headerLen int) string {
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
	names := make(map[string]int, len(newCols))
	for i := range newCols {
		if newCols[i].Name == "" {
			continue
		}
		name := strmangle.TitleCase(newCols[i].Name)
		index := names[name]
		names[name] = index + 1
		if index > 0 {
			name = fmt.Sprintf("%s%d", name, index+1)
		}
		newCols[i].Name = name
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
	if c.Nullable == nil {
		panic(fmt.Sprintf("Column %s has no nullable value defined", c.Name))
	}

	return types.GetNullable(currPkg, i, c.TypeName, *c.Nullable)
}

func (c QueryArg) RandomExpr(currPkg string, i language.Importer, types Types) string {
	typ := c.TypeDef(currPkg, i, types)
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
			normalized := internal.TypesReplacer.Replace(colTyp)
			return strings.NewReplacer(
				"SRC", fmt.Sprintf("random_%s(nil)", normalized),
				"BASETYPE", colTyp,
				"NULLTYPE", nullTyp.Name,
				"NULLVAL", "true",
			).Replace(nullTyp.CreateExpr)
		} else {
			normalized := internal.TypesReplacer.Replace(typ)
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
		return "[]" + c.TypeDef(currPkg, i, types)
	}
	return c.TypeDef(currPkg, i, types)
}

func (c QueryArg) TypeDef(currPkg string, i language.Importer, types Types) string {
	if len(c.Children) == 0 {
		return c.Col.Type(currPkg, i, types)
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
			return fmt.Sprintf("expr.ToArgs(%s...)", varName)
		}

		i.Import("github.com/stephenafamo/bob/dialect/" + dialect)
		return fmt.Sprintf("%s.Arg(%s)", dialect, varName)
	}

	if !a.CanBeMultiple {
		return a.groupExpression(i, dialect, queryName, varName)
	}

	groupExpression := a.groupExpression(i, dialect, queryName, "child")
	sb := strings.Builder{}
	sb.WriteString(fmt.Sprintf(`func() bob.Expression {
            expressions := make([]bob.Expression, len(%s))
            for i, child := range %s {
                expressions[i] = %s
            }
            return expr.Join{Exprs: expressions, Sep: ", "}
        }()
        `, varName, varName, groupExpression))

	return sb.String()
}

func (a QueryArg) groupExpression(i language.Importer, dialect, queryName, varName string) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf(`bob.ExpressionFunc(func(ctx context.Context, w io.Writer, d bob.Dialect, start int) ([]any, error) {
                  args := make([]any, 0, %d)`, len(a.Children)))

	start := a.Positions[0][0]
	for _, child := range a.Children {
		childName := strmangle.TitleCase(child.Col.Name)
		childExpression := child.ToExpression(i, dialect, queryName, fmt.Sprintf("%s.%s", varName, childName))
		sb.WriteString(fmt.Sprintf(`
            w.Write([]byte(%sSQL[%d:%d]))
            %sArgs, err := bob.Express(ctx, w, d, start+len(args), %s)
            if err != nil {
                return nil, err
            }
            args = append(args, %sArgs...)
            `,
			queryName, start, child.Positions[0][0],
			childName,
			strings.TrimSpace(childExpression),
			childName,
		))
		start = child.Positions[0][1]
	}

	sb.WriteString(fmt.Sprintf(`
            w.Write([]byte(%sSQL[%d:%d]))
            return args, nil
        })
    `, queryName, start, a.Positions[0][1]))

	return sb.String()
}
