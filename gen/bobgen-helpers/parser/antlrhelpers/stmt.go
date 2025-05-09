package antlrhelpers

import (
	"fmt"
	"strings"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/gen/drivers"
	"github.com/stephenafamo/bob/internal"
)

type StmtInfo struct {
	Node      Node
	QueryType bob.QueryType
	Comment   string
	Columns   []ReturnColumn
	EditRules []internal.EditRule
	Mods      *strings.Builder
	Imports   [][]string
}

type ReturnColumn struct {
	Name   string
	Type   NodeTypes
	Config drivers.QueryCol
}

type QuerySources = []QuerySource

type QuerySource struct {
	Schema  string
	Name    string
	Columns []ReturnColumn
	CTE     bool
}

func ExpandQuotedSource(buf *strings.Builder, source QuerySource) {
	for i, col := range source.Columns {
		if i > 0 {
			buf.WriteString(", ")
		}
		if source.Schema != "" {
			fmt.Fprintf(buf, "%q.%q.%q", source.Schema, source.Name, col.Name)
		} else {
			fmt.Fprintf(buf, "%q.%q", source.Name, col.Name)
		}
	}
}
