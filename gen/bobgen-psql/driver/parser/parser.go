package parser

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"

	pg "github.com/pganalyze/pg_query_go/v6"
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/gen/bobgen-helpers/parser"
	"github.com/stephenafamo/bob/gen/drivers"
	"github.com/stephenafamo/bob/gen/language"
	"github.com/stephenafamo/bob/internal"
	pgparse "github.com/wasilibs/go-pgquery"
)

func New(db *sql.DB, t tables, sharedSchema string, translator *Translator) *Parser {
	return &Parser{
		conn:         db,
		db:           t,
		sharedSchema: sharedSchema,
		translator:   translator,
	}
}

type Parser struct {
	conn         *sql.DB
	db           tables
	sharedSchema string
	translator   *Translator
}

func (p *Parser) ParseFolders(ctx context.Context, paths ...string) ([]drivers.QueryFolder, error) {
	return parser.ParseFolders(ctx, p, paths...)
}

func (p *Parser) ParseQueries(ctx context.Context, s string) ([]drivers.Query, error) {
	stmts, err := pgparse.Parse(s)
	if err != nil {
		return nil, fmt.Errorf("parse formatted: %w", err)
	}

	if len(stmts.Stmts) == 0 {
		return nil, fmt.Errorf("no statements found")
	}

	var end int32
	queries := make([]drivers.Query, len(stmts.Stmts))

	for i, stmt := range stmts.Stmts {
		start := end
		switch {
		case stmt.GetStmtLen() > 0:
			end = stmt.GetStmtLocation() + stmt.GetStmtLen()
		default:
			end = int32(len(s))
		}
		stmtStr := strings.Trim(s[start:end], " \t\n;")

		queries[i], err = p.ParseQuery(ctx, stmtStr)
		if err != nil {
			return nil, fmt.Errorf("parse query %d: %w", i, err)
		}
	}

	return queries, nil
}

func (p *Parser) ParseQuery(ctx context.Context, input string) (drivers.Query, error) {
	argTypes, resTypes, err := p.getArgsAndCols(ctx, input)
	if err != nil {
		return drivers.Query{}, fmt.Errorf("get args and cols: %w", err)
	}

	scanResult, err := pgparse.Scan(input)
	if err != nil {
		return drivers.Query{}, fmt.Errorf("scan: %w", err)
	}

	parseResult, err := pgparse.Parse(input)
	if err != nil {
		return drivers.Query{}, fmt.Errorf("parse single: %w", err)
	}

	if len(parseResult.Stmts) != 1 {
		return drivers.Query{}, fmt.Errorf("expected 1 statement, got %d", len(parseResult.Stmts))
	}

	w := walker{
		db:           p.db,
		sharedSchema: p.sharedSchema,
		input:        input,
		tokens:       scanResult.GetTokens(),
		mods:         &strings.Builder{},
		nullability:  make(map[position]nullable),
		names:        make(map[position]string),
		groups:       make(map[argPos]struct{}),
		multiple:     make(map[[2]int]struct{}),
		atom:         &atomic.Int64{},
	}

	stmt := parseResult.Stmts[0]
	info := w.walk(stmt.Stmt)
	switch node := stmt.Stmt.Node.(type) {
	case *pg.Node_SelectStmt:
		if len(node.SelectStmt.ValuesLists) > 0 {
			return drivers.Query{}, fmt.Errorf("VALUES statement is not supported")
		}

		info = info.children["SelectStmt"]
		w.modSelectStatement(node, info)

	case *pg.Node_InsertStmt:
		info = info.children["InsertStmt"]
		w.modInsertStatement(node, info)

	case *pg.Node_UpdateStmt:
		info = info.children["UpdateStmt"]
		w.modUpdateStatement(node, info)

	case *pg.Node_DeleteStmt:
		info = info.children["DeleteStmt"]
		w.modDeleteStatement(node, info)
	}

	source := w.getSource(stmt.Stmt, info)

	if len(w.errors) > 0 {
		return drivers.Query{}, errors.Join(w.errors...)
	}

	if len(source.columns) != len(resTypes) {
		return drivers.Query{}, fmt.Errorf("expected %d columns, got %d", len(resTypes), len(source.columns))
	}

	if len(w.args) != len(argTypes) {
		return drivers.Query{}, fmt.Errorf("expected %d args, got %d", len(resTypes), len(source.columns))
	}

	formatted, err := w.formattedQuery()
	if err != nil {
		return drivers.Query{}, fmt.Errorf("format: %w", err)
	}

	comment, err := w.getQueryComment(info.start)
	if err != nil {
		return drivers.Query{}, fmt.Errorf("get comment: %w", err)
	}
	name, configStr, _ := strings.Cut(comment, " ")

	// fmt.Printf("Names: %v\n", litter.Sdump(w.names))
	// fmt.Printf("Nullability: %v\n", litter.Sdump(w.nullability))
	// fmt.Printf("Args: %v\n", litter.Sdump(w.args))
	// fmt.Printf("Groups: %v\n", litter.Sdump(w.groups))
	// fmt.Printf("Multiples: %v\n", litter.Sdump(w.multiple))
	// fmt.Printf("FINAL %T %p: (%v) : %s\n", stmt.Stmt, stmt.Stmt, info, stmt.Stmt)
	// fmt.Printf("FINAL %T %p: (%v) : %s\n", stmt.Stmt, stmt.Stmt, pp.Sprint(info), stmt.Stmt)
	// fmt.Printf("Source: %v\n", pp.Sprint(source))
	// pp.Printf("Arg types: %v\n", argTypes)
	// pp.Printf("Res types: %v\n", resTypes)

	query := drivers.Query{
		Name:    name,
		SQL:     formatted,
		Type:    getQueryType(stmt.Stmt),
		Config:  parser.ParseQueryConfig(configStr),
		Columns: make([]drivers.QueryCol, len(source.columns)),
		Args:    w.getArgs(argTypes),
		Mods:    w,
	}

	for i, col := range source.columns {
		query.Columns[i] = drivers.QueryCol{
			Name:     col.name,
			DBName:   col.name,
			Nullable: internal.Pointer(col.nullable),
			TypeName: resTypes[i],
		}.Merge(parser.ParseQueryColumnConfig(
			w.getConfigComment(col.pos[1]),
		))
	}

	return query, nil
}

func getQueryType(stmt *pg.Node) bob.QueryType {
	switch stmt.Node.(type) {
	case *pg.Node_SelectStmt:
		return bob.QueryTypeSelect
	case *pg.Node_InsertStmt:
		return bob.QueryTypeInsert
	case *pg.Node_UpdateStmt:
		return bob.QueryTypeUpdate
	case *pg.Node_DeleteStmt:
		return bob.QueryTypeDelete
	default:
		return bob.QueryTypeUnknown
	}
}

func (w walker) IncludeInTemplate(i language.Importer) string {
	for _, im := range w.imports {
		i.Import(im...)
	}
	return w.mods.String()
}

func (w *walker) formattedQuery() (string, error) {
	var rules []internal.EditRule
	for _, token := range w.tokens {
		// fmt.Printf("Token: %s\n", token.String())
		switch token.GetToken() {
		case pg.Token_SQL_COMMENT:
			rules = append(rules, internal.Delete(int(token.GetStart()), int(token.GetEnd())))
		case pg.Token_C_COMMENT:
			rules = append(rules, internal.Delete(int(token.GetStart()), int(token.GetEnd()-1)))
		}
	}

	return internal.EditString(w.input, append(rules, w.editRules...)...)
}
