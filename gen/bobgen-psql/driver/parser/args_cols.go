package parser

import (
	"context"
	"fmt"
	"maps"
	"slices"

	"github.com/aarondl/opt/omit"
	"github.com/gofrs/uuid"
	"github.com/stephenafamo/bob/gen/drivers"
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
				w.getConfigComment(pos.original),
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
			w.getConfigComment(group.original),
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
