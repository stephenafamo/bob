package antlrhelpers

import (
	"slices"

	"github.com/antlr4-go/antlr/v4"
	"github.com/stephenafamo/bob/gen/drivers"
)

type Node interface {
	GetStart() antlr.Token
	GetStop() antlr.Token
	GetParent() antlr.Tree
	GetText() string
	GetParser() antlr.Parser
}

type NodeKey struct {
	start int
	stop  int
}

func Key(ctx Node) NodeKey {
	return NodeKey{
		start: ctx.GetStart().GetStart(),
		stop:  ctx.GetStop().GetStop(),
	}
}

type NodeInfo struct {
	Node                 Node
	ExprDescription      string
	Type                 NodeTypes
	ExprRef              Node
	IgnoreRefNullability bool

	// Go Info
	ArgKey         string // Positional or named arg in the query
	IsGroup        bool
	EditedPosition [2]int
	CanBeMultiple  bool
	Config         drivers.QueryCol
}

func GetDBType(exprs map[NodeKey]NodeInfo, e NodeInfo) NodeTypes {
	DBType := e.Type
	ignoreRefNullability := false

	keys := make(map[NodeKey]struct{})

	for DBType == nil && e.ExprRef != nil {
		key := Key(e.ExprRef)
		if _, ok := keys[key]; ok {
			break
		}

		e = exprs[key]
		DBType = e.Type
		ignoreRefNullability = e.IgnoreRefNullability

		keys[key] = struct{}{}
	}

	if ignoreRefNullability {
		DBType = slices.Clone(DBType)
		for i := range DBType {
			DBType[i].NullableF = nil
		}
	}

	return DBType
}
