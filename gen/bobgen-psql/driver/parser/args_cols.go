package parser

import (
	"context"
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strconv"
	"strings"

	"github.com/aarondl/opt/omit"
	"github.com/gofrs/uuid"
	pg "github.com/pganalyze/pg_query_go/v6"
	"github.com/stephenafamo/bob/gen/drivers"
	"github.com/stephenafamo/bob/internal"
	"github.com/stephenafamo/scan"
	"github.com/stephenafamo/scan/stdscan"
)

type argInfo struct {
	ArgType    string // "parameter" or "result"
	ColumnType string
	ColInfo
}

const queryInfo = `
	WITH prepared_details AS (
  SELECT
    'parameter' "type",
    u.*
  FROM
    pg_prepared_statements
    CROSS JOIN unnest(parameter_types::oid[])
    WITH ORDINALITY AS u ("oid", "index")
  WHERE
    name = $1
  UNION ALL
  SELECT
    'result' "type",
    u.*
  FROM
    pg_prepared_statements
    CROSS JOIN unnest(result_types::oid[])
    WITH ORDINALITY AS u ("oid", "index")
  WHERE
    name = $1
)
SELECT
  prep.type AS arg_type,
  CASE WHEN pg_type.typtype = 'e' THEN
    'ENUM'
  WHEN pg_type.typelem > 0 THEN
    'ARRAY'
  ELSE
    pg_type.typname
  END AS column_type,
  pg_type.typname AS udt_name,
  pgn.nspname AS udt_schema,
  CASE WHEN pg_type.typtype = 'e' THEN
    'USER-DEFINED'
  ELSE
    pg_type.typname
  END AS arr_type
FROM
  prepared_details prep
  LEFT JOIN pg_type ON pg_type.oid = prep.oid
  LEFT JOIN pg_namespace AS pgn ON pgn.oid = pg_type.typnamespace
ORDER BY
  type,
  index;
`

func (p *Parser) getArgsAndCols(ctx context.Context, q string) ([]string, []string, error) {
	queryID, err := uuid.NewV4()
	if err != nil {
		return nil, nil, fmt.Errorf("uuid: %w", err)
	}

	_, err = p.conn.ExecContext(ctx, fmt.Sprintf("PREPARE %q AS %s", queryID.String(), q))
	if err != nil {
		return nil, nil, fmt.Errorf("prepare: %w", err)
	}

	info, err := stdscan.All(
		ctx, p.conn, scan.StructMapper[argInfo](),
		queryInfo, queryID.String(),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("query: %w", err)
	}

	_, err = p.conn.ExecContext(ctx, fmt.Sprintf("DEALLOCATE %q", queryID.String()))
	if err != nil {
		return nil, nil, fmt.Errorf("deallocate: %w", err)
	}

	args := make([]string, 0, len(info))
	cols := make([]string, 0, len(info))

	for _, i := range info {
		driverCol := p.translator.TranslateColumnType(drivers.Column{DBType: i.ColumnType}, i.ColInfo)
		switch i.ArgType {
		case "parameter":
			args = append(args, driverCol.Type)
		case "result":
			cols = append(cols, driverCol.Type)
		default:
			return nil, nil, fmt.Errorf("unknown arg type: %s", i.ArgType)
		}
	}

	return args, cols, nil
}

