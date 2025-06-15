package parser

import (
	"fmt"
	"reflect"
	"slices"
	"strconv"
	"strings"

	pg "github.com/pganalyze/pg_query_go/v6"
	"github.com/stephenafamo/bob/internal"
)

func (w *walker) getSource(node *pg.Node, info nodeInfo, sources ...queryResult) queryResult {
	cloned := slices.Clone(sources)

	switch stmt := node.Node.(type) {
	case *pg.Node_SelectStmt:
		return w.getSelectSource(stmt.SelectStmt, info, cloned...)

	case *pg.Node_InsertStmt:
		return w.getInsertSource(stmt.InsertStmt, info)

	case *pg.Node_UpdateStmt:
		return w.getUpdateSource(stmt.UpdateStmt, info, cloned...)

	case *pg.Node_DeleteStmt:
		return w.getDeleteSource(stmt.DeleteStmt, info, cloned...)

	case *pg.Node_RangeVar:
		if rangeInfo, ok := info.children["RangeVar"]; ok {
			info = rangeInfo
		}
		return w.getTableSource(stmt.RangeVar, info)

	case *pg.Node_RangeSubselect:
		if subSelInfo, ok := info.children["RangeSubselect"]; ok {
			info = subSelInfo
		}
		sub := stmt.RangeSubselect
		source := w.getSource(sub.Subquery, info.children["Subquery"], cloned...)
		if sub.Alias == nil {
			return source
		}
		source.name = sub.Alias.Aliasname
		if len(source.columns) != len(sub.Alias.Colnames) {
			return source
		}

		colInfos := info.children["Alias"].children["Colnames"]
		for i := range sub.Alias.Colnames {
			aliasName := w.names[colInfos.children[strconv.Itoa(i)].position()]
			if aliasName != "" {
				source.columns[i].name = aliasName
			}
		}
		return source

	default:
		return queryResult{}
	}
}

func (w *walker) getTableSource(sub *pg.RangeVar, info nodeInfo) queryResult {
	schema := w.names[info.children["Schemaname"].position()]
	name := w.names[info.children["Relname"].position()]

	source := queryResult{}

	for _, table := range w.db {
		if table.Name != name {
			continue
		}

		switch {
		case table.Schema == schema: // schema matches
		case table.Schema == "" && schema == w.sharedSchema: // schema is shared
		default:
			continue
		}

		source = queryResult{
			schema:  table.Schema,
			name:    table.Name,
			columns: make([]col, len(table.Columns)),
		}
		for j, column := range table.Columns {
			source.columns[j] = col{name: column.Name, nullable: column.Nullable}
		}

		break
	}

	if source.name == "" || sub.Alias == nil {
		return source
	}

	source.schema = "" // empty schema for aliased tables
	source.name = sub.Alias.Aliasname
	if len(source.columns) != len(sub.Alias.Colnames) {
		return source
	}

	colInfos := info.children["Alias"].children["Colnames"]
	for i := range sub.Alias.Colnames {
		aliasName := w.names[colInfos.children[strconv.Itoa(i)].position()]
		if aliasName != "" {
			source.columns[i].name = aliasName
		}
	}

	return source
}

func (w *walker) getSelectSource(stmt *pg.SelectStmt, info nodeInfo, sources ...queryResult) queryResult {
	if len(stmt.ValuesLists) > 0 {
		list, isList := stmt.ValuesLists[0].Node.(*pg.Node_List)
		if isList {
			return w.getSourceFromList(
				list.List,
				info.children["ValuesLists"].children["0"].children["List"],
				sources...,
			)
		}
	}

	sources = w.addSourcesOfWithClause(stmt.WithClause, info.children["WithClause"], sources...)

	main := stmt
	mainInfo := info
	for main.Larg != nil {
		main = main.Larg
		mainInfo = mainInfo.children["Larg"]
	}

	if len(main.FromClause) == 0 {
		return w.getSourceFromTargets(main.TargetList, mainInfo.children["TargetList"].children, sources...)
	}

	from := main.FromClause[0]
	fromInfo := mainInfo.children["FromClause"].children["0"]
	sources = w.addSourcesOfFromItem(from, fromInfo, sources...)

	return w.getSourceFromTargets(main.TargetList, mainInfo.children["TargetList"].children, sources...)
}

func (w *walker) getInsertSource(stmt *pg.InsertStmt, info nodeInfo) queryResult {
	table := w.getTableSource(stmt.Relation, info.children["Relation"])
	return w.getSourceFromTargets(stmt.ReturningList, info.children["ReturningList"].children, table)
}

