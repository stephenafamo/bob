package parser

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/aarondl/opt/omit"
	"github.com/antlr4-go/antlr/v4"
	"github.com/stephenafamo/bob/gen/drivers"
	"github.com/stephenafamo/bob/internal"
	sqliteparser "github.com/stephenafamo/sqlparser/sqlite"
)

func New(t tables) Parser {
	return Parser{db: t}
}

type Parser struct {
	db tables
}

func (p Parser) ParseFolders(paths ...string) ([]drivers.QueryFolder, error) {
	allQueries := make([]drivers.QueryFolder, 0, len(paths))
	for _, path := range paths {
		queries, err := p.parseFolder(path)
		if err != nil {
			return nil, fmt.Errorf("parse folder: %w", err)
		}

		allQueries = append(allQueries, queries)
	}

	return allQueries, nil
}

func (p Parser) parseFolder(path string) (drivers.QueryFolder, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return drivers.QueryFolder{}, fmt.Errorf("read dir: %w", err)
	}

	files := make([]drivers.QueryFile, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if filepath.Ext(entry.Name()) != ".sql" {
			continue
		}

		file, err := p.parseFile(filepath.Join(path, entry.Name()))
		if err != nil {
			return drivers.QueryFolder{}, fmt.Errorf("parse file: %w", err)
		}

		files = append(files, file)
	}

	return drivers.QueryFolder{
		Path:  path,
		Files: files,
	}, nil
}

func (p Parser) parseFile(path string) (drivers.QueryFile, error) {
	file, err := os.ReadFile(path)
	if err != nil {
		return drivers.QueryFile{}, fmt.Errorf("read file: %w", err)
	}

	queries, err := p.parseMultiQueries(string(file))
	if err != nil {
		return drivers.QueryFile{}, fmt.Errorf("parse multi queries: %w", err)
	}

	return drivers.QueryFile{
		Path:    path,
		Queries: queries,
	}, nil
}

func (p Parser) parseMultiQueries(s string) ([]drivers.Query, error) {
	v := NewVisitor(p.db)
	infos, err := p.parse(v, s)
	if err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}

	queries := make([]drivers.Query, len(infos))
	for i, info := range infos {
		stmtStart := info.stmt.GetStart().GetStart()
		stmtStop := info.stmt.GetStop().GetStop()
		formatted, err := internal.EditStringSegment(s, stmtStart, stmtStop, info.editRules...)
		if err != nil {
			return nil, fmt.Errorf("format: %w", err)
		}

		cols := make([]drivers.QueryCol, len(info.columns))
		for i, col := range info.columns {
			cols[i] = drivers.QueryCol{
				Name:     col.name,
				DBName:   col.name,
				Nullable: omit.From(col.typ.Nullable()),
				TypeName: col.typ.Type(p.db),
			}.Merge(col.config)
		}

		args := v.getArgs(stmtStart, stmtStop)
		keys := make(map[string]int, len(args))
		names := make(map[string]int, len(args))
		queryArgs := make([]drivers.QueryArg, 0, len(args))
		for _, arg := range args {
			// If the key is already in the map, append the position
			key := arg.queryArgKey
			if oldIndex, ok := keys[key]; ok && key != "" {
				queryArgs[oldIndex].Positions = append(
					queryArgs[oldIndex].Positions, arg.EditedPosition,
				)
				continue
			}
			keys[arg.queryArgKey] = len(queryArgs)

			name := v.getNameString(arg.expr)
			index := names[name]
			names[name] = index + 1
			if index > 0 {
				name = fmt.Sprintf("%s_%d", name, index+1)
			}

			queryArgs = append(queryArgs, drivers.QueryArg{
				Col: drivers.QueryCol{
					Name:     name,
					Nullable: omit.From(arg.Type.Nullable()),
					TypeName: v.getDBType(arg).Type(p.db),
				}.Merge(arg.config),
				Positions:     [][2]int{arg.EditedPosition},
				CanBeMultiple: arg.CanBeMultiple,
			})
		}

		name, configStr, _ := strings.Cut(info.comment, " ")
		queries[i] = drivers.Query{
			Name: name,
			SQL:  formatted,
			Type: info.queryType,

			Config: drivers.QueryConfig{
				RowName:      info.comment + "Row",
				RowSliceName: "",
				GenerateRow:  true,
			}.Merge(drivers.ParseQueryConfig(configStr)),

			Columns: cols,
			Args:    groupArgs(queryArgs),
			Mods:    stmtToMod{info},
		}
	}

	return queries, nil
}

