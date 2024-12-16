package driver

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/antlr4-go/antlr/v4"
	"github.com/stephenafamo/bob/gen/drivers"
	"github.com/stephenafamo/bob/internal"
	sqliteparser "github.com/stephenafamo/sqlparser/sqlite"
)

func newParser(t tables) Parser {
	return Parser{db: t}
}

type Parser struct {
	db tables
}

func (p Parser) parseFolders(paths ...string) ([]drivers.QueryFolder, error) {
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
	v := &visitor{
		db:        p.db,
		exprs:     make(map[nodeKey]exprInfo),
		names:     make(map[nodeKey]exprName),
		functions: defaultFunctions,
	}
	input := antlr.NewInputStream(s)

	infos, err := p.parse(v, input)
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

		cols := make([]drivers.QueryArg, len(info.columns))
		for i, col := range info.columns {
			refs := make([]drivers.Ref, 0, len(col.typ))
			for _, typ := range col.typ {
				for _, ref := range typ.refs {
					refs = append(refs, drivers.Ref{
						Key:    ref.key(),
						Column: ref.column,
					})
				}
			}

			cols[i] = drivers.QueryArg{
				Name:     col.name,
				Nullable: col.typ.Nullable(),
				TypeName: translateColumnType(col.typ.ConfirmedAffinity()),
				Refs:     refs,
			}
		}

		args := make([]drivers.QueryArg, len(info.args))
		for i, arg := range info.args {
			refs := make([]drivers.Ref, 0, len(arg.DBType))
			for _, typ := range arg.DBType {
				for _, ref := range typ.refs {
					refs = append(refs, drivers.Ref{
						Key:    ref.key(),
						Column: ref.column,
					})
				}
			}

			args[i] = drivers.QueryArg{
				Name:     v.getNameString(arg.expr),
				Nullable: arg.DBType.Nullable(),
				TypeName: translateColumnType(arg.DBType.ConfirmedAffinity()),
				Refs:     refs,
			}
		}

		queries[i] = drivers.Query{
			Name:        info.comment,
			SQL:         formatted,
			RowName:     info.comment + "Row",
			GenerateRow: true,
			Columns:     cols,
			Args:        args,
		}
	}

	return queries, nil
}

func (p Parser) parse(v *visitor, input *antlr.InputStream) ([]stmtInfo, error) {
	lexer := sqliteparser.NewSQLiteLexer(input)
	stream := antlr.NewCommonTokenStream(lexer, 0)
	sqlParser := sqliteparser.NewSQLiteParser(stream)

	el := &errorListener{}
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

type errorListener struct {
	*antlr.DefaultErrorListener

	err string
}

func (el *errorListener) SyntaxError(recognizer antlr.Recognizer, offendingSymbol any, line, column int, msg string, e antlr.RecognitionException) {
	el.err = msg
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