func (w *walker) getUpdateSource(stmt *pg.UpdateStmt, info nodeInfo, sources ...queryResult) queryResult {
	sources = w.addSourcesOfWithClause(stmt.WithClause, info.children["WithClause"], sources...)

	table := w.getTableSource(stmt.Relation, info.children["Relation"])
	sources = append(sources, table)

	if len(stmt.FromClause) == 0 {
		return w.getSourceFromTargets(
			stmt.ReturningList,
			info.children["ReturningList"].children,
			sources...,
		)
	}

	from := stmt.FromClause[0]
	fromInfo := info.children["FromClause"].children["0"]
	sources = w.addSourcesOfFromItem(from, fromInfo, sources...)

	return w.getSourceFromTargets(
		stmt.ReturningList,
		info.children["ReturningList"].children,
		sources...,
	)
}

func (w *walker) getDeleteSource(stmt *pg.DeleteStmt, info nodeInfo, sources ...queryResult) queryResult {
	sources = w.addSourcesOfWithClause(stmt.WithClause, info.children["WithClause"], sources...)

	table := w.getTableSource(stmt.Relation, info.children["Relation"])
	sources = append(sources, table)

	if len(stmt.UsingClause) == 0 {
		return w.getSourceFromTargets(
			stmt.ReturningList,
			info.children["ReturningList"].children,
			sources...,
		)
	}

	from := stmt.UsingClause[0]
	fromInfo := info.children["UsingClause"].children["0"]
	sources = w.addSourcesOfFromItem(from, fromInfo, sources...)

	return w.getSourceFromTargets(
		stmt.ReturningList,
		info.children["ReturningList"].children,
		sources...,
	)
}

func (w *walker) addSourcesOfWithClause(with *pg.WithClause, info nodeInfo, sources ...queryResult) []queryResult {
	if with == nil {
		return sources
	}

	cteInfos := info.children["Ctes"]
	for i, cte := range with.Ctes {
		cteNodeWrap, ok := cte.Node.(*pg.Node_CommonTableExpr)
		if !ok {
			continue
		}

		cteNode := cteNodeWrap.CommonTableExpr
		cteInfo := cteInfos.children[strconv.Itoa(i)]

		stmtSource := w.getSource(
			cteNode.Ctequery, cteInfo.children["Ctequery"], sources...,
		)
		stmtSource.mustBeQualified = true

		if len(cteNode.Aliascolnames) != len(stmtSource.columns) {
			sources = append(sources, stmtSource)
			continue
		}

		aliasInfos := cteInfo.children["Aliascolnames"]
		for j := range cteNode.Aliascolnames {
			alias := w.names[aliasInfos.children[strconv.Itoa(j)].position()]
			if alias != "" {
				stmtSource.columns[j].name = alias
			}
		}

		sources = append(sources, stmtSource)
	}

	return sources
}

func (w *walker) addSourcesOfFromItem(from *pg.Node, fromInfo nodeInfo, sources ...queryResult) []queryResult {
	var joinedNodes []joinedInfo

	for {
		join := from.GetJoinExpr()
		if join == nil {
			break
		}
		joinInfo := fromInfo.children["JoinExpr"]

		// Update the main FROM
		from = join.Larg
		fromInfo = joinInfo.children["Larg"]

		infoKey := strings.TrimPrefix(
			reflect.TypeOf(join.Rarg.Node).Elem().Name(), "Node_")

		joined := joinedInfo{
			node:     join.Rarg,
			info:     joinInfo.children["Rarg"].children[infoKey],
			joinType: join.Jointype,
		}

		joinedNodes = append(joinedNodes, joined)
	}

	joinedNodes = append(joinedNodes, joinedInfo{
		node: from,
		info: fromInfo,
	})

	// Loop join in reverse order
	joinSources := make([]queryResult, 0, len(joinedNodes))
	for _, j := range slices.Backward(joinedNodes) {
		joinSource := w.getSource(
			j.node,
			j.info,
			sources...,
		)
		var right, left bool

		switch j.joinType {
		case pg.JoinType_JOIN_RIGHT:
			right = true

		case pg.JoinType_JOIN_LEFT:
			left = true

		case pg.JoinType_JOIN_FULL:
			right = true
			left = true
		}

		if right {
			for i := range joinSources {
				for j := range joinSources[i].columns {
					joinSources[i].columns[j].nullable = true
				}
			}
		}
		if left {
			for i := range joinSource.columns {
				joinSource.columns[i].nullable = true
			}
		}

		joinSources = append(joinSources, joinSource)
	}

	return append(sources, joinSources...)
}

