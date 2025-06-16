package parser

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/stephenafamo/bob/gen/bobgen-helpers/parser"
	"github.com/stephenafamo/bob/gen/drivers"
	"github.com/stephenafamo/bob/gen/language"
	"github.com/stephenafamo/bob/internal"
	mysqlparser "github.com/stephenafamo/sqlparser/mysql"
)

func New(t tables) Parser {
	return Parser{db: t}
}

type Parser struct {
	db tables
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
			typeName, typeLimits := TranslateColumnType(col.Type.ConfirmedDBType())
			cols[i] = drivers.QueryCol{
				Name:       col.Name,
				DBName:     col.Name,
				Nullable:   internal.Pointer(col.Type.Nullable()),
				TypeName:   typeName,
				TypeLimits: typeLimits,
			}.Merge(col.Config)
		}

		name, configStr, _ := strings.Cut(info.Comment, " ")
		queries[i] = drivers.Query{
			Name:    name,
			SQL:     formatted,
			Type:    info.QueryType,
			Config:  parser.ParseQueryConfig(configStr),
			Columns: cols,
			Args:    v.GetArgs(stmtStart, stmtStop, TranslateColumnType, v.getCommentToRight),
			Mods:    stmtToMod{info},
		}
	}

	return queries, nil
}

func (Parser) parse(v *visitor, input string) ([]StmtInfo, error) {
	el := &errorListener{}

	// Get all hidden tokens (usually comments) and add edit rules to remove them
	v.BaseRules = []internal.EditRule{}
	hiddenLexer := mysqlparser.NewMySqlLexer(antlr.NewInputStream(input))
	hiddenStream := antlr.NewCommonTokenStream(hiddenLexer, 1)
	hiddenStream.Fill()
	for _, token := range hiddenStream.GetAllTokens() {
		switch token.GetTokenType() {
		case mysqlparser.MySqlParserLINE_COMMENT,
			mysqlparser.MySqlParserCOMMENT_INPUT:
			v.BaseRules = append(
				v.BaseRules,
				internal.Delete(token.GetStart(), token.GetStop()),
			)
		}
	}

	// Get the regular tokens (usually the SQL statement)
	lexer := mysqlparser.NewMySqlLexer(antlr.NewInputStream(input))
	stream := antlr.NewCommonTokenStream(lexer, 0)
	sqlParser := mysqlparser.NewMySqlParser(stream)
	sqlParser.AddErrorListener(el)

	tree := sqlParser.Root()
	if el.err != "" {
		return nil, errors.New(el.err)
	}

	infos, ok := tree.Accept(v).([]StmtInfo)
	if v.Err != nil {
		return nil, fmt.Errorf("visitor: %w", v.Err)
	}

	// symNames := mysqlparser.MySqlLexerLexerStaticData.SymbolicNames
	// for _, tok := range stream.GetAllTokens() {
	// 	tType := tok.GetTokenType()
	// 	tTypeName := ""
	// 	if tType > 0 && tType < len(symNames) {
	// 		tTypeName = symNames[tType]
	// 	}
	//
	// 	fmt.Printf("%-20s %s\n",
	// 		tTypeName,
	// 		tok.GetText(),
	// 	)
	// }

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
var defaultFunctions = Functions{}
