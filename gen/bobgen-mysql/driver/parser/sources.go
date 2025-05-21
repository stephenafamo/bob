package parser

import (
	"fmt"
	"strings"

	"github.com/stephenafamo/bob/gen/bobgen-helpers/parser"
	"github.com/stephenafamo/bob/gen/bobgen-helpers/parser/antlrhelpers"
	"github.com/stephenafamo/bob/internal"
	mysqlparser "github.com/stephenafamo/sqlparser/mysql"
)

func (v *visitor) addSourcesFromWithClause(ctx mysqlparser.IWithClauseContext) {
	if ctx == nil {
		return
	}

	for _, cte := range ctx.AllCommonTableExpression() {
		stmt := cte.SubSelect().SelectStatement()
		columns, ok := stmt.Accept(v).([]ReturnColumn)
		if v.Err != nil {
			v.Err = fmt.Errorf("CTE select stmt: %w", v.Err)
			return
		}
		if !ok {
			v.Err = fmt.Errorf("could not get stmt info from %T", stmt)
			return
		}

		source := QuerySource{
			Name:    v.GetName(cte.CteName()),
			Columns: columns,
			CTE:     true,
		}

		columnNames := cte.AllCteColumnName()
		if len(columnNames) == 0 {
			v.Sources = append(v.Sources, source)
			return
		}

		if len(columnNames) != len(source.Columns) {
			v.Err = fmt.Errorf("column names do not match %d != %d", len(columnNames), len(source.Columns))
			return
		}

		for i, column := range columnNames {
			source.Columns[i].Name = v.GetName(column)
		}

		v.Sources = append(v.Sources, source)
	}
}

func (v *visitor) addSourcesFromTableSources(ctx mysqlparser.ITableSourcesContext) {
	v.Sources = append(v.Sources, v.getSourcesFromTableSources(ctx)...)
}

func (v *visitor) getSourcesFromTableSources(ctx mysqlparser.ITableSourcesContext) []QuerySource {
	if ctx == nil {
		return nil
	}

	tables := ctx.AllTableSource()
	if len(tables) == 0 {
		return nil
	}

	if len(tables) > 1 {
		v.Err = fmt.Errorf("using COMMA to join tables is not supported, use a JOIN instead")
		return nil
	}

	return v.getSourcesFromTableSource(tables[0])
}

func (v *visitor) getSourcesFromTableSource(ctx mysqlparser.ITableSourceContext) []QuerySource {
	if ctx == nil {
		return nil
	}

	if item := ctx.TableSourceItem(); item != nil {
		return v.getSourcesFromTableSourceItem(item)
	}

	var leftSources, rightSources []QuerySource
	isLeftJoin, isRightJoin := false, false

	leftSources = v.getSourcesFromTableSource(ctx.TableSource())

	switch join := ctx.JoinPart().(type) {
	case *mysqlparser.InnerJoinContext:
		rightSources = v.getSourcesFromTableSourceItem(join.TableSourceItem())
	case *mysqlparser.StraightJoinContext:
		rightSources = v.getSourcesFromTableSourceItem(join.TableSourceItem())
	case *mysqlparser.OuterJoinContext:
		rightSources = v.getSourcesFromTableSource(join.TableSource())
		isLeftJoin = join.LEFT() != nil
		isRightJoin = join.RIGHT() != nil
	case *mysqlparser.NaturalJoinContext:
		rightSources = v.getSourcesFromTableSourceItem(join.TableSourceItem())
		isLeftJoin = join.LEFT() != nil
		isRightJoin = join.RIGHT() != nil
	}

	if isLeftJoin {
		for i := range rightSources {
			for j := range rightSources[i].Columns {
				for k := range rightSources[i].Columns[j].Type {
					rightSources[i].Columns[j].Type[k].NullableF = antlrhelpers.Nullable
				}
			}
		}
	}

	if isRightJoin {
		for i := range leftSources {
			for j := range leftSources[i].Columns {
				for k := range leftSources[i].Columns[j].Type {
					leftSources[i].Columns[j].Type[k].NullableF = antlrhelpers.Nullable
				}
			}
		}
	}

	return append(leftSources, rightSources...)
}

func (v *visitor) getSourcesFromTableSourceItem(ctx mysqlparser.ITableSourceItemContext) []QuerySource {
	if ctx == nil {
		return nil
	}

	switch ctx := ctx.(type) {
	case *mysqlparser.AtomTableItemContext:
		return []QuerySource{v.getSourceFromTable(ctx)}

	case *mysqlparser.TableSourcesItemContext:
		return v.getSourcesFromTableSources(ctx.TableSources())

	case *mysqlparser.SubqueryTableItemContext:
		columns, ok := ctx.GetSubquery().Accept(v).([]ReturnColumn)
		if v.Err != nil {
			v.Err = fmt.Errorf("subquery: %w", v.Err)
			return nil
		}

		if !ok {
			v.Err = fmt.Errorf("could not get stmt info from %T", ctx.GetSubquery())
			return nil
		}

		source := QuerySource{
			Name:    getUIDName(ctx.GetTableAlias()),
			Columns: columns,
		}

		for i, colAlias := range ctx.GetColAlias() {
			if i < len(source.Columns) {
				source.Columns[i].Name = getUIDName(colAlias)
			}
		}

		return []QuerySource{source}

	case *mysqlparser.JsonTableItemContext:
		return nil

	default:
		return nil
	}
}

