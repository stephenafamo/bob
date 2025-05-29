package antlrhelpers

import (
	"fmt"
	"slices"
	"strconv"
	"sync/atomic"

	"github.com/stephenafamo/bob/gen/bobgen-helpers/parser"
	"github.com/stephenafamo/bob/gen/drivers"
	"github.com/stephenafamo/bob/internal"
)

type Visitor[C, I any] struct {
	Err       error
	DB        drivers.Tables[C, I]
	Args      []NodeKey
	Groups    []NodeKey
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
	if expr == nil {
		return ""
	}
	exprKey := Key(expr)
	return v.Names[exprKey]
}

func (v *Visitor[C, I]) SetArg(expr Node) {
	exprKey := Key(expr)
	if slices.Contains(v.Args, exprKey) {
		return
	}
	v.Args = append(v.Args, exprKey)
}

func (v *Visitor[C, I]) SetGroup(expr Node) {
	exprKey := Key(expr)
	if slices.Contains(v.Groups, exprKey) {
		return
	}
	v.Groups = append(v.Groups, exprKey)
}

func (v *Visitor[C, I]) MaybeSetName(key NodeKey, name string) {
	if name == "" {
		return
	}

	_, ok := v.Names[key]
	if ok {
		return
	}

	v.Names[key] = name
}

func (v *Visitor[C, I]) MaybeSetNodeName(ctx Node, name string) {
	v.MaybeSetName(Key(ctx), name)
}

func (w *Visitor[C, I]) MatchNames(p1, p2 NodeKey) {
	w.MaybeSetName(p1, w.Names[p2])
	w.MaybeSetName(p2, w.Names[p1])
}

func (v *Visitor[C, I]) MatchNodeNames(p1 Node, p2 Node) {
	v.MatchNames(Key(p1), Key(p2))
}

func (v Visitor[C, I]) GetArgs(start, stop int, translate func(string) (string, []string), comment func(Node) string) []drivers.QueryArg {
	groupArgs := make([]drivers.QueryArg, len(v.Groups))
	bindArgs := make([]drivers.QueryArg, len(v.Args))
	keys := make(map[string]int, len(bindArgs))

	for i, argNodeKey := range v.Args {
		arg, ok := v.Infos[argNodeKey]
		if !ok {
			continue
		}

		if arg.Node.GetStart().GetStart() < start || arg.Node.GetStop().GetStop() > stop {
			continue
		}

		key := arg.ArgKey
		if oldIndex, ok := keys[key]; ok && key != "" {
			bindArgs[oldIndex].Positions = append(
				bindArgs[oldIndex].Positions, arg.EditedPosition,
			)
			// if an arg is used multiple times, it can't be multiple
			bindArgs[oldIndex].CanBeMultiple = false
			// Merge the config
			bindArgs[oldIndex].Col = bindArgs[oldIndex].Col.Merge(
				parser.ParseQueryColumnConfig(comment(arg.Node)),
			)
			continue
		}
		keys[arg.ArgKey] = i

		name := v.Names[Key(arg.Node)]
		if _, notNumber := strconv.Atoi(arg.ArgKey); notNumber != nil && arg.ArgKey != "" {
			name = arg.ArgKey
		}
		if name == "" {
			name = "arg"
		}

		typeName, typeLimits := translate(GetDBType(v.Infos, arg).ConfirmedDBType())
		bindArgs[i] = drivers.QueryArg{
			Col: drivers.QueryCol{
				Name:       name,
				Nullable:   internal.Pointer(arg.Type.Nullable()),
				TypeName:   typeName,
				TypeLimits: typeLimits,
			}.Merge(parser.ParseQueryColumnConfig(comment(arg.Node))),
			Positions:     [][2]int{arg.EditedPosition},
			CanBeMultiple: arg.CanBeMultiple,
		}
	}
	bindArgs = slices.DeleteFunc(bindArgs, func(q drivers.QueryArg) bool {
		return len(q.Positions) == 0
	})

	for groupIndex, groupNodeKey := range v.Groups {
		group, ok := v.Infos[groupNodeKey]
		if !ok {
			continue
		}

		if group.Node.GetStart().GetStart() < start || group.Node.GetStop().GetStop() > stop {
			continue
		}

		name := v.Names[groupNodeKey]
		if name == "" {
			name = "group"
		}

		typeName, typeLimits := translate(GetDBType(v.Infos, group).ConfirmedDBType())
		groupArgs[groupIndex] = drivers.QueryArg{
			Col: drivers.QueryCol{
				Name:       name,
				Nullable:   internal.Pointer(group.Type.Nullable()),
				TypeName:   typeName,
				TypeLimits: typeLimits,
			}.Merge(parser.ParseQueryColumnConfig(comment(group.Node))),
			Positions:     [][2]int{group.EditedPosition},
			CanBeMultiple: group.CanBeMultiple,
		}
	}

	groupArgs = slices.DeleteFunc(groupArgs, func(q drivers.QueryArg) bool {
		return len(q.Positions) == 0
	})

	return parser.GetArgs(bindArgs, groupArgs)
}
