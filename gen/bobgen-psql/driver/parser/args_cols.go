package parser

import (
	"context"
	"fmt"
	"maps"
	"slices"

	"github.com/gofrs/uuid"
	"github.com/stephenafamo/bob/gen/bobgen-helpers/parser"
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

func (w *walker) getArgs(typs []string) []drivers.QueryArg {
	bindArgs := make([]drivers.QueryArg, len(w.args))
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
			configs = append(configs, parser.ParseQueryColumnConfig(
				w.getConfigComment(pos.original[1]),
			))
		}

		bindArgs[i] = drivers.QueryArg{
			Col: drivers.QueryCol{
				Name:     name,
				Nullable: internal.Pointer(nullable),
				TypeName: typs[i],
			}.Merge(configs...),
			Positions:     positions,
			CanBeMultiple: multiple,
		}
	}

	groups := slices.Collect(maps.Keys(w.groups))
	groupArgs := make([]drivers.QueryArg, len(groups))
	for groupIndex, group := range groups {
		name := fmt.Sprintf("group%d", group.edited[0])
		if computedName, ok := w.names[group.original]; ok && computedName != "" {
			name = computedName
		}

		_, multiple := w.multiple[group.edited]
		groupArgs[groupIndex] = drivers.QueryArg{
			Col: drivers.QueryCol{
				Name:     name,
				Nullable: internal.Pointer(false),
			}.Merge(parser.ParseQueryColumnConfig(
				w.getConfigComment(group.original[1]),
			)),
			Positions:     [][2]int{{int(group.edited[0]), int(group.edited[1])}},
			CanBeMultiple: multiple,
		}
	}

	return parser.GetArgs(bindArgs, groupArgs)
}