func (w *walker) getSource(node *pg.Node, info nodeInfo, sources ...queryResult) queryResult {
	cloned := slices.Clone(sources)

	switch stmt := node.Node.(type) {
	case *pg.Node_SelectStmt:
		return w.getSelectSource(stmt.SelectStmt, info, cloned...)

	case *pg.Node_RangeVar:
		sub := stmt.RangeVar
		source := getTableSource(
			w.db,
			sub.GetSchemaname(),
			sub.GetRelname(),
		)
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

	case *pg.Node_RangeSubselect:
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

func (w *walker) getSelectSource(stmt *pg.SelectStmt, info nodeInfo, sources ...queryResult) queryResult {
	if stmt.WithClause != nil {
		cteInfos := info.children["WithClause"].children["Ctes"]
		for i, cte := range stmt.WithClause.Ctes {
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
	}

	main := stmt
	mainInfo := info

	for main.Larg != nil {
		main = main.Larg
		mainInfo = mainInfo.children["Larg"]
	}

	if len(main.FromClause) == 0 {
		return w.getSourceFromTargets(main.TargetList, mainInfo.children["TargetList"].children, sources...)
	}

	var joinedNodes []joinedInfo

	from := main.FromClause[0]
	fromInfo := mainInfo.children["FromClause"].children["0"]

	// var joinSources []querySource
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

		joinedNodes = append(joinedNodes, joinedInfo{
			join.Rarg,
			joinInfo.children["Rarg"].children[infoKey],
		})
	}

	joinedNodes = append(joinedNodes, joinedInfo{
		node: from,
		info: fromInfo,
	})

	// Loop join in reverse order
	for _, j := range slices.Backward(joinedNodes) {
		joinSource := w.getSource(
			j.node,
			j.info,
			sources...,
		)
		sources = append(sources, joinSource)
	}

	return w.getSourceFromTargets(main.TargetList, mainInfo.children["TargetList"].children, sources...)
}

func (w *walker) getSourceFromTargets(targets []*pg.Node, infos map[string]nodeInfo, sources ...queryResult) queryResult {
	if len(targets) != len(infos) {
		return queryResult{}
	}

	source := queryResult{
		columns: make([]col, 0, len(targets)),
	}

	for i, target := range targets {
		targetInfo := infos[strconv.Itoa(i)]
		pos := targetInfo.position()

		column := col{
			pos:  pos,
			name: w.names[pos],
		}

		if getConfigComment(w.input, w.tokens, pos) != "" {
			w.errors = append(w.errors, fmt.Errorf("no comments after STAR column"))
		}

		if column.name == "*" {
			source.columns = append(
				source.columns,
				w.getStarColumns(target, targetInfo, sources...)...,
			)
			continue
		}

		if nullable := w.nullability[pos]; nullable != nil {
			column.nullable = nullable.IsNull(w.names, w.nullability, sources)
		}

		resTarget := target.GetResTarget()
		if resTarget != nil && resTarget.Name != "" {
			column.name = resTarget.GetName()
		}

		source.columns = append(source.columns, column)
	}

	return source
}

func (w *walker) getStarColumns(target *pg.Node, info nodeInfo, sources ...queryResult) []col {
	if target == nil {
		return nil
	}
	fmt.Println("getStarColumns", target.String())

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
		expandQuotedSource(buf, source)
		i++
	}
	w.editRules = append(
		w.editRules,
		internal.Insert(int(fieldsInfo.start), buf.String()),
	)

	return columns
}

func expandQuotedSource(buf *strings.Builder, source queryResult) {
	for i, col := range source.columns {
		if i > 0 {
			buf.WriteString(", ")
		}
		if source.schema != "" {
			fmt.Fprintf(buf, "%q.%q.%q", source.schema, source.name, col.name)
		} else {
			fmt.Fprintf(buf, "%q.%q", source.name, col.name)
		}
	}
}

func (w *walker) getArgs(typs []string) []drivers.QueryArg {
	args := make([]drivers.QueryArg, len(w.args))
	for i, arg := range w.args {
		name := fmt.Sprintf("arg%d", i+1)
		positions := make([][2]int, 0, len(arg))
		var nullable, multiple bool
		configs := make([]drivers.QueryCol, 0, len(arg))

		for _, pos := range arg {
			positions = append(positions, pos.edited)
			if w.names[pos.original] != "" {
				name = w.names[pos.original]
			}
			if w.nullability[pos.original] != nil {
				nullable = nullable || w.nullability[pos.original].IsNull(w.names, w.nullability, nil)
			}

			_, isMultiple := w.multiple[pos.edited]
			multiple = multiple || isMultiple
			configs = append(configs, drivers.ParseQueryColumnConfig(
				getConfigComment(w.input, w.tokens, pos.original),
			))
		}

		args[i] = drivers.QueryArg{
			Col: drivers.QueryCol{
				Name:     name,
				Nullable: omit.From(nullable),
				TypeName: typs[i],
			}.Merge(configs...),
			Positions:     positions,
			CanBeMultiple: multiple,
		}
	}

	groups := slices.Collect(maps.Keys(w.groups))
	slices.SortStableFunc(groups, func(a, b argPos) int {
		return (a.edited[1] - a.edited[0] - (b.edited[1] - b.edited[0]))
	})

	argIsInGroup := make([]bool, len(args))
	groupIsInGroup := make([]bool, len(groups))

	groupArgs := make([]drivers.QueryArg, len(groups))
	for groupIndex, group := range groups {
		var groupChildren []drivers.QueryArg
		for argIndex, arg := range args {
			// If the arg is in the group, we add it to the group's children
			if group.edited[0] <= w.args[argIndex][0].edited[0] && w.args[argIndex][0].edited[1] <= group.edited[1] {
				groupChildren = append(groupChildren, arg)
				argIsInGroup[argIndex] = true
				continue
			}
		}

		// If there is a smaller group that is a subset of this group, we add the arg to that group
		for smallGroupIndex, smallerGroup := range groups[:groupIndex] {
			if len(groupArgs[smallGroupIndex].Positions) == 0 {
				continue
			}
			if group.edited[0] <= smallerGroup.edited[0] && smallerGroup.edited[1] <= group.edited[1] {
				groupChildren = append(groupChildren, groupArgs[smallGroupIndex])
				groupIsInGroup[smallGroupIndex] = true
				continue
			}
		}

		if len(groupChildren) == 0 {
			// If there are no children, we can skip this group
			continue
		}

		// sort the children by their original position
		slices.SortFunc(groupChildren, func(a, b drivers.QueryArg) int {
			beginningDiff := int(a.Positions[0][0] - b.Positions[0][0])
			if beginningDiff == 0 {
				return int(a.Positions[0][1] - b.Positions[0][1])
			}
			return beginningDiff
		})
		fixDuplicateArgNames(groupChildren)

		_, multiple := w.multiple[group.edited]
		groupConfig := drivers.ParseQueryColumnConfig(
			getConfigComment(w.input, w.tokens, group.original),
		)

		groupArgs[groupIndex] = drivers.QueryArg{
			Col: drivers.QueryCol{
				Name:     fmt.Sprintf("group%d", group.edited[0]),
				Nullable: omit.From(false),
			}.Merge(groupConfig),
			Children:      groupChildren,
			Positions:     [][2]int{{int(group.edited[0]), int(group.edited[1])}},
			CanBeMultiple: multiple,
		}
		if name, ok := w.names[group.original]; ok && name != "" {
			groupArgs[groupIndex].Col.Name = name
		}
	}

	allArgs := make([]drivers.QueryArg, 0, len(args)+len(groupArgs))
	for i, arg := range args {
		if !argIsInGroup[i] {
			allArgs = append(allArgs, arg)
		}
	}

	for i, group := range groupArgs {
		if groupIsInGroup[i] {
			continue
		}

		switch len(group.Children) {
		case 0:
			// Do nothing
			continue
		case 1:
			// If the child arg has the same positions as the group, we can just use the child arg
			if group.Children[0].Positions[0][0] == group.Positions[0][0] &&
				group.Children[0].Positions[0][1] == group.Positions[0][1] {
				allArgs = append(allArgs, group.Children[0])
			} else {
				allArgs = append(allArgs, group)
			}
		default:
			allArgs = append(allArgs, group)
		}
	}

	slices.SortStableFunc(allArgs, func(a, b drivers.QueryArg) int {
		return int(a.Positions[0][0] - b.Positions[0][0])
	})
	fixDuplicateArgNames(allArgs)

	return allArgs
}

func fixDuplicateArgNames(args []drivers.QueryArg) {
	names := make(map[string]int, len(args))
	for i := range args {
		if args[i].Col.Name == "" {
			continue
		}
		name := args[i].Col.Name
		index := names[name]
		names[name] = index + 1
		if index > 0 {
			args[i].Col.Name = fmt.Sprintf("%s_%d", name, index+1)
		}
	}
}
