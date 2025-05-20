package parser

import (
	"slices"

	"github.com/stephenafamo/bob/gen/bobgen-helpers/parser/antlrhelpers"
	"github.com/stephenafamo/bob/gen/drivers"
	mysqlparser "github.com/stephenafamo/sqlparser/mysql"
)

type (
	tables       = drivers.Tables[any, any]
	Node         = antlrhelpers.Node
	NodeKey      = antlrhelpers.NodeKey
	NodeType     = antlrhelpers.NodeType
	NodeTypes    = antlrhelpers.NodeTypes
	NodeInfo     = antlrhelpers.NodeInfo
	ReturnColumn = antlrhelpers.ReturnColumn
	QuerySource  = antlrhelpers.QuerySource
	StmtInfo     = antlrhelpers.StmtInfo
	Function     = antlrhelpers.Function
	Functions    = antlrhelpers.Functions
)

func knownTypeNull(t string) NodeType {
	return antlrhelpers.KnownTypeNull(t)
}

func knownTypeNotNull(t string) NodeType {
	return antlrhelpers.KnownTypeNotNull(t)
}

func makeRef(sources []QuerySource, table, column string) NodeTypes {
	for _, source := range slices.Backward(sources) {
		if table != "" && table != source.Name {
			continue
		}

		for _, col := range source.Columns {
			if col.Name != column {
				continue
			}

			return col.Type
		}
	}

	return nil
}

func getColumnType(db tables, table, column string) NodeType {
	return antlrhelpers.GetColumnType(db, "", table, column)
}

func getFullColumnName(ctx mysqlparser.IFullColumnNameContext) string {
	if ctx == nil {
		return ""
	}

	allDotted := ctx.AllDottedId()
	switch len(allDotted) {
	case 0:
		return getUIDName(ctx.Uid())
	case 1:
		return getDottedIDName(allDotted[0])
	case 2:
		return getDottedIDName(allDotted[1])
	}

	return ""
}

func getFullIDName(ctx mysqlparser.IFullIdContext) string {
	if ctx == nil {
		return ""
	}

	if dotted := ctx.DottedId(); dotted != nil {
		return getDottedIDName(dotted)
	}

	return getUIDName(ctx.Uid())
}

func getDottedIDName(ctx mysqlparser.IDottedIdContext) string {
	if ctx == nil {
		return ""
	}

	if ctx.Uid() != nil {
		return getUIDName(ctx.Uid())
	}

	return ctx.GetText()[1:]
}

func getUIDName(ctx mysqlparser.IUidContext) string {
	if ctx == nil {
		return ""
	}

	if ctx.SimpleId() != nil {
		return getSimpleIDName(ctx.SimpleId())
	}

	return ctx.GetText()[1 : len(ctx.GetText())-1] // remove the quotes
}

func getSimpleIDName(ctx mysqlparser.ISimpleIdContext) string {
	if ctx == nil {
		return ""
	}

	return ctx.GetText()
}
