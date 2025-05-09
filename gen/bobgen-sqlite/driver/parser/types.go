package parser

import (
	"slices"
	"strings"

	antlrhelpers "github.com/stephenafamo/bob/gen/bobgen-helpers/parser/antlrhelpers"
	"github.com/stephenafamo/bob/gen/drivers"
	sqliteparser "github.com/stephenafamo/sqlparser/sqlite"
)

type (
	tables     = drivers.Tables[any, IndexExtra]
	IndexExtra = struct {
		Partial bool `json:"partial"`
	}

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

type identifiable interface {
	Identifier() sqliteparser.IIdentifierContext
}

func getName(i identifiable) string {
	if i == nil {
		return ""
	}
	ctx := i.Identifier()
	for ctx.OPEN_PAR() != nil {
		ctx = ctx.Identifier()
	}

	txt := ctx.GetText()
	if strings.ContainsAny(string(txt[0]), "\"`[") {
		return txt[1 : len(txt)-1]
	}

	return txt
}

func makeRef(sources []QuerySource, ctx *sqliteparser.Expr_qualified_column_nameContext) NodeTypes {
	schema := getName(ctx.Schema_name())
	table := getName(ctx.Table_name())
	column := getName(ctx.Column_name())
	if schema == "main" {
		schema = ""
	}

	for _, source := range slices.Backward(sources) {
		if table != "" && (schema != source.Schema || table != source.Name) {
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

func getColumnType(db tables, schema, table, column string) NodeType {
	if schema == "main" {
		schema = ""
	}

	colType := antlrhelpers.GetColumnType(db, schema, table, column)
	colType.DBType = getTypeFromTypeName(colType.DBType)

	return colType
}

func knownType(t string, nullable func() bool) NodeType {
	return NodeType{
		DBType:    getTypeFromTypeName(t),
		NullableF: nullable,
	}
}

// https://www.sqlite.org/datatype3.html
//
//nolint:misspell
func getTypeFromTypeName(t string) string {
	if t == "" {
		return ""
	}

	if strings.Contains(t, "INT") {
		return "INTEGER"
	}

	if strings.Contains(t, "CHAR") || strings.Contains(t, "CLOB") || strings.Contains(t, "TEXT") {
		return "TEXT"
	}

	if strings.Contains(t, "BLOB") {
		return "BLOB"
	}

	if strings.Contains(t, "REAL") || strings.Contains(t, "FLOA") || strings.Contains(t, "DOUB") {
		return "REAL"
	}

	return "NUMERIC"
}

func nullable() bool {
	return true
}

func notNullable() bool {
	return false
}

func anyNullable(fs ...func() bool) func() bool {
	return func() bool {
		for _, f := range fs {
			if f() {
				return true
			}
		}

		return false
	}
}

func allNullable(fs ...func() bool) func() bool {
	return func() bool {
		for _, f := range fs {
			if !f() {
				return false
			}
		}

		return true
	}
}

func neverNullable(...func() bool) func() bool {
	return notNullable
}
