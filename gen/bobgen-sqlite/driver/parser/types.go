package parser

import (
	"fmt"
	"slices"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/gen/drivers"
	"github.com/stephenafamo/bob/internal"
	sqliteparser "github.com/stephenafamo/sqlparser/sqlite"
)

type (
	tables     = drivers.Tables[any, IndexExtra]
	IndexExtra = struct {
		Partial bool `json:"partial"`
	}
)

type querySources []querySource

type querySource struct {
	schema  string
	name    string
	columns returns
	cte     bool
}

type stmtInfo struct {
	stmt      sqliteparser.ISql_stmtContext
	queryType bob.QueryType
	comment   string
	columns   returns
	editRules []internal.EditRule
	mods      *strings.Builder
	imports   [][]string
}

type returns []returnColumn

func (r returns) Print() string {
	s := &strings.Builder{}
	fmt.Fprintf(s, "%20s | %-25s | %s\n", "Name", "Type", "Options")
	fmt.Fprintln(s, strings.Repeat("-", 100))

	for _, col := range r {
		fmt.Fprintf(s, "%20s | %-25s | %s\n", col.name, col.typ, col.options)
	}

	return s.String()
}

type returnColumn struct {
	name    string
	typ     exprTypes
	options string // remove later
	config  drivers.QueryCol
}

type exprInfo struct {
	expr                 sqliteparser.IExprContext
	ExprDescription      string
	Type                 exprTypes
	ExprRef              sqliteparser.IExprContext
	IgnoreRefNullability bool

	// Go Info
	queryArgKey    string // Positional or named arg in the query
	isGroup        bool
	EditedPosition [2]int
	CanBeMultiple  bool
	options        string // remove later
	config         drivers.QueryCol
}

type exprName struct {
	names     func() []string
	childRefs map[nodeKey]exprChildNameRef
}

type exprChildNameRef func() (prefix, suffix []string)

type node interface {
	GetStart() antlr.Token
	GetStop() antlr.Token
	GetRuleIndex() int
	GetParent() antlr.Tree
	GetText() string
}

type nodeKey struct {
	start int
	stop  int
	rule  int
}

func key(ctx node) nodeKey {
	return nodeKey{
		start: ctx.GetStart().GetStart(),
		stop:  ctx.GetStop().GetStop(),
		rule:  ctx.GetRuleIndex(),
	}
}

type exprTypes []exprType

func (e exprTypes) Type(db tables) string {
	type keyCol = struct{ Key, Column string }
	refs := make([]keyCol, 0, len(e))
	for _, typ := range e {
		for _, ref := range typ.refs {
			refs = append(refs, keyCol{
				Key:    ref.key(),
				Column: ref.column,
			})
		}
	}

	if len(refs) == 0 {
		return TranslateColumnType(e.ConfirmedAffinity())
	}

	refTyp := db.GetColumn(refs[0].Key, refs[0].Column).Type

	for _, r := range refs[1:] {
		if refTyp != db.GetColumn(r.Key, r.Column).Type {
			return TranslateColumnType(e.ConfirmedAffinity())
		}
	}

	return refTyp
}

func (e exprTypes) ConfirmedAffinity() string {
	if len(e) == 0 {
		return ""
	}

	affinity := e[0].affinity

	for _, t := range e[1:] {
		if t.affinity != affinity {
			return ""
		}
	}

	return affinity
}

func (e exprTypes) Nullable() bool {
	for _, t := range e {
		if t.nullable() {
			return true
		}
	}

	return false
}

func (e exprTypes) List(t tables) []string {
	m := make([]string, len(e))
	for i, expr := range e {
		m[i] = expr.String()
	}

	return m
}

func (e exprTypes) String() string {
	m := make([]string, len(e))
	for i, expr := range e {
		m[i] = expr.String()
	}
	return strings.Join(m, ", ")
}

type ref struct {
	schema, table, column string
}

func (r ref) key() string {
	if r.schema == "" {
		return r.table
	}
	return r.schema + "." + r.table
}

func (r ref) String() string {
	if r.schema == "" {
		return fmt.Sprintf("%s.%s", r.table, r.column)
	}
	return fmt.Sprintf("%s.%s.%s", r.schema, r.table, r.column)
}

type identifiable interface {
	Identifier() sqliteparser.IIdentifierContext
}

func getName(i identifiable) string {
	if i == nil {
		return ""
	}
	ctx := i.Identifier()
	for ctx.OPEN_PAR() != nil {
		ctx = ctx.Identifier()
	}

	txt := ctx.GetText()
	if strings.ContainsAny(string(txt[0]), "\"`[") {
		return txt[1 : len(txt)-1]
	}

	return txt
}

