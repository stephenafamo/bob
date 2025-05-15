package parser

import (
	"fmt"
	"slices"
	"strings"

	"github.com/antlr4-go/antlr/v4"
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
