package parser

import (
	"slices"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/stephenafamo/bob/gen/bobgen-helpers/parser/antlrhelpers"
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
// Type matching
// ---------------------------------------------------------------------------

func (v *visitor) equateTypesAndNames(a, b Node) {
	v.MatchNodeNames(a, b)

	v.UpdateInfo(NodeInfo{
		Node:                 a,
		ExprDescription:      "equate A",
		ExprRef:              b,
		IgnoreRefNullability: true,
	})

	v.UpdateInfo(NodeInfo{
		Node:                 b,
		ExprDescription:      "equate B",
		ExprRef:              a,
		IgnoreRefNullability: true,
	})

	aChildren := v.getListNodes(a)
	bChildren := v.getListNodes(b)
	aSource := v.querySources[antlrhelpers.Key(a)]
	bSource := v.querySources[antlrhelpers.Key(b)]

	if len(aSource.Columns) == len(bChildren) {
		for i := range aSource.Columns {
			v.UpdateInfo(NodeInfo{
				Node:            bChildren[i],
				ExprDescription: "b from a source",
				Type:            aSource.Columns[i].Type,
			})
		}
	}

	if len(bSource.Columns) == len(aChildren) {
		for i := range bSource.Columns {
			v.UpdateInfo(NodeInfo{
				Node:            aChildren[i],
				ExprDescription: "a from b source",
				Type:            bSource.Columns[i].Type,
			})
		}
	}

	if len(aChildren) != len(bChildren) {
		return
	}

	for i := range aChildren {
		v.equateTypesAndNames(aChildren[i], bChildren[i])
	}
}

func (v *visitor) getListNodes(listable Node) []Node {
	switch listable := listable.(type) {
	case *mysqlparser.PredicateExpressionContext:
		return v.getListNodes(listable.Predicate())
	case *mysqlparser.ExpressionAtomPredicateContext:
		return v.getListNodes(listable.ExpressionAtom())
	case *mysqlparser.NestedExpressionAtomContext:
		return v.getExpressionListNodes(listable.ExpressionList())
	case mysqlparser.IExpressionListContext:
		return v.getExpressionListNodes(listable)
	case *mysqlparser.NestedRowExpressionAtomContext:
		return v.getRowNodes(listable)
	default:
		return nil
	}
}

func (v *visitor) getExpressionListNodes(listable mysqlparser.IExpressionListContext) []Node {
	expressions := listable.Expressions().AllExpression()
	nodes := make([]Node, len(expressions))

	for i, expression := range expressions {
		nodes[i] = expression
	}

	return nodes
}

func (v *visitor) getRowNodes(listable *mysqlparser.NestedRowExpressionAtomContext) []Node {
	expressions := listable.AllExpression()
	nodes := make([]Node, len(expressions))

	for i, expression := range expressions {
		nodes[i] = expression
	}

	return nodes
}