func (w *walker) getSourceFromTargets(targets []*pg.Node, infos map[string]nodeInfo, sources ...queryResult) queryResult {
	if len(targets) != len(infos) {
		return queryResult{}
	}

	source := queryResult{
		columns: make([]col, 0, len(targets)),
	}

	var prefix string

	for i, target := range targets {
		targetInfo := infos[strconv.Itoa(i)]
		pos := targetInfo.position()

		if newPrefix, found := w.getPrefixAnnotation(pos[0]); found {
			prefix = newPrefix
		}

		if w.names[pos] == "*" {
			if w.getConfigComment(pos[1]) != "" {
				w.errors = append(w.errors, fmt.Errorf("no comments after STAR column"))
			}

			source.columns = append(
				source.columns,
				w.getStarColumns(target, targetInfo, prefix, sources...)...,
			)

			continue
		}

		column := col{
			pos:  pos,
			name: w.names[pos],
		}

		if nullable := w.nullability[pos]; nullable != nil {
			column.nullable = nullable.IsNull(w.names, w.nullability, sources)
		}

		resTarget := target.GetResTarget()
		if resTarget != nil && resTarget.Name != "" {
			column.name = resTarget.GetName()
		}

		column.name = prefix + column.name
		source.columns = append(source.columns, column)

		if prefix != "" {
			valInfo := targetInfo.children["ResTarget"].children["Val"]
			w.editRules = append(
				w.editRules,
				internal.Replace(
					int(valInfo.end), int(targetInfo.end)-1,
					fmt.Sprintf(" AS %q", column.name),
				),
			)
		}
	}

	return source
}

func (w *walker) getSourceFromList(target *pg.List, info nodeInfo, sources ...queryResult) queryResult {
	result := queryResult{
		columns: make([]col, len(target.Items)),
	}

	itemsInfo := info.children["Items"]
	for i := range target.Items {
		pos := itemsInfo.children[strconv.Itoa(i)].position()
		result.columns[i] = col{
			pos:  pos,
			name: fmt.Sprintf("column%d", i+1),
		}

		if nullable, ok := w.nullability[pos]; ok {
			result.columns[i].nullable = nullable.IsNull(w.names, w.nullability, sources)
		}
	}

	return result
}

func (w *walker) getStarColumns(target *pg.Node, info nodeInfo, prefix string, sources ...queryResult) []col {
	if target == nil {
		return nil
	}

	fields := target.GetResTarget().GetVal().GetColumnRef().GetFields()
	if len(fields) == 0 {
		return nil
	}

	var schema, table, column string

	fieldsInfo := info.
		children["ResTarget"].
		children["Val"].
		children["ColumnRef"].
		children["Fields"]

	for i := range fields {
		name := w.names[fieldsInfo.children[strconv.Itoa(i)].position()]
		switch {
		case i == len(fields)-1:
			column = name
		case i == len(fields)-2:
			table = name
		case i == len(fields)-3:
			schema = name
		}
	}

	if column != "*" {
		panic("getStarColumns when column != '*'")
	}

	var columns []col

	w.editRules = append(
		w.editRules,
		internal.Delete(int(fieldsInfo.start), int(fieldsInfo.end)-1),
	)

	buf := &strings.Builder{}
	var i int
	for _, source := range sources {
		if source.mustBeQualified {
			continue
		}

		if table != "" && source.name != table {
			continue
		}

		if schema != "" && source.schema != schema {
			continue
		}

		columns = append(columns, source.columns...)

		if i > 0 {
			buf.WriteString(", ")
		}
		expandQuotedSource(buf, source, prefix)
		i++
	}
	w.editRules = append(
		w.editRules,
		internal.Insert(int(fieldsInfo.start), buf.String()),
	)

	for i := range columns {
		columns[i].name = prefix + columns[i].name
	}

	return columns
}

func expandQuotedSource(buf *strings.Builder, source queryResult, prefix string) {
	for i, col := range source.columns {
		if i > 0 {
			buf.WriteString(", ")
		}
		if source.schema != "" {
			fmt.Fprintf(buf, "%q.%q.%q AS %q", source.schema, source.name, col.name, prefix+col.name)
		} else {
			fmt.Fprintf(buf, "%q.%q AS %q", source.name, col.name, prefix+col.name)
		}
	}
}
