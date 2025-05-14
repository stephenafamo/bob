package parser

import (
	"slices"

	"github.com/stephenafamo/bob/gen/bobgen-helpers/parser/antlrhelpers"
	"github.com/stephenafamo/bob/gen/drivers"
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
