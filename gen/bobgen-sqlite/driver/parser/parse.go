package parser

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/stephenafamo/bob/gen/bobgen-helpers/parser"
	antlrhelpers "github.com/stephenafamo/bob/gen/bobgen-helpers/parser/antlrhelpers"
	"github.com/stephenafamo/bob/gen/drivers"
	"github.com/stephenafamo/bob/gen/language"
	"github.com/stephenafamo/bob/internal"
	sqliteparser "github.com/stephenafamo/sqlparser/sqlite"
)

func New(t tables, driver string) Parser {
	return Parser{db: t, driver: driver}
}

type Parser struct {
	db     tables
	driver string
}

func (p Parser) ParseFolders(ctx context.Context, paths ...string) ([]drivers.QueryFolder, error) {
	return parser.ParseFolders(ctx, p, paths...)
}

func (p Parser) ParseQueries(_ context.Context, s string) ([]drivers.Query, error) {
	v := NewVisitor(p.db)
	infos, err := p.parse(v, s)
	if err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}

	queries := make([]drivers.Query, len(infos))
	for i, info := range infos {
		stmtStart := info.Node.GetStart().GetStart()
		stmtStop := info.Node.GetStop().GetStop()
		formatted, err := internal.EditStringSegment(s, stmtStart, stmtStop, info.EditRules...)
		if err != nil {
			return nil, fmt.Errorf("format: %w", err)
		}

		cols := make([]drivers.QueryCol, len(info.Columns))
		for i, col := range info.Columns {
			cols[i] = drivers.QueryCol{
				Name:     col.Name,
				DBName:   col.Name,
				Nullable: internal.Pointer(col.Type.Nullable()),
				TypeName: TranslateColumnType(col.Type.ConfirmedDBType(), p.driver),
			}.Merge(col.Config)
		}

		name, configStr, _ := strings.Cut(info.Comment, " ")
		queries[i] = drivers.Query{
			Name:    name,
			SQL:     formatted,
			Type:    info.QueryType,
			Config:  parser.ParseQueryConfig(configStr),
			Columns: cols,
			Args: v.GetArgs(stmtStart, stmtStop, func(s string) (string, []string) {
				return TranslateColumnType(s, p.driver), nil
			}, v.getCommentToRight),
			Mods: stmtToMod{info},
		}
	}

	return queries, nil
}

func (Parser) parse(v *visitor, input string) ([]StmtInfo, error) {
	el := &errorListener{}

	// Get all hidden tokens (usually comments) and add edit rules to remove them
	v.BaseRules = []internal.EditRule{}
	hiddenLexer := sqliteparser.NewSQLiteLexer(antlr.NewInputStream(input))
	hiddenStream := antlr.NewCommonTokenStream(hiddenLexer, 1)
	hiddenStream.Fill()
	for _, token := range hiddenStream.GetAllTokens() {
		switch token.GetTokenType() {
		case sqliteparser.SQLiteParserSINGLE_LINE_COMMENT,
			sqliteparser.SQLiteParserMULTILINE_COMMENT:
			v.BaseRules = append(
				v.BaseRules,
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

	infos, ok := tree.Accept(v).([]StmtInfo)
	if v.Err != nil {
		return nil, fmt.Errorf("visitor: %w", v.Err)
	}

	if !ok {
		return nil, fmt.Errorf("visitor: expected stmtInfo, got %T", infos)
	}

	return infos, nil
}

type stmtToMod struct {
	info StmtInfo
}

func (s stmtToMod) IncludeInTemplate(i language.Importer) string {
	for _, im := range s.info.Imports {
		i.Import(im...)
	}
	return s.info.Mods.String()
}

type errorListener struct {
	*antlr.DefaultErrorListener

	err string
}

func (el *errorListener) SyntaxError(recognizer antlr.Recognizer, offendingSymbol any, line, column int, msg string, e antlr.RecognitionException) {
	el.err = msg
}

//nolint:gochecknoglobals
var defaultFunctions = Functions{
	"abs": {
		RequiredArgs: 1,
		Args:         []string{""},
		CalcReturnType: func(args ...string) string {
			if args[0] == "INTEGER" {
				return "INTEGER"
			}
			return "REAL"
		},
	},
	"changes": {
		ReturnType: "INTEGER",
	},
	"char": {
		RequiredArgs: 1,
		Variadic:     true,
		Args:         []string{"INTEGER"},
		ReturnType:   "TEXT",
	},
	"coalesce": {
		RequiredArgs:         1,
		Variadic:             true,
		Args:                 []string{""},
		ShouldArgsBeNullable: true,
		CalcReturnType: func(args ...string) string {
			for _, arg := range args {
				if arg != "" {
					return arg
				}
			}
			return ""
		},
		CalcNullable: antlrhelpers.AllNullable,
	},
	"concat": {
		RequiredArgs: 1,
		Variadic:     true,
		Args:         []string{"TEXT"},
		ReturnType:   "TEXT",
		CalcNullable: antlrhelpers.NeverNullable,
	},
	"concat_ws": {
		RequiredArgs: 2,
		Variadic:     true,
		Args:         []string{"TEXT", "TEXT"},
		ReturnType:   "TEXT",
		CalcNullable: func(args ...func() bool) func() bool {
			return args[0]
		},
	},
	"format": {
		RequiredArgs: 2,
		Variadic:     true,
		Args:         []string{"TEXT", ""},
		ReturnType:   "TEXT",
		CalcNullable: func(args ...func() bool) func() bool {
			return args[0]
		},
	},
	"glob": {
		RequiredArgs: 2,
		Args:         []string{"TEXT", "TEXT"},
		ReturnType:   "BOOLEAN",
	},
	"hex": {
		RequiredArgs: 1,
		Args:         []string{""},
		ReturnType:   "TEXT",
	},
	"ifnull": {
		RequiredArgs: 2,
		Args:         []string{""},
		CalcReturnType: func(args ...string) string {
			for _, arg := range args {
				if arg != "" {
					return arg
				}
			}
			return ""
		},
		CalcNullable: antlrhelpers.AllNullable,
	},
	"iif": {
		RequiredArgs: 3,
		Args:         []string{"BOOLEAN", "", ""},
		CalcReturnType: func(args ...string) string {
			return args[1]
		},
		CalcNullable: func(args ...func() bool) func() bool {
			return antlrhelpers.AnyNullable(args[1], args[2])
		},
	},
}