func (Parser) parse(v *visitor, input string) ([]stmtInfo, error) {
	el := &errorListener{}

	// Get all hidden tokens (usually comments) and add edit rules to remove them
	v.baseRules = []internal.EditRule{}
	hiddenLexer := sqliteparser.NewSQLiteLexer(antlr.NewInputStream(input))
	hiddenStream := antlr.NewCommonTokenStream(hiddenLexer, 1)
	hiddenStream.Fill()
	for _, token := range hiddenStream.GetAllTokens() {
		switch token.GetTokenType() {
		case sqliteparser.SQLiteParserSINGLE_LINE_COMMENT,
			sqliteparser.SQLiteParserMULTILINE_COMMENT:
			v.baseRules = append(
				v.baseRules,
				internal.Delete(token.GetStart(), token.GetStop()),
			)
		}
	}

	// Get the regular tokens (usually the SQL statement)
	lexer := sqliteparser.NewSQLiteLexer(antlr.NewInputStream(input))
	stream := antlr.NewCommonTokenStream(lexer, 0)
	sqlParser := sqliteparser.NewSQLiteParser(stream)
	sqlParser.AddErrorListener(el)

	tree := sqlParser.Parse()
	if el.err != "" {
		return nil, errors.New(el.err)
	}

	infos, ok := tree.Accept(v).([]stmtInfo)
	if v.err != nil {
		return nil, fmt.Errorf("visitor: %w", v.err)
	}

	if !ok {
		return nil, fmt.Errorf("visitor: expected stmtInfo, got %T", infos)
	}

	return infos, nil
}

type stmtToMod struct {
	info stmtInfo
}

func (s stmtToMod) IncludeInTemplate(i drivers.Importer) string {
	for _, im := range s.info.imports {
		i.Import(im...)
	}
	return s.info.mods.String()
}

type errorListener struct {
	*antlr.DefaultErrorListener

	err string
}

func (el *errorListener) SyntaxError(recognizer antlr.Recognizer, offendingSymbol any, line, column int, msg string, e antlr.RecognitionException) {
	el.err = msg
}

func groupArgs(args []drivers.QueryArg) []drivers.QueryArg {
	newArgs := make([]drivers.QueryArg, 0, len(args))

Outer:
	for i, arg := range args {
		if len(arg.Positions) != 1 {
			newArgs = append(newArgs, args[i])
			continue
		}

		for j, arg2 := range args {
			if i == j {
				continue
			}

			if len(arg2.Positions) != 1 {
				continue
			}

			if arg2.Positions[0][0] <= arg.Positions[0][0] &&
				arg2.Positions[0][1] >= arg.Positions[0][1] {
				// arg2 is a parent of arg
				// since arg1 has a parent, it should be skipped
				continue Outer
			}

			if arg.Positions[0][0] <= arg2.Positions[0][0] &&
				arg.Positions[0][1] >= arg2.Positions[0][1] {
				// arg is a parent of arg2
				args[i].Children = append(args[i].Children, arg2)
			}
		}

		newArgs = append(newArgs, args[i])
	}

	return newArgs
}

//nolint:gochecknoglobals
var defaultFunctions = functions{
	"abs": {
		requiredArgs: 1,
		args:         []string{""},
		calcReturnType: func(args ...string) string {
			if args[0] == "INTEGER" {
				return "INTEGER"
			}
			return "REAL"
		},
	},
	"changes": {
		returnType: "INTEGER",
	},
	"char": {
		requiredArgs: 1,
		variadic:     true,
		args:         []string{"INTEGER"},
		returnType:   "TEXT",
	},
	"coalesce": {
		requiredArgs:         1,
		variadic:             true,
		args:                 []string{""},
		shouldArgsBeNullable: true,
		calcReturnType: func(args ...string) string {
			for _, arg := range args {
				if arg != "" {
					return arg
				}
			}
			return ""
		},
		calcNullable: allNullable,
	},
	"concat": {
		requiredArgs: 1,
		variadic:     true,
		args:         []string{"TEXT"},
		returnType:   "TEXT",
		calcNullable: neverNullable,
	},
	"concat_ws": {
		requiredArgs: 2,
		variadic:     true,
		args:         []string{"TEXT", "TEXT"},
		returnType:   "TEXT",
		calcNullable: func(args ...func() bool) func() bool {
			return args[0]
		},
	},
	"format": {
		requiredArgs: 2,
		variadic:     true,
		args:         []string{"TEXT", ""},
		returnType:   "TEXT",
		calcNullable: func(args ...func() bool) func() bool {
			return args[0]
		},
	},
	"glob": {
		requiredArgs: 2,
		args:         []string{"TEXT", "TEXT"},
		returnType:   "BOOLEAN",
	},
	"hex": {
		requiredArgs: 1,
		args:         []string{""},
		returnType:   "TEXT",
	},
	"ifnull": {
		requiredArgs: 2,
		args:         []string{""},
		calcReturnType: func(args ...string) string {
			for _, arg := range args {
				if arg != "" {
					return arg
				}
			}
			return ""
		},
		calcNullable: allNullable,
	},
	"iif": {
		requiredArgs: 3,
		args:         []string{"BOOLEAN", "", ""},
		calcReturnType: func(args ...string) string {
			return args[1]
		},
		calcNullable: func(args ...func() bool) func() bool {
			return anyNullable(args[1], args[2])
		},
	},
}
