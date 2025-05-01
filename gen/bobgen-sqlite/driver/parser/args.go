package parser

import (
	"fmt"
	"slices"

	"github.com/aarondl/opt/omit"
	"github.com/stephenafamo/bob/gen/drivers"
	sqliteparser "github.com/stephenafamo/sqlparser/sqlite"
)

func (v *visitor) getArgs(start, stop int) []drivers.QueryArg {
	args, groups := v.sortExprsIntoArgsAndGroups(start, stop)

	argIsInGroup := make([]bool, len(args))
	groupIsInGroup := make([]bool, len(groups))

	queryArgs := make([]drivers.QueryArg, len(args))
	keys := make(map[string]int, len(queryArgs))
	for i, arg := range args {
		key := arg.queryArgKey
		if oldIndex, ok := keys[key]; ok && key != "" {
			queryArgs[oldIndex].Positions = append(
				queryArgs[oldIndex].Positions, arg.EditedPosition,
			)
			continue
		}
		keys[arg.queryArgKey] = len(queryArgs)

		name := v.getNameString(arg.expr)
		if name == "" {
			name = fmt.Sprintf("arg%d", i+1)
		}

		queryArgs[i] = drivers.QueryArg{
			Col: drivers.QueryCol{
				Name:     name,
				Nullable: omit.From(arg.Type.Nullable()),
				TypeName: v.getDBType(arg).Type(v.db),
			}.Merge(arg.config),
			Positions:     [][2]int{arg.EditedPosition},
			CanBeMultiple: arg.CanBeMultiple,
		}
	}
	queryArgs = slices.DeleteFunc(queryArgs, func(q drivers.QueryArg) bool {
		return len(q.Positions) == 0
	})

	// Sort the groups by the size of the group
	slices.SortStableFunc(groups, func(a, b exprInfo) int {
		return (a.expr.GetStop().GetStop() - a.expr.GetStart().GetStart()) -
			(b.expr.GetStop().GetStop() - b.expr.GetStart().GetStart())
	})

	groupArgs := make([]drivers.QueryArg, len(groups))
	for groupIndex, group := range groups {
		var groupChildren []drivers.QueryArg
		for argIndex, arg := range args {
			if key(group.expr) == key(arg.expr) {
				continue
			}

			if group.expr.GetStart().GetStart() <= arg.expr.GetStart().GetStart() &&
				arg.expr.GetStop().GetStop() <= group.expr.GetStop().GetStop() {
				argIsInGroup[argIndex] = true
				groupChildren = append(groupChildren, queryArgs[argIndex])
			}
		}

		// If there is a smaller group that is a subset of this group, we add the arg to that group
		for smallGroupIndex, smallerGroup := range groups[:groupIndex] {
			if len(groupArgs[smallGroupIndex].Positions) == 0 {
				continue
			}

			if group.expr.GetStart().GetStart() <= smallerGroup.expr.GetStart().GetStart() &&
				smallerGroup.expr.GetStop().GetStop() <= group.expr.GetStop().GetStop() {
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
			Children:      groupChildren,
			Positions:     [][2]int{group.EditedPosition},
			CanBeMultiple: group.CanBeMultiple,
		}
	}

	allArgs := make([]drivers.QueryArg, 0, len(args)+len(groupArgs))
	for i, arg := range queryArgs {
		if argIsInGroup[i] {
			continue
		}
		allArgs = append(allArgs, arg)
	}

	for i, group := range groupArgs {
		if groupIsInGroup[i] {
			continue
		}

		switch {
		case len(group.Children) == 0:
			// Do nothing
			continue
		case len(group.Children) == 1:
			if !group.CanBeMultiple {
				allArgs = append(allArgs, group.Children[0])
				continue
			}
			fallthrough
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
			drivers.ParseQueryColumnConfig(args[i].options),
		)
	}

	for i := range groups {
		groups[i].options = v.getCommentToRight(groups[i].expr)
		// Merge in case the name is configured in the bind
		groups[i].config = groups[i].config.Merge(
			drivers.ParseQueryColumnConfig(groups[i].options),
		)
	}

	return args, groups
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
