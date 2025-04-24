package drivers

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/aarondl/opt/omit"
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/orm"
	"github.com/volatiletech/strmangle"
)

type TemplateInclude interface {
	IncludeInTemplate(Importer) string
}

type QueryFolder struct {
	Path  string
	Files []QueryFile
}

type QueryFile struct {
	Path    string
	Queries []Query
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

// ParseQueryConfig parses a user configuration string into a QueryCoonfig.
// The configuration string should be in the format:
// "row_name:row_slice_name:generate_row"
func ParseQueryConfig(options string) QueryConfig {
	var i int
	var part string
	var found bool

	col := QueryConfig{
		GenerateRow: true,
	}
	for {
		part, options, found = strings.Cut(options, ":")
		switch i {
		case 0:
			col.RowName = part
		case 1:
			col.RowSliceName = part
		case 2:
			switch part {
			case "true", "yes":
				col.GenerateRow = true
			case "false", "no", "skip":
				col.GenerateRow = false
			}
		}
		if !found {
			break
		}
		i++
	}

	return col
}

// ParseQueryColumnConfig parses a user configuration string into a QueryCol.
// The configuration string should be in the format:
// "name:type:notnull"
func ParseQueryColumnConfig(options string) QueryCol {
	var i int
	var part string
	var found bool

	col := QueryCol{}
	for {
		part, options, found = strings.Cut(options, ":")
		switch i {
		case 0:
			col.Name = part
		case 1:
			col.TypeName = part
		case 2:
			switch part {
			case "null", "true", "yes":
				col.Nullable.Set(true)
			case "notnull", "nnull", "false", "no":
				col.Nullable.Set(false)
			}
		}
		if !found {
			break
		}
		i++
	}

	return col
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

func (c QueryCol) Type(i Importer, types Types) string {
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

func (c QueryArg) Type(i Importer, types Types) string {
	if c.CanBeMultiple {
		return "[]" + c.TypeDef(i, types)
	}
	return c.TypeDef(i, types)
}

func (c QueryArg) TypeDef(i Importer, types Types) string {
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
			child.ToExpression(dialect, queryName, fmt.Sprintf("%s.%s", varName, childName)),
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

type QueryParser interface {
	ParseQueries(ctx context.Context, s string) ([]Query, error)
}

func ParseFolders(ctx context.Context, parser QueryParser, paths ...string) ([]QueryFolder, error) {
	allQueries := make([]QueryFolder, 0, len(paths))
	for _, path := range paths {
		queries, err := parseFolder(ctx, parser, path)
		if err != nil {
			return nil, fmt.Errorf("parse folder: %w", err)
		}

		allQueries = append(allQueries, queries)
	}

	return allQueries, nil
}

func parseFolder(ctx context.Context, parser QueryParser, path string) (QueryFolder, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return QueryFolder{}, fmt.Errorf("read dir: %w", err)
	}

	files := make([]QueryFile, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if filepath.Ext(entry.Name()) != ".sql" {
			continue
		}

		file, err := parseFile(ctx, parser, filepath.Join(path, entry.Name()))
		if err != nil {
			return QueryFolder{}, fmt.Errorf("parse file: %w", err)
		}

		files = append(files, file)
	}

	return QueryFolder{
		Path:  path,
		Files: files,
	}, nil
}

func parseFile(ctx context.Context, parser QueryParser, path string) (QueryFile, error) {
	file, err := os.ReadFile(path)
	if err != nil {
		return QueryFile{}, fmt.Errorf("read file: %w", err)
	}

	queries, err := parser.ParseQueries(ctx, string(file))
	if err != nil {
		return QueryFile{}, fmt.Errorf("parse multi queries: %w", err)
	}

	return QueryFile{
		Path:    path,
		Queries: queries,
	}, nil
}