func (v *visitor) getSourceFromTable(ctx interface {
	TableName() mysqlparser.ITableNameContext
	GetTableAlias() mysqlparser.IUidContext
},
) QuerySource {
	source := v.getSourceFromTableName(ctx.TableName())
	if v.Err != nil {
		v.Err = fmt.Errorf("source from table name: %w", v.Err)
	}

	tableAlias := getUIDName(ctx.GetTableAlias())
	if tableAlias == "" {
		return source
	}

	source.Name = tableAlias
	return source
}

func (v *visitor) getSourceFromTableName(ctx mysqlparser.ITableNameContext) QuerySource {
	tableName := getFullIDName(ctx.FullId())
	// First check the sources to see if the table exists
	// do this ONLY if no schema is provided
	for _, source := range v.Sources {
		if source.Name == tableName {
			return source
		}
	}

	for _, table := range v.DB {
		if table.Name != tableName {
			continue
		}

		source := QuerySource{
			Name:    tableName,
			Columns: make([]ReturnColumn, len(table.Columns)),
		}
		for i, col := range table.Columns {
			source.Columns[i] = ReturnColumn{
				Name: col.Name,
				Type: NodeTypes{getColumnType(v.DB, table.Name, col.Name)},
			}
		}
		return source
	}

	v.Err = fmt.Errorf("table not found: %s", tableName)
	return QuerySource{}
}

func (v *visitor) getSourceFromSelectElements(ctx mysqlparser.ISelectElementsContext) QuerySource {
	// Get the return columns
	var returnSource QuerySource

	for _, resultColumn := range ctx.AllSelectElement() {
		switch resultColumn := resultColumn.(type) {
		case *mysqlparser.SelectStarElementContext: // *
			start := resultColumn.GetStart().GetStart()
			stop := resultColumn.GetStop().GetStop()
			v.StmtRules = append(v.StmtRules, internal.Delete(start, stop))

			buf := &strings.Builder{}
			var i int
			for _, source := range v.Sources {
				if source.CTE {
					continue
				}

				returnSource.Columns = append(returnSource.Columns, source.Columns...)

				if i > 0 {
					buf.WriteString(", ")
				}
				ExpandQuotedSource(buf, source)
				i++
			}
			v.StmtRules = append(v.StmtRules, internal.Insert(start, buf.String()))

		case *mysqlparser.SelectTableElementContext: // table.*
			table := getFullIDName(resultColumn.FullId())
			start := resultColumn.GetStart().GetStart()
			stop := resultColumn.GetStop().GetStop()
			v.StmtRules = append(v.StmtRules, internal.Delete(start, stop))

			buf := &strings.Builder{}
			var i int
			for _, source := range v.Sources {
				if source.CTE {
					continue
				}

				if source.Name != table {
					continue
				}

				returnSource.Columns = append(returnSource.Columns, source.Columns...)

				if i > 0 {
					buf.WriteString(", ")
				}
				ExpandQuotedSource(buf, source)
				i++
			}
			v.StmtRules = append(v.StmtRules, internal.Insert(start, buf.String()))

		case *mysqlparser.SelectColumnElementContext: // table?.column AS alias
			col := resultColumn.FullColumnName()
			alias := getUIDName(resultColumn.Uid())
			if alias == "" {
				alias = getFullColumnName(col)
			}

			returnSource.Columns = append(returnSource.Columns, ReturnColumn{
				Name:   alias,
				Config: parser.ParseQueryColumnConfig(v.getCommentToRight(col)),
				Type:   v.Infos[antlrhelpers.Key(col)].Type,
			})

		case *mysqlparser.SelectFunctionElementContext: // func() AS alias
			col := resultColumn.FunctionCall()
			alias := getUIDName(resultColumn.Uid())

			returnSource.Columns = append(returnSource.Columns, ReturnColumn{
				Name:   alias, // may be empty
				Config: parser.ParseQueryColumnConfig(v.getCommentToRight(col)),
				Type:   v.Infos[antlrhelpers.Key(col)].Type,
			})

		case *mysqlparser.SelectExpressionElementContext: // expr AS alias
			col := resultColumn.Expression()
			alias := getUIDName(resultColumn.Uid())

			returnSource.Columns = append(returnSource.Columns, ReturnColumn{
				Name:   alias, // may be empty
				Config: parser.ParseQueryColumnConfig(v.getCommentToRight(col)),
				Type:   v.Infos[antlrhelpers.Key(col)].Type,
			})
		}
	}

	return returnSource
}

func ExpandQuotedSource(buf *strings.Builder, source QuerySource) {
	for i, col := range source.Columns {
		if i > 0 {
			buf.WriteString(", ")
		}
		if source.Schema != "" {
			fmt.Fprintf(buf, "`%s`.`%s`.`%s`", source.Schema, source.Name, col.Name)
		} else {
			fmt.Fprintf(buf, "`%s`.`%s`", source.Name, col.Name)
		}
	}
}
