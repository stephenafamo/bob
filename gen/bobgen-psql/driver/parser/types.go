package parser

import (
	"fmt"
	"io"
	"slices"
	"strconv"

	pg "github.com/pganalyze/pg_query_go/v6"
	"github.com/stephenafamo/bob/gen/drivers"
)

type tables = drivers.Tables[any, IndexExtra]

type IndexExtra = struct {
	NullsFirst    []bool   `json:"nulls_first"` // same length as Columns
	NullsDistinct bool     `json:"nulls_not_distinct"`
	Where         string   `json:"where_clause"`
	Include       []string `json:"include"`
}

type position [2]int32

func (i position) String() string {
	return fmt.Sprintf("%d:%d", i[0], i[1])
}

func (i position) LitterDump(w io.Writer) {
	_, _ = fmt.Fprintf(w, "(%d, %d)", i[0], i[1])
}

type joinedInfo struct {
	node     *pg.Node
	info     nodeInfo
	joinType pg.JoinType
}

func (j joinedInfo) LitterDump(w io.Writer) {
	_, _ = fmt.Fprintf(w, "(%T, %v)", j.node.Node, j.info.children)
}

type nullable interface {
	IsNull(map[position]string, map[position]nullable, []queryResult) bool
}

func makeAnyNullable(n ...nullable) nullable {
	return anyNullable(n)
}

type anyNullable []nullable

func (a anyNullable) IsNull(names map[position]string, index map[position]nullable, sources []queryResult) bool {
	for _, n := range a {
		if n.IsNull(names, index, sources) {
			return true
		}
	}

	return false
}

type alwaysNullable struct{}

func (a alwaysNullable) IsNull(map[position]string, map[position]nullable, []queryResult) bool {
	return true
}

func (i alwaysNullable) LitterDump(w io.Writer) {
	_, _ = fmt.Fprintf(w, "(TRUE)")
}

type columnNullable struct {
	def  *pg.ColumnRef
	info nodeInfo
}

func (i columnNullable) LitterDump(w io.Writer) {
	fmt.Fprintf(w, "(%v)", i.info)
}

func (c columnNullable) IsNull(names map[position]string, index map[position]nullable, sources []queryResult) bool {
	if len(sources) == 0 {
		return false
	}

	if len(c.def.Fields) < 1 || len(c.def.Fields) > 3 {
		return false
	}

	var schema, table, column string

	fieldsInfo := c.info.children["Fields"]
	for i := range c.def.Fields {
		name := names[fieldsInfo.children[strconv.Itoa(i)].position()]
		switch {
		case i == len(c.def.Fields)-1:
			column = name
		case i == len(c.def.Fields)-2:
			table = name
		case i == len(c.def.Fields)-3:
			schema = name
		}
	}

	for _, source := range slices.Backward(sources) {
		if table == "" && source.mustBeQualified {
			continue
		}
		if table != "" && (schema != source.schema || table != source.name) {
			continue
		}

		for _, col := range source.columns {
			if col.name != column {
				continue
			}

			return col.nullable
		}
	}

	return false
}

type argPos struct {
	// The original position of the arg in the input string
	original position
	// The position of the arg in the edited string
	edited [2]int
}

func (a argPos) LitterDump(w io.Writer) {
	fmt.Fprintf(w, "(%v, %v)", a.original, a.edited)
}
