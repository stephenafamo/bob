package parser

import (
	"fmt"
	"slices"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/stephenafamo/bob/gen/bobgen-helpers/parser"
	antlrhelpers "github.com/stephenafamo/bob/gen/bobgen-helpers/parser/antlrhelpers"
	"github.com/stephenafamo/bob/internal"
	sqliteparser "github.com/stephenafamo/sqlparser/sqlite"
)

// ---------------------------------------------------------------------------
// Comment getter
// ---------------------------------------------------------------------------

func (v *visitor) getCommentToLeft(ctx Node) string {
	stream, isCommon := ctx.GetParser().GetTokenStream().(*antlr.CommonTokenStream)
	if !isCommon {
		return ""
	}

	tokenIndex := ctx.GetStart().GetTokenIndex()
	hiddenTokens := stream.GetHiddenTokensToLeft(tokenIndex, 1)
	for _, token := range slices.Backward(hiddenTokens) {
		if token.GetTokenType() == sqliteparser.SQLiteParserSINGLE_LINE_COMMENT {
			return strings.TrimSpace(token.GetText()[2:])
		}
	}

	return ""
}

func (v *visitor) getCommentToRight(ctx Node) string {
	stream, isCommon := ctx.GetParser().GetTokenStream().(*antlr.CommonTokenStream)
	if !isCommon {
		return ""
	}

	tokenIndex := ctx.GetStop().GetTokenIndex()
	hiddenTokens := stream.GetHiddenTokensToRight(tokenIndex, 1)
	for _, token := range hiddenTokens {
		if token.GetTokenType() == sqliteparser.SQLiteParserMULTILINE_COMMENT {
			txt := token.GetText()
			return strings.TrimSpace(txt[2 : len(txt)-2])
		}
	}

	return ""
}

// ---------------------------------------------------------------------------
// Edit Rule helpers
// ---------------------------------------------------------------------------
func (v *visitor) quoteIdentifiable(ctx interface {
	Identifier() sqliteparser.IIdentifierContext
},
) {
	if ctx == nil {
		return
	}

	v.quoteIdentifier(ctx.Identifier())
}

func (v *visitor) quoteIdentifier(ctx sqliteparser.IIdentifierContext) {
	if ctx == nil {
		return
	}

	switch ctx.GetParent().(type) {
	case sqliteparser.ISimple_funcContext,
		sqliteparser.IAggregate_funcContext,
		sqliteparser.IWindow_funcContext,
		sqliteparser.ITable_function_nameContext:
		return
	}

	idContext := ctx

	for idContext.OPEN_PAR() != nil {
		idContext = idContext.Identifier()
	}

	txt := ctx.GetText()
	if strings.ContainsAny(string(txt[0]), "\"`[") {
		txt = txt[1 : len(txt)-1]
	}

	v.StmtRules = append(v.StmtRules, internal.Replace(ctx.GetStart().GetStart(), ctx.GetStop().GetStop(), fmt.Sprintf("%q", txt)))
}

// ---------------------------------------------------------------------------
// Source helpers
// ---------------------------------------------------------------------------
func (v *visitor) getSourceFromTable(ctx interface {
	Schema_name() sqliteparser.ISchema_nameContext
	Table_name() sqliteparser.ITable_nameContext
	Table_alias() sqliteparser.ITable_aliasContext
},
) QuerySource {
	schema := getName(ctx.Schema_name())
	v.quoteIdentifiable(ctx.Schema_name())

	tableName := getName(ctx.Table_name())
	v.quoteIdentifiable(ctx.Table_name())

	alias := getName(ctx.Table_alias())
	v.quoteIdentifiable(ctx.Table_alias())

	hasAlias := alias != ""
	if alias == "" {
		alias = tableName
	}

	// First check the sources to see if the table exists
	// do this ONLY if no schema is provided
	if schema == "" {
		for _, source := range v.Sources {
			if source.Name == tableName {
				return QuerySource{
					Name:    alias,
					Columns: source.Columns,
				}
			}
		}
	}

	for _, table := range v.DB {
		if table.Name != tableName {
			continue
		}

		switch {
		case table.Schema == schema: // schema matches
		case table.Schema == "" && schema == "main": // schema is shared
		default:
			continue
		}

		source := QuerySource{
			Name:    alias,
			Columns: make([]ReturnColumn, len(table.Columns)),
		}
		if !hasAlias {
			source.Schema = schema
		}
		for i, col := range table.Columns {
			source.Columns[i] = ReturnColumn{
				Name: col.Name,
				Type: NodeTypes{getColumnType(v.DB, table.Schema, table.Name, col.Name)},
			}
		}
		return source
	}

	v.Err = fmt.Errorf("table not found: %s", tableName)
	return QuerySource{}
}

func (v *visitor) sourceFromColumns(columns []sqliteparser.IResult_columnContext) QuerySource {
	// Get the return columns
	var returnSource QuerySource

	for _, resultColumn := range columns {
		switch {
		case resultColumn.STAR() != nil: // Has a STAR: * OR table_name.*
			table := getName(resultColumn.Table_name())
			hasTable := table != "" // the result column is table_name.*

			start := resultColumn.GetStart().GetStart()
			stop := resultColumn.GetStop().GetStop()
			v.StmtRules = append(v.StmtRules, internal.Delete(start, stop))

			buf := &strings.Builder{}
			var i int
			for _, source := range v.Sources {
				if source.CTE {
					continue
				}
				if hasTable && source.Name != table {
					continue
				}

				returnSource.Columns = append(returnSource.Columns, source.Columns...)

				if i > 0 {
					buf.WriteString(", ")
				}
				antlrhelpers.ExpandQuotedSource(buf, source)
				i++
			}
			v.StmtRules = append(v.StmtRules, internal.Insert(start, buf.String()))

		case resultColumn.Expr() != nil: // expr (AS_? alias)?
			expr := resultColumn.Expr()
			alias := getName(resultColumn.Alias())
			if alias == "" {
				if expr, ok := expr.(*sqliteparser.Expr_qualified_column_nameContext); ok {
					alias = getName(expr.Column_name())
				}
			}

			returnSource.Columns = append(returnSource.Columns, ReturnColumn{
				Name:   alias,
				Config: parser.ParseQueryColumnConfig(v.getCommentToRight(expr)),
				Type:   v.Infos[antlrhelpers.Key(resultColumn.Expr())].Type,
			})
		}
	}

	return returnSource
}
