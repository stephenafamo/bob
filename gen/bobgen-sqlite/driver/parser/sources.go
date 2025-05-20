package parser

import (
	"fmt"
	"strings"

	"github.com/stephenafamo/bob/gen/bobgen-helpers/parser"
	antlrhelpers "github.com/stephenafamo/bob/gen/bobgen-helpers/parser/antlrhelpers"
	"github.com/stephenafamo/bob/internal"
	sqliteparser "github.com/stephenafamo/sqlparser/sqlite"
)

func (v *visitor) addSourcesFromWithClause(ctx sqliteparser.IWith_clauseContext) {
	if ctx == nil {
		return
	}

	for _, cte := range ctx.AllCommon_table_expression() {
		columns, ok := cte.Select_stmt().Accept(v).([]ReturnColumn)
		if v.Err != nil {
			v.Err = fmt.Errorf("CTE select stmt: %w", v.Err)
			return
		}
		if !ok {
			v.Err = fmt.Errorf("could not get stmt info")
			return
		}

		source := QuerySource{
			Name:    getName(cte.Table_name()),
			Columns: columns,
			CTE:     true,
		}

		columnNames := cte.AllColumn_name()
		if len(columnNames) == 0 {
			v.Sources = append(v.Sources, source)
			return
		}

		if len(columnNames) != len(source.Columns) {
			v.Err = fmt.Errorf("column names do not match %d != %d", len(columnNames), len(source.Columns))
			return
		}

		for i, column := range columnNames {
			source.Columns[i].Name = getName(column)
		}

		v.Sources = append(v.Sources, source)
	}
}

func (v *visitor) addSourcesFromFrom_item(ctx sqliteparser.IFrom_itemContext) {
	tables := ctx.AllTable_or_subquery()

	sources := make([]QuerySource, len(tables))
	for i, table := range tables {
		sources[i] = v.getSourceFromTableOrSubQuery(table)
		if v.Err != nil {
			v.Err = fmt.Errorf("table or subquery %d: %w", i, v.Err)
			return
		}
	}

	for i, joinOp := range ctx.AllJoin_operator() {
		fullJoin := joinOp.FULL_() != nil
		leftJoin := fullJoin || joinOp.LEFT_() != nil
		rightJoin := fullJoin || joinOp.RIGHT_() != nil

		if leftJoin {
			right := sources[i+1]
			for i := range right.Columns {
				for j := range right.Columns[i].Type {
					right.Columns[i].Type[j].NullableF = antlrhelpers.Nullable
				}
			}
		}

		if rightJoin {
			left := sources[i+1]
			for i := range left.Columns {
				for j := range left.Columns[i].Type {
					left.Columns[i].Type[j].NullableF = antlrhelpers.Nullable
				}
			}
		}
	}

	v.Sources = append(v.Sources, sources...)
}

func (v *visitor) getSourceFromTableOrSubQuery(ctx sqliteparser.ITable_or_subqueryContext) QuerySource {
	switch {
	case ctx.Table_name() != nil:
		return v.getSourceFromTable(ctx)

	case ctx.Select_stmt() != nil:
		columns, ok := ctx.Select_stmt().Accept(v).([]ReturnColumn)
		if v.Err != nil {
			v.Err = fmt.Errorf("table select stmt: %w", v.Err)
			return QuerySource{}
		}
		if !ok {
			v.Err = fmt.Errorf("could not get stmt info")
			return QuerySource{}
		}

		return QuerySource{
			Name:    getName(ctx.Table_alias()),
			Columns: columns,
		}

	case ctx.Table_or_subquery() != nil:
		return v.getSourceFromTableOrSubQuery(ctx.Table_or_subquery())

	default:
		v.Err = fmt.Errorf("unknown table or subquery: %#v", antlrhelpers.Key(ctx))
		return QuerySource{}
	}
}

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

	if schema == "main" {
		schema = ""
	}

	for _, table := range v.DB {
		if table.Name != tableName {
			continue
		}

		switch {
		case table.Schema == schema: // schema matches
		case table.Schema == "main" && schema == "": // schema is shared
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

func (v *visitor) getSourceFromColumns(columns []sqliteparser.IResult_columnContext) QuerySource {
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
				Type:   v.Infos[antlrhelpers.Key(expr)].Type,
			})
		}
	}

	return returnSource
}