func makeRef(sources querySources, ctx *sqliteparser.Expr_qualified_column_nameContext) exprTypes {
	schema := getName(ctx.Schema_name())
	table := getName(ctx.Table_name())
	column := getName(ctx.Column_name())
	if schema == "main" {
		schema = ""
	}

	for _, source := range slices.Backward(sources) {
		if table != "" && (schema != source.schema || table != source.name) {
			continue
		}

		for _, col := range source.columns {
			if col.name != column {
				continue
			}

			return col.typ
		}
	}

	// fmt.Printf("could not find column name: %q.%q.%q in %#v\n", schema, table, column, sources)
	return nil
}

func typeFromRef(db tables, schema, table, column string) exprType {
	if schema == "" && table == "" {
		// Find first table with matching column
		for _, table := range db {
			for _, col := range table.Columns {
				if col.Name == column {
					return exprType{
						affinity:  getAffinity(col.DBType),
						nullableF: func() bool { return col.Nullable },
						typeName:  []string{col.DBType},
						refs:      []ref{{table.Schema, table.Name, column}},
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

	return exprType{
		affinity:  getAffinity(col.DBType),
		nullableF: func() bool { return col.Nullable },
		typeName:  []string{col.DBType},
		refs:      []ref{{schema, table, column}},
	}
}

func knownType(t string, nullable func() bool) exprType {
	return exprType{
		affinity:  getAffinity(t),
		nullableF: nullable,
		typeName:  []string{t},
	}
}

type exprType struct {
	affinity  string
	nullableF func() bool
	typeName  []string
	refs      []ref
}

func (e exprType) nullable() bool {
	if e.nullableF != nil {
		return e.nullableF()
	}

	return false
}

func (e exprType) Merge(e2 exprType) (exprType, bool) {
	switch {
	case e.nullableF != nil && e2.nullableF != nil:
		current := e.nullableF()
		e.nullableF = func() bool {
			return current || e2.nullableF()
		}

	case e2.nullableF != nil:
		e.nullableF = e2.nullableF
	}

	e.typeName = append(e.typeName, e2.typeName...)
	e.refs = append(e.refs, e2.refs...)

	if e2.affinity == "" {
		return e, true
	}

	if e.affinity == "" {
		e.affinity = e2.affinity
		return e, true
	}

	return e, e.affinity == e2.affinity
}

func (e exprType) String() string {
	if e.nullable() {
		return fmt.Sprintf("%s NULLABLE", e.typString())
	}

	return fmt.Sprintf("%s NOT NULL", e.typString())
}

func (e exprType) typString() string {
	if len(e.refs) == 1 {
		return e.refs[0].String()
	}

	if len(e.refs) > 1 {
		return fmt.Sprintf("%s +%d", e.refs[0].String(), len(e.refs)-1)
	}

	return e.affinity
}

// https://www.sqlite.org/datatype3.html
//
//nolint:misspell
func getAffinity(t string) string {
	if t == "" {
		return ""
	}

	if strings.Contains(t, "INT") {
		return "INTEGER"
	}

	if strings.Contains(t, "CHAR") || strings.Contains(t, "CLOB") || strings.Contains(t, "TEXT") {
		return "TEXT"
	}

	if strings.Contains(t, "BLOB") {
		return "BLOB"
	}

	if strings.Contains(t, "REAL") || strings.Contains(t, "FLOA") || strings.Contains(t, "DOUB") {
		return "REAL"
	}

	return "NUMERIC"
}

func nullable() bool {
	return true
}

func notNullable() bool {
	return false
}

func anyNullable(fs ...func() bool) func() bool {
	return func() bool {
		for _, f := range fs {
			if f() {
				return true
			}
		}

		return false
	}
}

func allNullable(fs ...func() bool) func() bool {
	return func() bool {
		for _, f := range fs {
			if !f() {
				return false
			}
		}

		return true
	}
}

func neverNullable(...func() bool) func() bool {
	return notNullable
}

type functions map[string]function

type function struct {
	requiredArgs         int
	variadic             bool
	args                 []string
	returnType           string
	calcReturnType       func(...string) string // If present, will be used to calculate the return type
	shouldArgsBeNullable bool
	calcNullable         func(...func() bool) func() bool // will be provided with the nullability of the args
}

func (f function) argType(i int) string {
	if i >= len(f.args) {
		return f.args[len(f.args)-1]
	}

	return f.args[i]
}
