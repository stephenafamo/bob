package parser

import (
	"github.com/stephenafamo/bob/gen/bobgen-helpers/parser"
	"github.com/stephenafamo/bob/gen/drivers"
	sqliteparser "github.com/stephenafamo/sqlparser/sqlite"
)

func (v *visitor) getArgs(start, stop int) []drivers.QueryArg {
	args, groups := v.sortExprsIntoArgsAndGroups(start, stop)
	return v.GetArgs(args, groups, TranslateColumnType)
}

func (v *visitor) sortExprsIntoArgsAndGroups(start, stop int) ([]NodeInfo, []NodeInfo) {
	args := []NodeInfo{}
	groups := []NodeInfo{}

	// Sort the exprs into groups and binds
	for _, expr := range v.Infos {
		if expr.Node.GetStart().GetStart() < start || expr.Node.GetStop().GetStop() > stop {
			continue
		}

		if expr.IsGroup {
			groups = append(groups, expr)
			continue
		}

		if _, ok := expr.Node.(*sqliteparser.Expr_bindContext); !ok {
			continue
		}

		args = append(args, expr)
	}

	for i := range args {
		// Merge in case the name is configured in the bind
		args[i].Config = args[i].Config.Merge(
			parser.ParseQueryColumnConfig(v.getCommentToRight(args[i].Node)),
		)
	}

	for i := range groups {
		// Merge in case the name is configured in the bind
		groups[i].Config = groups[i].Config.Merge(
			parser.ParseQueryColumnConfig(v.getCommentToRight(groups[i].Node)),
		)
	}

	return args, groups
}
