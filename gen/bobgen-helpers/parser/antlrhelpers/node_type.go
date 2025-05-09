package antlrhelpers

import (
	"fmt"

	"github.com/stephenafamo/bob/gen/drivers"
)

type NodeTypes []NodeType

func (e NodeTypes) ConfirmedDBType() string {
	if len(e) == 0 {
		return ""
	}

	fristType := e[0].DBType

	for _, t := range e[1:] {
		if t.DBType != fristType {
			return ""
		}
	}

	return fristType
}

func (e NodeTypes) Nullable() bool {
	for _, t := range e {
		if t.Nullable() {
			return true
		}
	}

	return false
}

// Get the common types between the two sets of node types.
func (existing NodeTypes) Match(newTypes NodeTypes) NodeTypes {
	matchingDBTypes := NodeTypes{}
Outer:
	for _, t := range newTypes {
		for _, ct := range existing {
			merged, ok := mergeTypes(t, ct)
			if ok {
				matchingDBTypes = append(matchingDBTypes, merged)
				continue Outer
			}
		}
	}

	return matchingDBTypes
}

func (e NodeTypes) List() []string {
	m := make([]string, len(e))
	for i, expr := range e {
		m[i] = expr.String()
	}

	return m
}

type NodeType struct {
	DBType    string
	NullableF func() bool
}

func (e NodeType) Nullable() bool {
	if e.NullableF != nil {
		return e.NullableF()
	}

	return false
}

func mergeTypes(e, e2 NodeType) (NodeType, bool) {
	switch {
	case e.NullableF != nil && e2.NullableF != nil:
		current := e.NullableF()
		e.NullableF = func() bool {
			return current || e2.NullableF()
		}

	case e2.NullableF != nil:
		e.NullableF = e2.NullableF
	}

	if e2.DBType == "" {
		return e, true
	}

	if e.DBType == "" {
		e.DBType = e2.DBType
		return e, true
	}

	return e, e.DBType == e2.DBType
}

func (e NodeType) String() string {
	if e.Nullable() {
		return fmt.Sprintf("%s NULLABLE", e.DBType)
	}

	return fmt.Sprintf("%s NOT NULL", e.DBType)
}

func GetColumnType[C, I any](db drivers.Tables[C, I], schema, table, column string) NodeType {
	if schema == "" && table == "" {
		// Find first table with matching column
		for _, table := range db {
			for _, col := range table.Columns {
				if col.Name == column {
					return NodeType{
						DBType:    col.DBType,
						NullableF: func() bool { return col.Nullable },
					}
				}
			}
		}
		panic(fmt.Sprintf("could not find column name: %q in %#v", column, db))
	}

	key := fmt.Sprintf("%s.%s", schema, table)
	if schema == "" {
		key = table
	}

	col := db.GetColumn(key, column)

	return NodeType{
		DBType:    col.DBType,
		NullableF: func() bool { return col.Nullable },
	}
}
