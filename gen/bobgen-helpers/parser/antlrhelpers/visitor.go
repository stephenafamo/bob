package antlrhelpers

import (
	"fmt"
	"slices"
	"sync/atomic"

	"github.com/aarondl/opt/omit"
	"github.com/stephenafamo/bob/gen/bobgen-helpers/parser"
	"github.com/stephenafamo/bob/gen/drivers"
	"github.com/stephenafamo/bob/internal"
)

type Visitor[C, I any] struct {
	Err       error
	DB        drivers.Tables[C, I]
	Names     map[NodeKey]string
	Infos     map[NodeKey]NodeInfo
	Sources   []QuerySource
	Functions Functions
	BaseRules []internal.EditRule

	// Refresh these for each statement
	StmtRules []internal.EditRule
	Atom      *atomic.Int64
}

func (v *Visitor[C, I]) UpdateInfo(info NodeInfo) {
	key := Key(info.Node)

	currentExpr, ok := v.Infos[key]
	if !ok {
		v.Infos[key] = info
		return
	}

	currentExpr.Node = info.Node
	currentExpr.IsGroup = currentExpr.IsGroup || info.IsGroup
	currentExpr.CanBeMultiple = currentExpr.CanBeMultiple || info.CanBeMultiple

	if info.EditedPosition != [2]int{} {
		currentExpr.EditedPosition = info.EditedPosition
	}

	if info.ExprDescription != "" {
		currentExpr.ExprDescription += ","
		currentExpr.ExprDescription += info.ExprDescription
	}

	if info.ExprRef != nil {
		currentExpr.ExprRef = info.ExprRef
		currentExpr.IgnoreRefNullability = info.IgnoreRefNullability
	}

	if info.Type == nil {
		v.Infos[key] = currentExpr
		return
	}

	if currentExpr.Type == nil {
		currentExpr.Type = info.Type
		v.Infos[key] = currentExpr
		return
	}

	matchingDBTypes := currentExpr.Type.Match(info.Type)
	if len(matchingDBTypes) == 0 {
		panic(fmt.Sprintf(
			"No matching DBType found for %s: \n%v\n%v",
			info.Node.GetText(),
			currentExpr.Type.List(),
			info.Type.List(),
		))
	}

	currentExpr.Type = matchingDBTypes
	v.Infos[key] = currentExpr
}

func (v Visitor[C, I]) GetName(expr Node) string {
	exprKey := Key(expr)
	return v.Names[exprKey]
}

func (v *Visitor[C, I]) MaybeSetName(ctx Node, name string) {
	if name == "" {
		return
	}

	key := Key(ctx)
	_, ok := v.Names[key]
	if ok {
		return
	}

	v.Names[key] = name
}

func (w *Visitor[C, I]) MatchNames(p1 Node, p2 Node) {
	w.MaybeSetName(p1, w.Names[Key(p2)])
	w.MaybeSetName(p2, w.Names[Key(p1)])
}

func (v Visitor[C, I]) GetArgs(args, groups []NodeInfo, translate func(string) string) []drivers.QueryArg {
	bindArgs := make([]drivers.QueryArg, len(args))
	keys := make(map[string]int, len(bindArgs))
	for i, arg := range args {
		key := arg.ArgKey
		if oldIndex, ok := keys[key]; ok && key != "" {
			bindArgs[oldIndex].Positions = append(
				bindArgs[oldIndex].Positions, arg.EditedPosition,
			)
			// if an arg is used multiple times, it can't be multiple
			bindArgs[oldIndex].CanBeMultiple = false
			// Merge the config
			bindArgs[oldIndex].Col = bindArgs[oldIndex].Col.Merge(arg.Config)
			continue
		}
		keys[arg.ArgKey] = i

		name := v.Names[Key(arg.Node)]
		if name == "" {
			name = "arg"
		}

		bindArgs[i] = drivers.QueryArg{
			Col: drivers.QueryCol{
				Name:     name,
				Nullable: omit.From(arg.Type.Nullable()),
				TypeName: translate(GetDBType(v.Infos, arg).ConfirmedDBType()),
			}.Merge(arg.Config),
			Positions:     [][2]int{arg.EditedPosition},
			CanBeMultiple: arg.CanBeMultiple,
		}
	}
	bindArgs = slices.DeleteFunc(bindArgs, func(q drivers.QueryArg) bool {
		return len(q.Positions) == 0
	})

	groupArgs := make([]drivers.QueryArg, len(groups))
	for groupIndex, group := range groups {
		name := v.Names[Key(group.Node)]
		if name == "" {
			name = "group"
		}

		groupArgs[groupIndex] = drivers.QueryArg{
			Col: drivers.QueryCol{
				Name:     name,
				Nullable: omit.From(group.Type.Nullable()),
				TypeName: translate(GetDBType(v.Infos, group).ConfirmedDBType()),
			}.Merge(group.Config),
			Positions:     [][2]int{group.EditedPosition},
			CanBeMultiple: group.CanBeMultiple,
		}
	}

	return parser.GetArgs(bindArgs, groupArgs)
}
