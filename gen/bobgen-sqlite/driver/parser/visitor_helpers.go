package parser

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/stephenafamo/bob/gen/bobgen-helpers/parser"
	"github.com/stephenafamo/bob/internal"
	sqliteparser "github.com/stephenafamo/sqlparser/sqlite"
)

//---------------------------------------------------------------------------
// Type helpers
//---------------------------------------------------------------------------

func matchTypes(existing, newTypes exprTypes) exprTypes {
	matchingDBTypes := exprTypes{}
Outer:
	for _, t := range newTypes {
		for _, ct := range existing {
			merged, ok := t.Merge(ct)
			if ok {
				matchingDBTypes = append(matchingDBTypes, merged)
				continue Outer
			}
		}
	}

	return matchingDBTypes
}

func (v *visitor) getDBType(e exprInfo) exprTypes {
	DBType := e.Type
	ignoreRefNullability := false

	keys := make(map[nodeKey]struct{})

	for DBType == nil && e.ExprRef != nil {
		key := key(e.ExprRef)
		if _, ok := keys[key]; ok {
			break
		}

		e = v.exprs[key]
		DBType = e.Type
		ignoreRefNullability = e.IgnoreRefNullability

		keys[key] = struct{}{}
	}

	if ignoreRefNullability {
		DBType = slices.Clone(DBType)
		for i := range DBType {
			DBType[i].nullableF = nil
		}
	}

	return DBType
}

func (v *visitor) updateExprInfo(info exprInfo) {
	key := key(info.expr)

	currentExpr, ok := v.exprs[key]
	if !ok {
		v.exprs[key] = info
		return
	}

	currentExpr.expr = info.expr
	currentExpr.isGroup = currentExpr.isGroup || info.isGroup
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
		v.exprs[key] = currentExpr
		return
	}

	if currentExpr.Type == nil {
		currentExpr.Type = info.Type
		v.exprs[key] = currentExpr
		return
	}

	matchingDBTypes := matchTypes(currentExpr.Type, info.Type)
	if len(matchingDBTypes) == 0 {
		panic(fmt.Sprintf(
			"No matching DBType found for %s: \n%v\n%v",
			info.expr.GetText(),
			currentExpr.Type.List(v.db),
			info.Type.List(v.db),
		))
	}

	currentExpr.Type = matchingDBTypes
	v.exprs[key] = currentExpr
}

// ---------------------------------------------------------------------------
// Name helpers
// ---------------------------------------------------------------------------

func (v *visitor) getNameString(expr node) string {
	return strings.Join(v.getNames(expr), "_")
}

func (v *visitor) getExprName(expr node) []string {
	if expr == nil {
		return nil
	}

	exprKey := key(expr)
	name := v.names[exprKey]

	if name.names != nil {
		return name.names()
	}

	return nil
}

func (v *visitor) getNames(expr node) []string {
	exprKey := key(expr)
	name := v.getExprName(expr)

	for parent, ok := expr.GetParent().(node); ok && parent != nil; parent, ok = parent.GetParent().(node) {
		if len(name) > 0 {
			return internal.FilterNonZero(name)
		}

		parentName := v.names[key(parent)]

		if ref, ok := parentName.childRefs[exprKey]; ok {
			prefix, suffix := ref()
			name = append(prefix, name...)
			name = append(name, suffix...)
		}

		exprKey = key(parent)
	}

	return internal.FilterNonZero(name)
}

func (v *visitor) addName(ctx node, name exprName) {
	selfKey := key(ctx)
	self := v.names[selfKey]

	if name.names != nil {
		self.names = name.names
	}

	if self.childRefs == nil {
		self.childRefs = map[nodeKey]exprChildNameRef{}
	}

	maps.Copy(self.childRefs, name.childRefs)
	v.names[selfKey] = self
}

func (v *visitor) addRawName(ctx sqliteparser.IExprContext, names ...string) {
	v.addName(ctx, exprName{
		names: func() []string {
			return names
		},
	})
}

