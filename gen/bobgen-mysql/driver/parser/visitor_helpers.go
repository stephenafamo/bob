package parser

import (
	"fmt"
	"slices"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	mysqlparser "github.com/stephenafamo/sqlparser/mysql"
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
		if token.GetTokenType() == mysqlparser.MySqlParserLINE_COMMENT {
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
		if token.GetTokenType() == mysqlparser.MySqlParserCOMMENT_INPUT {
			txt := token.GetText()
			return strings.TrimSpace(txt[2 : len(txt)-2])
		}
	}

	return ""
}

// ---------------------------------------------------------------------------
// Source helpers
// ---------------------------------------------------------------------------
func (v *visitor) getSourceFromTable(tableName string, tableAlias string, colaliases ...string) QuerySource {
	if tableAlias == "" {
		tableAlias = tableName
	}

	// First check the sources to see if the table exists
	// do this ONLY if no schema is provided
	for _, source := range v.Sources {
		if source.Name == tableName {
			return QuerySource{
				Name:    tableAlias,
				Columns: source.Columns,
			}
		}
	}

	for _, table := range v.DB {
		if table.Name != tableName {
			continue
		}

		source := QuerySource{
			Name:    tableAlias,
			Columns: make([]ReturnColumn, len(table.Columns)),
		}
		for i, col := range table.Columns {
			source.Columns[i] = ReturnColumn{
				Name: col.Name,
				Type: NodeTypes{getColumnType(v.DB, table.Name, col.Name)},
			}
			if len(colaliases) > i {
				source.Columns[i].Name = colaliases[i]
			}
		}
		return source
	}

	v.Err = fmt.Errorf("table not found: %s", tableName)
	return QuerySource{}
}
