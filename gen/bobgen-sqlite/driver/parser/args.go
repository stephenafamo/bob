package parser

import (
	"fmt"
	"slices"

	"github.com/aarondl/opt/omit"
	"github.com/stephenafamo/bob/gen/bobgen-helpers/parser"
	"github.com/stephenafamo/bob/gen/drivers"
	sqliteparser "github.com/stephenafamo/sqlparser/sqlite"
)

func (v *visitor) getArgs(start, stop int) []drivers.QueryArg {
	args, groups := v.sortExprsIntoArgsAndGroups(start, stop)

	bindArgs := make([]drivers.QueryArg, len(args))
	keys := make(map[string]int, len(bindArgs))
	for i, arg := range args {
		key := arg.queryArgKey
		if oldIndex, ok := keys[key]; ok && key != "" {
			bindArgs[oldIndex].Positions = append(
				bindArgs[oldIndex].Positions, arg.EditedPosition,
			)
			continue
		}
		keys[arg.queryArgKey] = len(bindArgs)

		name := v.getNameString(arg.expr)
		if name == "" {
			name = fmt.Sprintf("arg%d", i+1)
		}

		bindArgs[i] = drivers.QueryArg{
			Col: drivers.QueryCol{
				Name:     name,
				Nullable: omit.From(arg.Type.Nullable()),
				TypeName: v.getDBType(arg).Type(v.db),
			}.Merge(arg.config),
			Positions:     [][2]int{arg.EditedPosition},
			CanBeMultiple: arg.CanBeMultiple,
		}
	}
	bindArgs = slices.DeleteFunc(bindArgs, func(q drivers.QueryArg) bool {
		return len(q.Positions) == 0
	})

	groupArgs := make([]drivers.QueryArg, len(groups))
	for groupIndex, group := range groups {
		name := v.getNameString(group.expr)
		if name == "" {
			name = fmt.Sprintf("group%d", group.EditedPosition[0])
		}

		groupArgs[groupIndex] = drivers.QueryArg{
			Col: drivers.QueryCol{
				Name:     name,
				Nullable: omit.From(group.Type.Nullable()),
				TypeName: v.getDBType(group).Type(v.db),
			}.Merge(group.config),
			Positions:     [][2]int{group.EditedPosition},
			CanBeMultiple: group.CanBeMultiple,
		}
	}

	return parser.GetArgs(bindArgs, groupArgs)
}

func (v *visitor) sortExprsIntoArgsAndGroups(start, stop int) ([]exprInfo, []exprInfo) {
	args := make([]exprInfo, 0, len(v.exprs))
	groups := make([]exprInfo, 0, len(v.exprs))

	// Sort the exprs into groups and binds
	for _, expr := range v.exprs {
		if expr.expr.GetStart().GetStart() < start || expr.expr.GetStop().GetStop() > stop {
			continue
		}

		if expr.isGroup {
			groups = append(groups, expr)
			continue
		}

		if _, ok := expr.expr.(*sqliteparser.Expr_bindContext); !ok {
			continue
		}

		args = append(args, expr)
	}

	for i := range args {
		args[i].options = v.getCommentToRight(args[i].expr)
		// Merge in case the name is configured in the bind
		args[i].config = args[i].config.Merge(
			parser.ParseQueryColumnConfig(args[i].options),
		)
	}

	for i := range groups {
		groups[i].options = v.getCommentToRight(groups[i].expr)
		// Merge in case the name is configured in the bind
		groups[i].config = groups[i].config.Merge(
			parser.ParseQueryColumnConfig(groups[i].options),
		)
	}

	return args, groups
}