func (v *visitor) addLRName(ctx interface {
	sqliteparser.IExprContext
	GetLhs() sqliteparser.IExprContext
	GetRhs() sqliteparser.IExprContext
}, op string,
) {
	lhs := ctx.GetLhs()
	rhs := ctx.GetRhs()

	v.addName(ctx, exprName{
		names: func() []string {
			names := append(v.getExprName(lhs), op)
			return append(names, v.getExprName(rhs)...)
		},
		childRefs: map[nodeKey]exprChildNameRef{
			key(lhs): func() ([]string, []string) { return nil, append([]string{op}, v.getExprName(rhs)...) },
			key(rhs): func() ([]string, []string) { return append(v.getExprName(lhs), op), nil },
		},
	})
}

// ---------------------------------------------------------------------------
// Comment getter
// ---------------------------------------------------------------------------

func (v *visitor) getCommentToLeft(ctx node) string {
	stream, isCommon := ctx.GetParser().GetTokenStream().(*antlr.CommonTokenStream)
	if !isCommon {
		return ""
	}

	tokenIndex := ctx.GetStart().GetTokenIndex()
	hiddenTokens := stream.GetHiddenTokensToLeft(tokenIndex, 1)
	for _, token := range slices.Backward(hiddenTokens) {
		if token.GetTokenType() == sqliteparser.SQLiteParserSINGLE_LINE_COMMENT {
			return strings.TrimSpace(token.GetText()[2:])
		}
	}

	return ""
}

func (v *visitor) getCommentToRight(ctx node) string {
	stream, isCommon := ctx.GetParser().GetTokenStream().(*antlr.CommonTokenStream)
	if !isCommon {
		return ""
	}

	tokenIndex := ctx.GetStop().GetTokenIndex()
	hiddenTokens := stream.GetHiddenTokensToRight(tokenIndex, 1)
	for _, token := range hiddenTokens {
		if token.GetTokenType() == sqliteparser.SQLiteParserMULTILINE_COMMENT {
			txt := token.GetText()
			return strings.TrimSpace(txt[2 : len(txt)-2])
		}
	}

	return ""
}

// ---------------------------------------------------------------------------
// Edit Rule helpers
// ---------------------------------------------------------------------------
func (v *visitor) quoteIdentifiable(ctx interface {
	Identifier() sqliteparser.IIdentifierContext
},
) {
	if ctx == nil {
		return
	}

	v.quoteIdentifier(ctx.Identifier())
}

func (v *visitor) quoteIdentifier(ctx sqliteparser.IIdentifierContext) {
	if ctx == nil {
		return
	}

	switch ctx.GetParent().(type) {
	case sqliteparser.ISimple_funcContext,
		sqliteparser.IAggregate_funcContext,
		sqliteparser.IWindow_funcContext,
		sqliteparser.ITable_function_nameContext:
		return
	}

	idContext := ctx

	for idContext.OPEN_PAR() != nil {
		idContext = idContext.Identifier()
	}

	txt := ctx.GetText()
	if strings.ContainsAny(string(txt[0]), "\"`[") {
		txt = txt[1 : len(txt)-1]
	}

	v.stmtRules = append(v.stmtRules, internal.Replace(ctx.GetStart().GetStart(), ctx.GetStop().GetStop(), fmt.Sprintf("%q", txt)))
}

func expandQuotedSource(buf *strings.Builder, source querySource) {
	for i, col := range source.columns {
		if i > 0 {
			buf.WriteString(", ")
		}
		if source.schema != "" {
			fmt.Fprintf(buf, "%q.%q.%q", source.schema, source.name, col.name)
		} else {
			fmt.Fprintf(buf, "%q.%q", source.name, col.name)
		}
	}
}

