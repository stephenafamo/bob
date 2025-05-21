package drivers

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/aarondl/opt/omit"
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/gen/language"
	"github.com/stephenafamo/bob/orm"
	"github.com/volatiletech/strmangle"
)

type TemplateInclude interface {
	IncludeInTemplate(language.Importer) string
}

type QueryFolder struct {
	Path  string
	Files []QueryFile
}

type QueryFile struct {
	Path    string
	Queries []Query
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

	Columns []QueryCol      `json:"columns"`
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
	RowName      string `json:"row_name"`
	RowSliceName string `json:"row_slice_name"`
	GenerateRow  bool   `json:"generate_row"`
}

func (q QueryConfig) Merge(other QueryConfig) QueryConfig {
	if other.RowName != "" {
		q.RowName = other.RowName
	}

	if other.RowSliceName != "" {
		q.RowSliceName = other.RowSliceName
	}

	q.GenerateRow = q.GenerateRow && other.GenerateRow

	return q
}

type QueryCol struct {
	Name     string         `json:"name"`
	DBName   string         `json:"db_name"`
	Nullable omit.Val[bool] `json:"nullable"`
	TypeName string         `json:"type"`
}

func (q QueryCol) Merge(others ...QueryCol) QueryCol {
	for _, other := range others {
		if other.Name != "" {
			q.Name = other.Name
		}

		if other.TypeName != "" {
			q.TypeName = other.TypeName
		}

		if other.Nullable.IsSet() {
			q.Nullable = other.Nullable
		}
	}

	return q
}

type QueryArg struct {
	Col       QueryCol   `json:"col"`
	Children  []QueryArg `json:"children"`
	Positions [][2]int   `json:"positions"`

	CanBeMultiple bool `json:"can_be_multiple"`
}

func (c QueryCol) Type(i language.Importer, types Types) string {
	typ := c.TypeName

	typDef, ok := types[typ]
	if ok && typDef.AliasOf != "" {
		typ = typDef.AliasOf
	}

	i.ImportList(typDef.Imports)
	if c.Nullable.MustGet() {
		i.Import("github.com/aarondl/opt/null")
		typ = fmt.Sprintf("null.Val[%s]", typ)
	}

	return typ
}

func (c QueryArg) Type(i language.Importer, types Types) string {
	if c.CanBeMultiple {
		return "[]" + c.TypeDef(i, types)
	}
	return c.TypeDef(i, types)
}

func (c QueryArg) TypeDef(i language.Importer, types Types) string {
	if len(c.Children) == 0 {
		return c.Col.Type(i, types)
	}

	var sb strings.Builder
	sb.WriteString("struct{\n")
	for _, child := range c.Children {
		sb.WriteString(strmangle.TitleCase(child.Col.Name))
		sb.WriteString(" ")
		sb.WriteString(child.Type(i, types))
		sb.WriteString("\n")
	}
	sb.WriteString("}")

	return sb.String()
}

func (a QueryArg) ToExpression(dialect, queryName, varName string) string {
	if len(a.Children) == 0 {
		if a.CanBeMultiple {
			return fmt.Sprintf("expr.ToArgs(%s...)", varName)
		}

		return fmt.Sprintf("%s.Arg(%s)", dialect, varName)
	}

	if !a.CanBeMultiple {
		return a.groupExpression(dialect, queryName, varName)
	}

	groupExpression := a.groupExpression(dialect, queryName, "child")
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

func (a QueryArg) groupExpression(dialect, queryName, varName string) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf(`bob.ExpressionFunc(func(ctx context.Context, w io.Writer, d bob.Dialect, start int) ([]any, error) {
                  args := make([]any, 0, %d)`, len(a.Children)))

	start := a.Positions[0][0]
	for _, child := range a.Children {
		childName := strmangle.TitleCase(child.Col.Name)
		childExpression := child.ToExpression(dialect, queryName, fmt.Sprintf("%s.%s", varName, childName))
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
