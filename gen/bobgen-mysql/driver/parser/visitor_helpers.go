package parser

import (
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
