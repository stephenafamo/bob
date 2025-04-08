package parser

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/stephenafamo/bob/gen/drivers"
	"github.com/stephenafamo/bob/internal"
	sqliteparser "github.com/stephenafamo/sqlparser/sqlite"
)

//---------------------------------------------------------------------------
// Print helpers
//---------------------------------------------------------------------------

func (v *visitor) printExprs(input string, start, stop int, exprs ...exprInfo) string {
	s := &strings.Builder{}
	fmt.Fprintf(
		s,
		"%20s | %5s | %-25s | %-25s | %-15s | %-15s | %s\n",
		"TYPE", "Null?", "DBType", "Name", "Position", "Text", "Options",
	)

	fmt.Fprintln(s, strings.Repeat("-", 140))

	for _, expr := range exprs {
		if expr.expr.GetStart().GetStart() < start || expr.expr.GetStop().GetStop() > stop {
			continue
		}

		types := strings.Split(expr.ExprDescription, ",")
		dbType := v.getDBType(expr)
		fmt.Fprintf(
			s,
			"%20s | %5t | %-25s | %-25s | %-15s | %-15s | %s\n",
			types[0], dbType.Nullable(), v.getDBType(expr),
			v.getNameString(expr.expr), fmt.Sprint(expr.EditedPosition),
			input[expr.expr.GetStart().GetStart():expr.expr.GetStop().GetStop()+1],
			expr.options+"----"+expr.queryArgKey,
		)
		for _, typ := range types[1:] {
			fmt.Fprintf(s, "%20s | %5s | %-25s | %-25s | %-15s | %-15s | %s\n", typ, "", "", "", "", "", "")
		}
		fmt.Fprintf(s, "%20s | %5s | %-25s | %-25s | %-15s | %-15s | %s\n", "", "", "", "", "", "", "")
	}

	return s.String()
}

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

func (v *visitor) getArgs(start, stop int) []exprInfo {
	exprs := make([]exprInfo, 0, len(v.exprs))
	groups := make([]exprInfo, 0, len(v.exprs))

	// Only get bind expressions
	for _, expr := range v.exprs {
		if expr.expr.GetStart().GetStart() < start || expr.expr.GetStop().GetStop() > stop {
			continue
		}

		if expr.isGroup {
			groups = append(groups, expr)
		}

		if _, ok := expr.expr.(*sqliteparser.Expr_bindContext); !ok {
			continue
		}

		exprs = append(exprs, expr)
	}

GroupLoop:
	for _, group := range groups {
		for _, expr := range exprs {
			if key(group.expr) == key(expr.expr) {
				continue
			}

			if expr.expr.GetStart().GetStart() >= group.expr.GetStart().GetStart() &&
				expr.expr.GetStop().GetStop() <= group.expr.GetStop().GetStop() {
				exprs = append(exprs, group)
				continue GroupLoop
			}
		}
	}

	// We want to sort the exprs by the order they appear in the input
	slices.SortFunc(exprs, func(i, j exprInfo) int {
		iKey := key(i.expr)
		jKey := key(j.expr)

		if iKey.start != jKey.start {
			return iKey.start - jKey.start
		}

		if iKey.stop != jKey.stop {
			return jKey.stop - iKey.stop
		}

		return iKey.rule - jKey.rule
	})

	for i, expr := range exprs {
		exprs[i].options = v.getCommentToRight(expr.expr)
		// Merge in case the name is configured in the bind
		exprs[i].config = exprs[i].config.Merge(
			drivers.ParseQueryColumnConfig(v.getCommentToRight(expr.expr)),
		)
	}

	return exprs
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

func (v *visitor) getCommentToLeft(ctx interface {
	GetParser() antlr.Parser
	GetStart() antlr.Token
	GetSourceInterval() antlr.Interval
},
) string {
	stream, isCommon := ctx.GetParser().GetTokenStream().(*antlr.CommonTokenStream)
	if !isCommon {
		return ""
	}

	tokenIndex := ctx.GetStart().GetTokenIndex()
	hiddenTokens := stream.GetHiddenTokensToLeft(tokenIndex, 1)
	for _, token := range hiddenTokens {
		if token.GetTokenType() == sqliteparser.SQLiteParserSINGLE_LINE_COMMENT {
			return strings.TrimSpace(token.GetText()[2:])
		}
	}

	return ""
}

func (v *visitor) getCommentToRight(ctx interface {
	GetParser() antlr.Parser
	GetStop() antlr.Token
	GetSourceInterval() antlr.Interval
},
) string {
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
