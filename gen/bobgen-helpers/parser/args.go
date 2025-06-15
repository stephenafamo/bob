package parser

import (
	"fmt"
	"slices"

	"github.com/stephenafamo/bob/gen/drivers"
)

func GetArgs(bindArgs, groupArgs []drivers.QueryArg) []drivers.QueryArg {
	argIsInGroup := make([]bool, len(bindArgs))
	groupIsInGroup := make([]bool, len(groupArgs))

	// Sort the groups by the size of the group
	slices.SortStableFunc(groupArgs, func(a, b drivers.QueryArg) int {
		return (a.Positions[0][1] - a.Positions[0][0]) -
			(b.Positions[0][1] - b.Positions[0][0])
	})

	for groupIndex, group := range groupArgs {
		var groupChildren []drivers.QueryArg
		for argIndex, arg := range bindArgs {
			// Do not add args with multiple positions into groups
			if len(arg.Positions) > 1 {
				continue
			}

			// So we don't add the same arg to multiple groups
			if argIsInGroup[argIndex] {
				continue
			}

			if group.Positions[0][0] <= arg.Positions[0][0] &&
				arg.Positions[0][1] <= group.Positions[0][1] {
				argIsInGroup[argIndex] = true
				groupChildren = append(groupChildren, bindArgs[argIndex])
			}
		}

		// If there is a smaller group that is a subset of this group
		// we add the smaller group to this group's children
		for smallGroupIndex, smallerGroup := range groupArgs[:groupIndex] {
			// Do not add empty groups
			if len(groupArgs[smallGroupIndex].Children) == 0 {
				continue
			}

			// So we don't add the same group to multiple groups
			if groupIsInGroup[smallGroupIndex] {
				continue
			}

			if group.Positions[0][0] <= smallerGroup.Positions[0][0] &&
				smallerGroup.Positions[0][1] <= group.Positions[0][1] {
				groupChildren = append(groupChildren, groupArgs[smallGroupIndex])
				groupIsInGroup[smallGroupIndex] = true
				continue
			}
		}

		// If there are no children, we can skip this group
		if len(groupChildren) == 0 {
			continue
		}

		sortAndFixNames(groupChildren)
		groupArgs[groupIndex].Children = groupChildren
	}

	allArgs := make([]drivers.QueryArg, 0, len(bindArgs)+len(groupArgs))
	for i, arg := range bindArgs {
		if argIsInGroup[i] {
			continue
		}
		allArgs = append(allArgs, arg)
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
			if !group.CanBeMultiple {
				allArgs = append(allArgs, group.Children[0])
				continue
			}
			fallthrough
		default:
			allArgs = append(allArgs, group)
		}
	}

	sortAndFixNames(allArgs)
	return allArgs
}

func sortAndFixNames(args []drivers.QueryArg) {
	// Sort the args by their original position
	sortArgsByPosition(args)

	// Fix duplicate arg names
	fixDuplicateArgNames(args)
}

func sortArgsByPosition(args []drivers.QueryArg) {
	slices.SortStableFunc(args, func(a, b drivers.QueryArg) int {
		beginningDiff := int(a.Positions[0][0] - b.Positions[0][0])
		if beginningDiff == 0 {
			return int(a.Positions[0][1] - b.Positions[0][1])
		}
		return beginningDiff
	})
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