func (v *visitor) getSourceFromTable(ctx interface {
	Schema_name() sqliteparser.ISchema_nameContext
	Table_name() sqliteparser.ITable_nameContext
	Table_alias() sqliteparser.ITable_aliasContext
},
) querySource {
	schema := getName(ctx.Schema_name())
	v.quoteIdentifiable(ctx.Schema_name())

	tableName := getName(ctx.Table_name())
	v.quoteIdentifiable(ctx.Table_name())

	alias := getName(ctx.Table_alias())
	v.quoteIdentifiable(ctx.Table_alias())

	hasAlias := alias != ""
	if alias == "" {
		alias = tableName
	}

	// First check the sources to see if the table exists
	// do this ONLY if no schema is provided
	if schema == "" {
		for _, source := range v.sources {
			if source.name == tableName {
				return querySource{
					name:    alias,
					columns: source.columns,
				}
			}
		}
	}

	for _, table := range v.db {
		if table.Name != tableName {
			continue
		}

		switch {
		case table.Schema == schema: // schema matches
		case table.Schema == "" && schema == "main": // schema is shared
		default:
			continue
		}

		source := querySource{
			name:    alias,
			columns: make([]returnColumn, len(table.Columns)),
		}
		if !hasAlias {
			source.schema = schema
		}
		for i, col := range table.Columns {
			source.columns[i] = returnColumn{
				name: col.Name,
				typ:  exprTypes{typeFromRef(v.db, table.Schema, table.Name, col.Name)},
			}
		}
		return source
	}

	v.err = fmt.Errorf("table not found: %s", tableName)
	return querySource{}
}

func (v *visitor) sourceFromColumns(columns []sqliteparser.IResult_columnContext) querySource {
	// Get the return columns
	var returnSource querySource

	for _, resultColumn := range columns {
		switch {
		case resultColumn.STAR() != nil: // Has a STAR: * OR table_name.*
			table := getName(resultColumn.Table_name())
			hasTable := table != "" // the result column is table_name.*

			start := resultColumn.GetStart().GetStart()
			stop := resultColumn.GetStop().GetStop()
			v.stmtRules = append(v.stmtRules, internal.Delete(start, stop))

			buf := &strings.Builder{}
			var i int
			for _, source := range v.sources {
				if source.cte {
					continue
				}
				if hasTable && source.name != table {
					continue
				}

				returnSource.columns = append(returnSource.columns, source.columns...)

				if i > 0 {
					buf.WriteString(", ")
				}
				expandQuotedSource(buf, source)
				i++
			}
			v.stmtRules = append(v.stmtRules, internal.Insert(start, buf.String()))

		case resultColumn.Expr() != nil: // expr (AS_? alias)?
			expr := resultColumn.Expr()
			alias := getName(resultColumn.Alias())
			if alias == "" {
				if expr, ok := expr.(*sqliteparser.Expr_qualified_column_nameContext); ok {
					alias = getName(expr.Column_name())
				}
			}

			returnSource.columns = append(returnSource.columns, returnColumn{
				name:    alias,
				options: v.getCommentToRight(expr),
				config:  parser.ParseQueryColumnConfig(v.getCommentToRight(expr)),
				typ:     v.exprs[key(resultColumn.Expr())].Type,
			})
		}
	}

	return returnSource
}

// ---------------------------------------------------------------------------
// Function helpers
// ---------------------------------------------------------------------------

func (v *visitor) getFunctionType(funcName string, argTypes []string) (function, error) {
	f, ok := v.functions[funcName]
	if !ok {
		return function{}, fmt.Errorf("function %q not found", funcName)
	}

	if len(argTypes) < f.requiredArgs {
		return function{}, fmt.Errorf("too few arguments for function %q, %d/%d", funcName, len(argTypes), f.requiredArgs)
	}

	if !f.variadic && len(argTypes) > len(f.args) {
		return function{}, fmt.Errorf("too many arguments for function %q, %d/%d", funcName, len(argTypes), len(f.args))
	}

	for i, arg := range argTypes {
		// We don't know the type of the given argument
		if arg == "" {
			continue
		}

		argID := i
		if f.variadic && i >= len(f.args) {
			argID = len(f.args) - 1
		}

		// means the func can take any type in this position
		if f.args[argID] == "" {
			continue
		}

		if !strings.EqualFold(f.args[argID], arg) {
			return function{}, fmt.Errorf("function %q(%s) expects %s at position %d, got %s", funcName, strings.Join(argTypes, ", "), f.args[argID], i+1, arg)
		}
	}

	if f.calcReturnType != nil {
		f.returnType = f.calcReturnType(argTypes...)
	}

	return f, nil
}
