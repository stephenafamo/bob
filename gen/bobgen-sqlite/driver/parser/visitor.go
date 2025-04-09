package parser

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"sync/atomic"

	"github.com/antlr4-go/antlr/v4"
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/gen/drivers"
	"github.com/stephenafamo/bob/internal"
	sqliteparser "github.com/stephenafamo/sqlparser/sqlite"
)

var _ sqliteparser.SQLiteParserVisitor = &visitor{}

func NewVisitor(db tables) *visitor {
	return &visitor{
		db:        db,
		functions: defaultFunctions,
		exprs:     make(map[nodeKey]exprInfo),
		names:     make(map[nodeKey]exprName),
		atom:      &atomic.Int64{},
		mods:      &strings.Builder{},
	}
}

type visitor struct {
	err       error
	db        tables
	sources   querySources
	functions functions
	names     map[nodeKey]exprName
	exprs     map[nodeKey]exprInfo
	baseRules []internal.EditRule

	// Refresh these for each statement
	stmtRules []internal.EditRule
	atom      *atomic.Int64
	mods      *strings.Builder
	imports   [][]string
}

func (v *visitor) Visit(tree antlr.ParseTree) any { return tree.Accept(v) }

func (v *visitor) VisitChildren(ctx antlr.RuleNode) any {
	if v.err != nil {
		v.err = fmt.Errorf("visiting children: %w", v.err)
		return nil
	}

	for i, child := range ctx.GetChildren() {
		if tree, ok := child.(antlr.ParseTree); ok {
			tree.Accept(v)
		}

		if v.err != nil {
			v.err = fmt.Errorf("visiting child %d: %w", i, v.err)
			return nil
		}
	}

	return nil
}

func (v *visitor) VisitTerminal(ctx antlr.TerminalNode) any {
	token := ctx.GetSymbol()
	literals := sqliteparser.SQLiteLexerLexerStaticData.LiteralNames
	if token.GetTokenType() >= len(literals) {
		return nil
	}

	literal := literals[token.GetTokenType()]
	if len(literal) < 4 {
		return nil
	}
	v.stmtRules = append(v.stmtRules, internal.Replace(
		token.GetStart(),
		token.GetStop(),
		literal[1:len(literal)-1],
	))

	return nil
}

func (v *visitor) VisitErrorNode(_ antlr.ErrorNode) any { return nil }

func (v *visitor) VisitParse(ctx *sqliteparser.ParseContext) any {
	return ctx.Sql_stmt_list().Accept(v)
}

func (v *visitor) VisitSql_stmt_list(ctx *sqliteparser.Sql_stmt_listContext) any {
	stmts := ctx.AllSql_stmt()
	allresp := make([]stmtInfo, len(stmts))

	for i, stmt := range stmts {
		for _, child := range stmt.GetChildren() {
			if _, isTerminal := child.(antlr.TerminalNode); isTerminal {
				continue
			}

			v.stmtRules = slices.Clone(v.baseRules)
			v.atom = &atomic.Int64{}
			v.mods = &strings.Builder{}
			v.imports = nil

			resp := child.(antlr.ParseTree).Accept(v)
			if v.err != nil {
				v.err = fmt.Errorf("stmt %d: %w", i, v.err)
				return nil
			}

			info, ok := resp.(returns)
			if !ok {
				v.err = fmt.Errorf("stmt %d: could not columns, got %T", i, resp)
				return nil
			}

			allresp[i] = stmtInfo{
				stmt:      stmt,
				columns:   info,
				editRules: slices.Clone(v.stmtRules),
				comment:   v.getCommentToLeft(stmt),
				mods:      v.mods,
				imports:   slices.Clone(v.imports),
			}

			switch child.(type) {
			case *sqliteparser.Select_stmtContext:
				allresp[i].queryType = bob.QueryTypeSelect
			case *sqliteparser.Insert_stmtContext:
				allresp[i].queryType = bob.QueryTypeInsert
			case *sqliteparser.Update_stmtContext:
				allresp[i].queryType = bob.QueryTypeUpdate
			case *sqliteparser.Delete_stmtContext:
				allresp[i].queryType = bob.QueryTypeDelete
			}
		}
	}

	return allresp
}

func (v *visitor) VisitSql_stmt(ctx *sqliteparser.Sql_stmtContext) any {
	return v.VisitChildren(ctx)
}

func (v *visitor) VisitAlter_table_stmt(ctx *sqliteparser.Alter_table_stmtContext) any {
	return v.VisitChildren(ctx)
}

func (v *visitor) VisitAnalyze_stmt(ctx *sqliteparser.Analyze_stmtContext) any {
	return v.VisitChildren(ctx)
}

func (v *visitor) VisitAttach_stmt(ctx *sqliteparser.Attach_stmtContext) any {
	return v.VisitChildren(ctx)
}

func (v *visitor) VisitBegin_stmt(ctx *sqliteparser.Begin_stmtContext) any {
	return v.VisitChildren(ctx)
}

func (v *visitor) VisitCommit_stmt(ctx *sqliteparser.Commit_stmtContext) any {
	return v.VisitChildren(ctx)
}

func (v *visitor) VisitRollback_stmt(ctx *sqliteparser.Rollback_stmtContext) any {
	return v.VisitChildren(ctx)
}

func (v *visitor) VisitSavepoint_stmt(ctx *sqliteparser.Savepoint_stmtContext) any {
	return v.VisitChildren(ctx)
}

func (v *visitor) VisitRelease_stmt(ctx *sqliteparser.Release_stmtContext) any {
	return v.VisitChildren(ctx)
}

func (v *visitor) VisitCreate_index_stmt(ctx *sqliteparser.Create_index_stmtContext) any {
	return v.VisitChildren(ctx)
}

func (v *visitor) VisitIndexed_column(ctx *sqliteparser.Indexed_columnContext) any {
	return v.VisitChildren(ctx)
}

func (v *visitor) VisitCreate_table_stmt(ctx *sqliteparser.Create_table_stmtContext) any {
	return v.VisitChildren(ctx)
}

func (v *visitor) VisitColumn_def(ctx *sqliteparser.Column_defContext) any {
	return v.VisitChildren(ctx)
}

func (v *visitor) VisitType_name(ctx *sqliteparser.Type_nameContext) any {
	return v.VisitChildren(ctx)
}

func (v *visitor) VisitColumn_constraint(ctx *sqliteparser.Column_constraintContext) any {
	return v.VisitChildren(ctx)
}

func (v *visitor) VisitSigned_number(ctx *sqliteparser.Signed_numberContext) any {
	return v.VisitChildren(ctx)
}

func (v *visitor) VisitTable_constraint(ctx *sqliteparser.Table_constraintContext) any {
	return v.VisitChildren(ctx)
}

func (v *visitor) VisitForeign_key_clause(ctx *sqliteparser.Foreign_key_clauseContext) any {
	return v.VisitChildren(ctx)
}

func (v *visitor) VisitConflict_clause(ctx *sqliteparser.Conflict_clauseContext) any {
	return v.VisitChildren(ctx)
}

func (v *visitor) VisitCreate_trigger_stmt(ctx *sqliteparser.Create_trigger_stmtContext) any {
	return v.VisitChildren(ctx)
}

func (v *visitor) VisitCreate_view_stmt(ctx *sqliteparser.Create_view_stmtContext) any {
	return v.VisitChildren(ctx)
}

func (v *visitor) VisitCreate_virtual_table_stmt(ctx *sqliteparser.Create_virtual_table_stmtContext) any {
	return v.VisitChildren(ctx)
}

func (v *visitor) VisitDelete_stmt(ctx *sqliteparser.Delete_stmtContext) any {
	return v.VisitChildren(ctx)
}

func (v *visitor) VisitDetach_stmt(ctx *sqliteparser.Detach_stmtContext) any {
	return v.VisitChildren(ctx)
}

func (v *visitor) VisitDrop_stmt(ctx *sqliteparser.Drop_stmtContext) any {
	return v.VisitChildren(ctx)
}

func (v *visitor) VisitExpr_case(ctx *sqliteparser.Expr_caseContext) any {
	return v.VisitChildren(ctx)
}

func (v *visitor) VisitExpr_arithmetic(ctx *sqliteparser.Expr_arithmeticContext) any {
	opName := ctx.GetParser().GetSymbolicNames()[ctx.GetOperator().GetTokenType()]
	opName = strings.ToLower(opName)
	v.addLRName(ctx, opName)

	v.VisitChildren(ctx)
	if v.err != nil {
		return nil
	}

	lhsType := v.exprs[key(ctx.GetLhs())].Type
	rhsType := v.exprs[key(ctx.GetRhs())].Type

	typ := []exprType{knownType("INTEGER", notNullable), knownType("REAL", notNullable)}

	switch {
	case len(lhsType) == 1 && len(rhsType) == 1:
		typ = []exprType{knownType("REAL", notNullable)}
		lhs := lhsType[0]
		rhs := rhsType[0]
		if lhs.affinity == "INTEGER" &&
			rhs.affinity == "INTEGER" {
			typ = []exprType{knownType("INTEGER", anyNullable(lhs.nullable, rhs.nullable))}
		}

	case len(lhsType) == 1 && len(rhsType) == 0:
		typ = []exprType{knownType("REAL", notNullable)}
		lhs := lhsType[0]
		if lhs.affinity == "INTEGER" {
			typ = []exprType{knownType("INTEGER", lhs.nullable)}
		}

	case len(lhsType) == 0 && len(rhsType) == 1:
		typ = []exprType{knownType("REAL", notNullable)}
		rhs := rhsType[0]
		if rhs.affinity == "INTEGER" {
			typ = []exprType{knownType("INTEGER", rhs.nullable)}
		}
	}

	v.updateExprInfo(exprInfo{
		expr:            ctx,
		ExprDescription: "Arithmetic",
		Type:            typ,
	})

	v.updateExprInfo(exprInfo{
		expr:            ctx.GetLhs(),
		ExprDescription: "Arithmetic LHS",
		Type:            typ,
	})

	v.updateExprInfo(exprInfo{
		expr:            ctx.GetRhs(),
		ExprDescription: "Arithmetic RHS",
		Type:            typ,
	})

	return nil
}

func (v *visitor) VisitExpr_json_extract_string(ctx *sqliteparser.Expr_json_extract_stringContext) any {
	v.VisitChildren(ctx)
	if v.err != nil {
		return nil
	}

	v.updateExprInfo(exprInfo{
		expr:            ctx,
		ExprDescription: "JSON->>",
		Type:            []exprType{knownType("", nullable)},
	})

	v.updateExprInfo(exprInfo{
		expr:            ctx.GetLhs(),
		ExprDescription: "JSON->> LHS",
		Type:            []exprType{knownType("JSON", notNullable)},
	})

	v.updateExprInfo(exprInfo{
		expr:            ctx.GetRhs(),
		ExprDescription: "JSON->> RHS",
		Type: []exprType{
			knownType("TEXT", notNullable),
			knownType("INTEGER", notNullable),
		},
	})

	return nil
}

func (v *visitor) VisitExpr_raise(ctx *sqliteparser.Expr_raiseContext) any {
	return v.VisitChildren(ctx)
}

func (v *visitor) VisitExpr_bool(ctx *sqliteparser.Expr_boolContext) any {
	v.VisitChildren(ctx)
	if v.err != nil {
		return nil
	}

	v.updateExprInfo(exprInfo{
		expr:            ctx,
		ExprDescription: "AND/OR",
		Type:            []exprType{knownType("BOOLEAN", notNullable)},
	})

	v.updateExprInfo(exprInfo{
		expr:            ctx.GetLhs(),
		ExprDescription: "AND/OR LHS",
		Type:            []exprType{knownType("BOOLEAN", nullable)},
	})

	v.updateExprInfo(exprInfo{
		expr:            ctx.GetRhs(),
		ExprDescription: "AND/OR RHS",
		Type:            []exprType{knownType("BOOLEAN", nullable)},
	})

	return nil
}

func (v *visitor) VisitExpr_is(ctx *sqliteparser.Expr_isContext) any {
	v.addLRName(ctx, "Is")

	v.VisitChildren(ctx)
	if v.err != nil {
		return nil
	}

	v.updateExprInfo(exprInfo{
		expr:            ctx,
		ExprDescription: "IS",
		Type:            []exprType{knownType("BOOLEAN", notNullable)},
	})

	v.updateExprInfo(exprInfo{
		expr:            ctx.GetLhs(),
		ExprDescription: "IS LHS",
		ExprRef:         ctx.GetRhs(),
		Type:            []exprType{knownType("", nullable)},
	})

	v.updateExprInfo(exprInfo{
		expr:            ctx.GetRhs(),
		ExprDescription: "IS RHS",
		ExprRef:         ctx.GetLhs(),
		Type:            []exprType{knownType("", nullable)},
	})

	return nil
}

func (v *visitor) VisitExpr_concat(ctx *sqliteparser.Expr_concatContext) any {
	v.VisitChildren(ctx)
	if v.err != nil {
		return nil
	}

	v.updateExprInfo(exprInfo{
		expr:            ctx,
		ExprDescription: "Concat",
		Type:            []exprType{knownType("TEXT", notNullable)},
	})

	v.updateExprInfo(exprInfo{
		expr:            ctx.GetLhs(),
		ExprDescription: "Concat LHS",
		Type:            []exprType{knownType("TEXT", notNullable)},
	})

	v.updateExprInfo(exprInfo{
		expr:            ctx.GetRhs(),
		ExprDescription: "Concat RHS",
		Type:            []exprType{knownType("TEXT", notNullable)},
	})

	return nil
}

func (v *visitor) VisitExpr_list(ctx *sqliteparser.Expr_listContext) any {
	exprs := ctx.AllExpr()
	if len(exprs) == 1 {
		v.addName(ctx, exprName{
			names: func() []string {
				return v.getExprName(exprs[0])
			},
		})
	}

	return v.VisitChildren(ctx)
}

func (v *visitor) VisitExpr_in(ctx *sqliteparser.Expr_inContext) any {
	v.VisitChildren(ctx)
	if v.err != nil {
		return nil
	}

	rhsExpr := ctx.GetRhsExpr()
	if rhsExpr == nil {
		return nil
	}

	rhsList, rhsIsList := rhsExpr.(*sqliteparser.Expr_listContext)
	if !rhsIsList {
		return nil
	}
	rhsChildren := rhsList.AllExpr()

	// If there is only one child, it can be multiple
	singleIn := len(rhsChildren) == 1

	lhs := ctx.GetLhs()
	lhsList, lhsIsList := lhs.(*sqliteparser.Expr_listContext)

	var lhsChildren []sqliteparser.IExprContext
	if lhsIsList {
		lhsChildren = lhsList.AllExpr()
	}

	childRefs := make(map[nodeKey]exprChildNameRef, len(rhsChildren))
	for _, child := range rhsChildren {
		v.updateExprInfo(exprInfo{
			expr:                 child,
			ExprDescription:      "IN RHS",
			ExprRef:              lhs,
			IgnoreRefNullability: true,
			isGroup:              true,
			CanBeMultiple:        singleIn,
		})
		childRefs[key(child)] = func() ([]string, []string) {
			return nil, v.getExprName(lhs)
		}

		childList, childIsList := child.(*sqliteparser.Expr_listContext)
		if !childIsList {
			continue
		}
		grandChildren := childList.AllExpr()

		if len(grandChildren) != len(lhsChildren) {
			v.err = fmt.Errorf("IN: list length mismatch %d != %d", len(grandChildren), len(lhsChildren))
			return nil
		}

		grandChildRefs := make(map[nodeKey]exprChildNameRef, len(grandChildren))
		for i, grandChild := range grandChildren {
			v.updateExprInfo(exprInfo{
				expr:                 grandChild,
				ExprDescription:      "IN RHS List",
				ExprRef:              lhsChildren[i],
				IgnoreRefNullability: true,
			})
			grandChildRefs[key(grandChild)] = func() ([]string, []string) {
				return nil, v.getExprName(lhsChildren[i])
			}
		}

		v.addName(childList, exprName{
			childRefs: grandChildRefs,
		})

		v.stmtRules = append(v.stmtRules,
			internal.RecordPoints(
				childList.GetStart().GetStart(),
				childList.GetStop().GetStop(),
				func(start, end int) error {
					v.updateExprInfo(exprInfo{
						expr:           childList,
						EditedPosition: [2]int{start, end},
					})
					return nil
				},
			)...,
		)
	}

	v.addName(rhsList, exprName{
		childRefs: childRefs,
	})

	return nil
}

func (v *visitor) VisitExpr_collate(ctx *sqliteparser.Expr_collateContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by SQLiteParser#expr_modulo.
func (v *visitor) VisitExpr_modulo(ctx *sqliteparser.Expr_moduloContext) any {
	v.VisitChildren(ctx)
	if v.err != nil {
		return nil
	}

	v.updateExprInfo(exprInfo{
		expr:            ctx,
		ExprDescription: "Modulo",
		Type:            []exprType{knownType("INTEGER", notNullable)},
	})

	v.updateExprInfo(exprInfo{
		expr:            ctx.GetLhs(),
		ExprDescription: "Modulo LHS",
		Type:            []exprType{knownType("INTEGER", notNullable)},
	})

	v.updateExprInfo(exprInfo{
		expr:            ctx.GetRhs(),
		ExprDescription: "Modulo RHS",
		Type:            []exprType{knownType("INTEGER", notNullable)},
	})

	return nil
}

func (v *visitor) VisitExpr_qualified_column_name(ctx *sqliteparser.Expr_qualified_column_nameContext) any {
	v.addRawName(
		ctx,
		getName(ctx.Schema_name()),
		getName(ctx.Table_name()),
		getName(ctx.Column_name()),
	)

	v.VisitChildren(ctx)
	if v.err != nil {
		return nil
	}

	v.updateExprInfo(exprInfo{
		expr:            ctx,
		ExprDescription: "Qualified",
		Type:            makeRef(v.sources, ctx),
	})

	return nil
}

func (v *visitor) VisitExpr_match(ctx *sqliteparser.Expr_matchContext) any {
	v.VisitChildren(ctx)
	if v.err != nil {
		return nil
	}

	v.updateExprInfo(exprInfo{
		expr:            ctx,
		ExprDescription: "Match",
		Type:            []exprType{knownType("BOOLEAN", notNullable)},
	})

	v.updateExprInfo(exprInfo{
		expr:            ctx.GetRhs(),
		ExprDescription: "Modulo RHS",
		Type:            []exprType{knownType("TEXT", notNullable)},
	})

	return nil
}

func (v *visitor) VisitExpr_like(ctx *sqliteparser.Expr_likeContext) any {
	v.VisitChildren(ctx)
	if v.err != nil {
		return nil
	}

	v.updateExprInfo(exprInfo{
		expr:            ctx,
		ExprDescription: "LIKE",
		Type:            []exprType{knownType("BOOLEAN", notNullable)},
	})

	v.updateExprInfo(exprInfo{
		expr:            ctx.GetLhs(),
		ExprDescription: "Like LHS",
		Type:            []exprType{knownType("TEXT", notNullable)},
	})

	v.updateExprInfo(exprInfo{
		expr:            ctx.GetRhs(),
		ExprDescription: "Like RHS",
		Type:            []exprType{knownType("TEXT", notNullable)},
	})

	return nil
}

func (v *visitor) VisitExpr_null_comp(ctx *sqliteparser.Expr_null_compContext) any {
	v.VisitChildren(ctx)
	if v.err != nil {
		return nil
	}

	v.updateExprInfo(exprInfo{
		expr:            ctx,
		ExprDescription: "NULL Comparison",
		Type:            []exprType{knownType("BOOLEAN", notNullable)},
	})

	v.updateExprInfo(exprInfo{
		expr:            ctx.Expr(),
		ExprDescription: "NULL Comparison Expr",
		Type:            []exprType{knownType("", notNullable)},
	})

	return nil
}

func (v *visitor) VisitExpr_json_extract_json(ctx *sqliteparser.Expr_json_extract_jsonContext) any {
	v.VisitChildren(ctx)
	if v.err != nil {
		return nil
	}

	v.updateExprInfo(exprInfo{
		expr:            ctx,
		ExprDescription: "JSON->",
		Type:            []exprType{knownType("JSON", notNullable)},
	})

	v.updateExprInfo(exprInfo{
		expr:            ctx.GetLhs(),
		ExprDescription: "JSON-> LHS",
		Type:            []exprType{knownType("JSON", notNullable)},
	})

	v.updateExprInfo(exprInfo{
		expr:            ctx.GetRhs(),
		ExprDescription: "JSON-> RHS",
		Type:            []exprType{knownType("TEXT", notNullable)},
	})

	return nil
}

func (v *visitor) VisitExpr_exists_select(ctx *sqliteparser.Expr_exists_selectContext) any {
	return v.VisitChildren(ctx)
}

func (v *visitor) VisitExpr_comparison(ctx *sqliteparser.Expr_comparisonContext) any {
	opName := ctx.GetParser().GetSymbolicNames()[ctx.GetOperator().GetTokenType()]
	opName = strings.ToLower(opName)
	if opName == "eq" {
		opName = ""
	}
	v.addLRName(ctx, opName)

	v.VisitChildren(ctx)

	if v.err != nil {
		return nil
	}

	v.updateExprInfo(exprInfo{
		expr:            ctx,
		ExprDescription: "Comparison",
		Type:            []exprType{knownType("BOOLEAN", notNullable)},
	})

	v.updateExprInfo(exprInfo{
		expr:                 ctx.GetLhs(),
		ExprDescription:      "Comparison LHS",
		ExprRef:              ctx.GetRhs(),
		IgnoreRefNullability: true,
	})

	v.updateExprInfo(exprInfo{
		expr:                 ctx.GetRhs(),
		ExprDescription:      "Comparison RHS",
		ExprRef:              ctx.GetLhs(),
		IgnoreRefNullability: true,
	})

	return nil
}

func (v *visitor) VisitExpr_literal(ctx *sqliteparser.Expr_literalContext) any {
	v.VisitChildren(ctx)
	if v.err != nil {
		return nil
	}

	var DBType exprType

	typ := ctx.Literal_value().GetLiteralType().GetTokenType()
	switch typ {
	case sqliteparser.SQLiteParserNUMERIC_LITERAL:
		v.addRawName(ctx, ctx.GetText())

		if strings.ContainsAny(ctx.GetText(), ".eE") {
			DBType = knownType("REAL", notNullable)
			break
		}

		text := strings.ReplaceAll(ctx.GetText(), "_", "")
		if len(text) < 2 {
			DBType = knownType("INTEGER", notNullable)
			break
		}

		base := 10

		if strings.EqualFold(text[0:2], "0x") {
			text = text[2:]
			base = 16
		}

		_, err := strconv.ParseInt(text, base, 64)
		if err == nil {
			DBType = knownType("INTEGER", notNullable)
			break
		}

		if errors.Is(err, strconv.ErrRange) {
			DBType = knownType("REAL", notNullable)
			break
		}

		v.err = fmt.Errorf("cannot parse numeric integer: %s", ctx.GetText())
		return nil

	case sqliteparser.SQLiteParserSTRING_LITERAL:
		DBType = knownType("TEXT", notNullable)
		txt := strings.ReplaceAll(ctx.GetText(), "'", "")
		v.addRawName(ctx, txt)

	case sqliteparser.SQLiteParserBLOB_LITERAL:
		DBType = knownType("BLOB", notNullable)
		v.addRawName(ctx, "BLOB")

	case sqliteparser.SQLiteParserNULL_:
		DBType = knownType("", nullable)
		v.addRawName(ctx, "NULL")

	case sqliteparser.SQLiteParserTRUE_,
		sqliteparser.SQLiteParserFALSE_:
		DBType = knownType("BOOLEAN", notNullable)
		v.addRawName(ctx, ctx.GetText())

	case sqliteparser.SQLiteParserCURRENT_TIME_,
		sqliteparser.SQLiteParserCURRENT_DATE_,
		sqliteparser.SQLiteParserCURRENT_TIMESTAMP_:
		DBType = knownType("DATETIME", notNullable)
		v.addRawName(ctx, ctx.GetText()[:len(ctx.GetText())-1])

	default:
		v.err = fmt.Errorf("unknown literal type: %d", typ)
		return nil
	}

	info := exprInfo{
		expr:            ctx,
		ExprDescription: "Literal",
	}

	if len(DBType.typeName) > 0 {
		info.Type = []exprType{DBType}
	}

	v.updateExprInfo(info)

	return nil
}

func (v *visitor) VisitExpr_cast(ctx *sqliteparser.Expr_castContext) any {
	v.VisitChildren(ctx)
	if v.err != nil {
		return nil
	}

	v.updateExprInfo(exprInfo{
		expr:            ctx,
		ExprDescription: "CAST",
		Type:            []exprType{knownType(ctx.Type_name().GetText(), notNullable)},
	})

	return nil
}

func (v *visitor) VisitExpr_string_op(ctx *sqliteparser.Expr_string_opContext) any {
	v.VisitChildren(ctx)
	if v.err != nil {
		return nil
	}

	v.updateExprInfo(exprInfo{
		expr:            ctx,
		ExprDescription: "String OP",
		Type:            []exprType{knownType("BOOLEAN", notNullable)},
	})

	v.updateExprInfo(exprInfo{
		expr:            ctx.GetLhs(),
		ExprDescription: "String OP LHS",
		Type:            []exprType{knownType("TEXT", notNullable)},
	})

	v.updateExprInfo(exprInfo{
		expr:            ctx.GetRhs(),
		ExprDescription: "String OP RHS",
		Type:            []exprType{knownType("TEXT", notNullable)},
	})

	return nil
}

func (v *visitor) VisitExpr_between(ctx *sqliteparser.Expr_betweenContext) any {
	return v.VisitChildren(ctx)
}

func (v *visitor) VisitExpr_bitwise(ctx *sqliteparser.Expr_bitwiseContext) any {
	v.VisitChildren(ctx)
	if v.err != nil {
		return nil
	}

	v.updateExprInfo(exprInfo{
		expr:            ctx,
		ExprDescription: "Bitwise",
		Type:            []exprType{knownType("INTEGER", notNullable)},
	})

	v.updateExprInfo(exprInfo{
		expr:            ctx.GetLhs(),
		ExprDescription: "Bitwise LHS",
		Type:            []exprType{knownType("INTEGER", notNullable)},
	})

	v.updateExprInfo(exprInfo{
		expr:            ctx.GetRhs(),
		ExprDescription: "Bitwise RHS",
		Type:            []exprType{knownType("INTEGER", notNullable)},
	})

	return nil
}

func (v *visitor) VisitExpr_unary(ctx *sqliteparser.Expr_unaryContext) any {
	v.VisitChildren(ctx)
	if v.err != nil {
		return nil
	}

	tokenTyp := ctx.Unary_operator().GetOperator().GetTokenType()
	switch tokenTyp {
	case sqliteparser.SQLiteParserPLUS:
		// Returns the same type as the operand
		v.updateExprInfo(exprInfo{
			expr:            ctx,
			ExprDescription: "Unary Plus",
			ExprRef:         ctx.Expr(),
		})

		v.updateExprInfo(exprInfo{
			expr:            ctx.Expr(),
			ExprDescription: "Unary Plus Expr",
			ExprRef:         ctx,
		})

	case sqliteparser.SQLiteParserMINUS:
		// Always INTEGER, should be used with a numeric literal
		v.updateExprInfo(exprInfo{
			expr:            ctx,
			ExprDescription: "Unary Minus",
			Type:            []exprType{knownType("INTEGER", notNullable), knownType("REAL", notNullable)},
		})

		v.updateExprInfo(exprInfo{
			expr:            ctx.Expr(),
			ExprDescription: "Unary Minus Expr",
			Type:            []exprType{knownType("INTEGER", notNullable), knownType("REAL", notNullable)},
		})

	case sqliteparser.SQLiteParserTILDE:
		// Bitwise NOT
		// Always INTEGER, should be used with a numeric literal
		v.updateExprInfo(exprInfo{
			expr:            ctx,
			ExprDescription: "Unary Tilde",
			Type:            []exprType{knownType("INTEGER", notNullable)},
		})

		v.updateExprInfo(exprInfo{
			expr:            ctx.Expr(),
			ExprDescription: "Unary Tilde Expr",
			Type:            []exprType{knownType("INTEGER", notNullable)},
		})

	case sqliteparser.SQLiteParserNOT_:
		// Returns a BOOLEAN (should technically only be used with a boolean expression)
		v.updateExprInfo(exprInfo{
			expr:            ctx,
			ExprDescription: "Unary NOT",
			Type:            []exprType{knownType("BOOLEAN", notNullable)},
		})

		v.updateExprInfo(exprInfo{
			expr:            ctx.Expr(),
			ExprDescription: "Unary NOT Expr",
			Type:            []exprType{knownType("BOOLEAN", notNullable)},
		})
	}

	return nil
}

func (v *visitor) VisitExpr_bind(ctx *sqliteparser.Expr_bindContext) any {
	v.VisitChildren(ctx)
	if v.err != nil {
		return nil
	}

	info := exprInfo{
		expr:            ctx,
		ExprDescription: "Bind",
		queryArgKey:     ctx.GetText()[1:],
	}

	if ctx.NAMED_BIND_PARAMETER() != nil {
		info.config.Name = info.queryArgKey
	}

	parent, ok := ctx.GetParent().(*sqliteparser.Expr_castContext)
	if ok {
		info.ExprRef = parent
	}

	v.updateExprInfo(info)

	v.stmtRules = append(v.stmtRules, internal.EditCallback(
		internal.ReplaceFromFunc(
			ctx.GetStart().GetStart(), ctx.GetStop().GetStop(),
			func() string {
				return fmt.Sprintf("?%d", v.atom.Add(1))
			},
		),
		func(start, end int, _, _ string) error {
			v.updateExprInfo(exprInfo{
				expr:           ctx,
				EditedPosition: [2]int{start, end},
			})
			return nil
		}),
	)

	return nil
}

func (v *visitor) VisitExpr_simple_func(ctx *sqliteparser.Expr_simple_funcContext) any {
	v.VisitChildren(ctx)
	if v.err != nil {
		v.err = fmt.Errorf("simple function invocation: %w", v.err)
		return nil
	}

	args := ctx.AllExpr()
	argTypes := make([]string, len(args))
	missingTypes := make([]bool, len(args))
	nullable := make([]func() bool, len(args))
	for i, arg := range args {
		argTypes[i] = v.exprs[key(arg)].Type.ConfirmedAffinity()
		nullable[i] = v.exprs[key(arg)].Type.Nullable
		if argTypes[i] == "" {
			missingTypes[i] = true
		}
	}

	funcName := getName(ctx.Simple_func())
	funcDef, err := v.getFunctionType(funcName, argTypes)
	if err != nil {
		v.err = fmt.Errorf("simple function invocation: %w", err)
		return nil
	}

	for i, arg := range args {
		if missingTypes[i] {
			v.updateExprInfo(exprInfo{
				expr:            arg,
				ExprDescription: "Function Arg",
				Type: []exprType{knownType(
					funcDef.argType(i),
					func() bool { return funcDef.shouldArgsBeNullable },
				)},
			})
		}
	}

	info := exprInfo{
		expr:            ctx,
		ExprDescription: "Function Arg",
		Type: []exprType{knownType(
			funcDef.returnType,
			anyNullable(nullable...),
		)},
	}

	if funcDef.calcNullable != nil {
		info.Type[0].nullableF = funcDef.calcNullable(nullable...)
	}

	v.updateExprInfo(info)
	v.addRawName(ctx, funcName)

	return nil
}

func (v *visitor) VisitExpr_aggregate_func(ctx *sqliteparser.Expr_aggregate_funcContext) any {
	return v.VisitChildren(ctx)
}

func (v *visitor) VisitExpr_window_func(ctx *sqliteparser.Expr_window_funcContext) any {
	return v.VisitChildren(ctx)
}

func (v *visitor) VisitRaise_function(ctx *sqliteparser.Raise_functionContext) any {
	return v.VisitChildren(ctx)
}

func (v *visitor) VisitLiteral_value(ctx *sqliteparser.Literal_valueContext) any {
	return v.VisitChildren(ctx)
}

func (v *visitor) VisitValue_row(ctx *sqliteparser.Value_rowContext) any {
	return v.VisitChildren(ctx)
}

func (v *visitor) VisitValues_clause(ctx *sqliteparser.Values_clauseContext) any {
	return v.VisitChildren(ctx)
}

func (v *visitor) VisitInsert_stmt(ctx *sqliteparser.Insert_stmtContext) any {
	return v.VisitChildren(ctx)
}

func (v *visitor) VisitReturning_clause(ctx *sqliteparser.Returning_clauseContext) any {
	return v.VisitChildren(ctx)
}

func (v *visitor) VisitUpsert_clause(ctx *sqliteparser.Upsert_clauseContext) any {
	return v.VisitChildren(ctx)
}

func (v *visitor) VisitPragma_stmt(ctx *sqliteparser.Pragma_stmtContext) any {
	return v.VisitChildren(ctx)
}

func (v *visitor) VisitPragma_value(ctx *sqliteparser.Pragma_valueContext) any {
	return v.VisitChildren(ctx)
}

func (v *visitor) VisitReindex_stmt(ctx *sqliteparser.Reindex_stmtContext) any {
	return v.VisitChildren(ctx)
}

// Should return a stmt info
func (v *visitor) VisitSelect_stmt(ctx *sqliteparser.Select_stmtContext) any {
	v.modSelect_stmt(ctx, v.mods)

	// Create a new visitor, to not mix sources
	// however, we clone any existing sources to the new visitor
	v2 := &visitor{
		db:        v.db,
		exprs:     v.exprs,
		names:     v.names,
		sources:   slices.Clone(v.sources),
		functions: v.functions,
		atom:      v.atom,
		mods:      &strings.Builder{},
	}

	if ctx.With_clause() != nil {
		ctx.With_clause().Accept(v2)
		if v.err != nil {
			v.err = fmt.Errorf("with clause: %w", v.err)
			return nil
		}
	}

	// Should return a source
	// Use the first select core to get the columns
	sourceAny := v2.visitSelect_core(ctx.Select_core())
	source, ok := sourceAny.(querySource)
	if v2.err != nil {
		v.err = fmt.Errorf("select core 0: %w", v2.err)
		return nil
	}

	if !ok {
		v.err = fmt.Errorf("could not get source from select core 0: %T", sourceAny)
		return nil
	}
	v.stmtRules = append(v.stmtRules, v2.stmtRules...)

	for i, compound := range ctx.AllCompound_select() {
		v3 := &visitor{
			db:        v.db,
			exprs:     v.exprs,
			names:     v.names,
			sources:   slices.Clone(v2.sources),
			functions: v.functions,
			atom:      v.atom,
			mods:      &strings.Builder{},
		}

		coreSource, ok := v3.visitSelect_core(compound.Select_core()).(querySource)
		if v3.err != nil {
			v.err = fmt.Errorf("select core %d: %w", i, v3.err)
			return nil
		}

		if !ok {
			v.err = fmt.Errorf("could not get source from select core %d", i)
		}

		if len(source.columns) != len(coreSource.columns) {
			v.err = fmt.Errorf("select core %d: column count mismatch %d != %d", i, len(source.columns), len(coreSource.columns))
		}

		v.stmtRules = append(v.stmtRules, v3.stmtRules...)

		for i, col := range source.columns {
			matchingTypes := matchTypes(
				col.typ, coreSource.columns[i].typ,
			)

			if len(source.columns[i].typ) == 0 {
				v.err = fmt.Errorf(
					"select core %d: column %d type mismatch:\n%v\n%v",
					i, i, col.typ, coreSource.columns[i].typ,
				)
			}

			source.columns[i].typ = matchingTypes
		}
	}

	if order := ctx.Order_by_stmt(); order != nil {
		order.Accept(v)
		if v.err != nil {
			v.err = fmt.Errorf("order by: %w", v.err)
			return nil
		}
	}

	if limit := ctx.Limit_stmt(); limit != nil {
		limit.Accept(v)
		if v.err != nil {
			v.err = fmt.Errorf("limit: %w", v.err)
			return nil
		}
	}

	return source.columns
}

// Should return a query source
func (v *visitor) VisitSelect_core(ctx *sqliteparser.Select_coreContext) any {
	return nil // do not visit children automatically
}

// Should return a query source
func (v *visitor) visitSelect_core(ctx sqliteparser.ISelect_coreContext) any {
	v.visitFrom_item(ctx.From_item())
	if v.err != nil {
		v.err = fmt.Errorf("from item: %w", v.err)
		return nil
	}

	// Evaluate all the expressions
	v.VisitChildren(ctx)
	if v.err != nil {
		v.err = fmt.Errorf("select core children: %w", v.err)
		return nil
	}

	// Get the return columns
	var returnSource querySource

	for _, resultColumn := range ctx.AllResult_column() {
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
			v.stmtRules = append(v.stmtRules, internal.Insert(stop, buf.String()))

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
				config:  drivers.ParseQueryColumnConfig(v.getCommentToRight(expr)),
				typ:     v.exprs[key(resultColumn.Expr())].Type,
			})
		}
	}

	return returnSource
}

//nolint:nestif
func (v *visitor) modSelect_stmt(ctx sqliteparser.ISelect_stmtContext, sb *strings.Builder) {
	if with := ctx.With_clause(); with != nil {
		if with.RECURSIVE_() != nil {
			sb.WriteString("q.SetRecursive(true)\n")
		}
		for _, cte := range with.AllCommon_table_expression() {
			v.stmtRules = append(v.stmtRules,
				internal.RecordPoints(
					cte.GetStart().GetStart(),
					cte.GetStop().GetStop(),
					func(start, end int) error {
						fmt.Fprintf(sb, "q.AppendCTE(o.expr(%d, %d))\n", start, end)
						return nil
					},
				)...,
			)
		}
	}

	{
		core := ctx.Select_core()

		if distinct := core.DISTINCT_(); distinct != nil {
			sb.WriteString("q.Distinct = true\n")
		}

		allResults := core.AllResult_column()
		if len(allResults) > 0 {
			v.stmtRules = append(v.stmtRules,
				internal.RecordPoints(
					allResults[0].GetStart().GetStart(),
					allResults[len(allResults)-1].GetStop().GetStop(),
					func(start, end int) error {
						fmt.Fprintf(sb, "q.AppendSelect(o.expr(%d, %d))\n", start, end)
						return nil
					},
				)...,
			)
		}

		if from := core.From_item(); from != nil {
			v.stmtRules = append(v.stmtRules,
				internal.RecordPoints(
					from.GetStart().GetStart(),
					from.GetStop().GetStop(),
					func(start, end int) error {
						fmt.Fprintf(sb, "q.SetTable(o.expr(%d, %d))\n", start, end)
						return nil
					},
				)...,
			)
		}

		if where := core.Where_stmt(); where != nil {
			v.stmtRules = append(v.stmtRules,
				internal.RecordPoints(
					where.GetStart().GetStart()+len("WHERE "),
					where.GetStop().GetStop(),
					func(start, end int) error {
						fmt.Fprintf(sb, "q.AppendWhere(o.expr(%d, %d))\n", start, end)
						return nil
					},
				)...,
			)
		}

		if groupBy := core.Group_by_stmt(); groupBy != nil {
			v.stmtRules = append(v.stmtRules,
				internal.RecordPoints(
					groupBy.GetStart().GetStart(),
					groupBy.GetStop().GetStop(),
					func(start, end int) error {
						fmt.Fprintf(sb, "q.AppendGroup(o.expr(%d, %d))\n", start, end)
						return nil
					},
				)...,
			)
		}

		if having := core.GetHavingExpr(); having != nil {
			v.stmtRules = append(v.stmtRules,
				internal.RecordPoints(
					having.GetStart().GetStart(),
					having.GetStop().GetStop(),
					func(start, end int) error {
						fmt.Fprintf(sb, "q.AppendHaving(o.expr(%d, %d))\n", start, end)
						return nil
					},
				)...,
			)
		}

		for _, window := range core.AllWindow_stmt() {
			v.stmtRules = append(v.stmtRules,
				internal.RecordPoints(
					window.GetStart().GetStart(),
					window.GetStop().GetStop(),
					func(start, end int) error {
						fmt.Fprintf(sb, "q.AppendWindow(o.expr(%d, %d))\n", start, end)
						return nil
					},
				)...,
			)
		}
	}

	compounds := ctx.AllCompound_select()

	if len(compounds) > 0 {
		v.imports = append(v.imports, []string{"github.com/stephenafamo/bob/clause"})
	}

	for _, compound := range ctx.AllCompound_select() {
		strategy := strings.ToUpper(compound.Compound_operator().GetText())
		all := compound.Compound_operator().ALL_() != nil
		if all {
			strategy = strategy[:len(strategy)-3]
		}
		v.stmtRules = append(v.stmtRules,
			internal.RecordPoints(
				compound.Select_core().GetStart().GetStart(),
				compound.Select_core().GetStop().GetStop(),
				func(start, end int) error {
					fmt.Fprintf(sb, `
                        q.AppendCombine(clause.Combine{
                            Strategy: "%s",
                            All: %t,
                            Query: bob.BaseQuery[bob.Expression]{
                                Expression: o.expr(%d, %d),
                                QueryType: bob.QueryTypeSelect,
                                Dialect: dialect.Dialect,
                            },
                        })
                    `, strategy, all, start, end)
					return nil
				},
			)...,
		)
	}

	if order := ctx.Order_by_stmt(); order != nil {
		v.stmtRules = append(v.stmtRules,
			internal.RecordPoints(
				order.GetStart().GetStart()+len("ORDER BY "),
				order.GetStop().GetStop(),
				func(start, end int) error {
					fmt.Fprintf(sb, "q.AppendOrder(o.expr(%d, %d))\n", start, end)
					return nil
				},
			)...,
		)
	}

	if limit := ctx.Limit_stmt(); limit != nil {
		limiter := limit.GetFirstExpr()
		comma := limit.COMMA()
		if comma != nil {
			limiter = limit.GetLastExpr()
		}

		v.stmtRules = append(v.stmtRules,
			internal.RecordPoints(
				limiter.GetStart().GetStart(),
				limiter.GetStop().GetStop(),
				func(start, end int) error {
					fmt.Fprintf(sb, "q.SetLimit(o.expr(%d, %d))\n", start, end)
					return nil
				},
			)...,
		)

		if comma == nil {
			if offset := limit.GetLastExpr(); offset != nil {
				v.stmtRules = append(v.stmtRules,
					internal.RecordPoints(
						offset.GetStart().GetStart(),
						offset.GetStop().GetStop(),
						func(start, end int) error {
							fmt.Fprintf(sb, "q.SetOffset(o.expr(%d, %d))\n", start, end)
							return nil
						},
					)...,
				)
			}
		} else {
			var s string
			v.stmtRules = append(v.stmtRules, internal.EditCallback(
				internal.Delete(
					limit.GetFirstExpr().GetStart().GetStart()-1,
					comma.GetSymbol().GetStop(),
				),
				func(_, _ int, before, after string) error {
					s = before
					return nil
				}),
			)

			v.stmtRules = append(v.stmtRules, internal.EditCallback(
				internal.InsertFromFunc(
					limit.GetLastExpr().GetStop().GetStop()+1,
					func() string {
						return fmt.Sprintf(" OFFSET %s", s[1:len(s)-1])
					},
				),
				func(start, end int, _, _ string) error {
					fmt.Fprintf(sb, "q.SetOffset(%q)\n", s[1:len(s)-1])
					return nil
				}),
			)
		}
	}
}

func (v *visitor) VisitResult_column(ctx *sqliteparser.Result_columnContext) any {
	return v.VisitChildren(ctx)
}

func (v *visitor) visitFrom_item(ctx sqliteparser.IFrom_itemContext) {
	tables := ctx.AllTable_or_subquery()

	sources := make(querySources, len(tables))
	for i, table := range tables {
		sources[i] = v.visitTable_or_subquery(table)
		if v.err != nil {
			v.err = fmt.Errorf("table or subquery %d: %w", i, v.err)
			return
		}
	}

	for i, joinOp := range ctx.AllJoin_operator() {
		fullJoin := joinOp.FULL_() != nil
		leftJoin := fullJoin || joinOp.LEFT_() != nil
		rightJoin := fullJoin || joinOp.RIGHT_() != nil

		if leftJoin {
			right := sources[i+1]
			for i := range right.columns {
				for j := range right.columns[i].typ {
					right.columns[i].typ[j].nullableF = nullable
				}
			}
		}

		if rightJoin {
			left := sources[i+1]
			for i := range left.columns {
				for j := range left.columns[i].typ {
					left.columns[i].typ[j].nullableF = nullable
				}
			}
		}
	}

	v.sources = append(v.sources, sources...)
}

func (v *visitor) VisitFrom_item(ctx *sqliteparser.From_itemContext) any {
	// return v.VisitChildren(ctx)
	return nil // do not visit children automatically
}

func (v *visitor) VisitJoin_operator(ctx *sqliteparser.Join_operatorContext) any {
	return v.VisitChildren(ctx)
}

func (v *visitor) VisitJoin_constraint(ctx *sqliteparser.Join_constraintContext) any {
	return v.VisitChildren(ctx)
}

func (v *visitor) VisitTable_or_subquery(ctx *sqliteparser.Table_or_subqueryContext) any {
	panic("should not be called")
}

func (v *visitor) visitTable_or_subquery(ctx sqliteparser.ITable_or_subqueryContext) querySource {
	switch {
	case ctx.Table_name() != nil:
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
			if table.Schema == schema && table.Name == tableName {
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
		}

		v.err = fmt.Errorf("table not found: %s", tableName)
		return querySource{}

	case ctx.Select_stmt() != nil:
		columns, ok := ctx.Select_stmt().Accept(v).(returns)
		if v.err != nil {
			v.err = fmt.Errorf("table select stmt: %w", v.err)
			return querySource{}
		}
		if !ok {
			v.err = fmt.Errorf("could not get stmt info")
			return querySource{}
		}

		return querySource{
			name:    getName(ctx.Table_alias()),
			columns: columns,
		}

	case ctx.Table_or_subquery() != nil:
		return v.visitTable_or_subquery(ctx.Table_or_subquery())

	default:
		v.err = fmt.Errorf("unknown table or subquery: %#v", key(ctx))
		return querySource{}
	}
}

func (v *visitor) VisitCompound_select(ctx *sqliteparser.Compound_selectContext) any {
	return v.VisitChildren(ctx)
}

func (v *visitor) VisitCompound_operator(ctx *sqliteparser.Compound_operatorContext) any {
	return v.VisitChildren(ctx)
}

func (v *visitor) VisitUpdate_stmt(ctx *sqliteparser.Update_stmtContext) any {
	return v.VisitChildren(ctx)
}

func (v *visitor) VisitColumn_name_list(ctx *sqliteparser.Column_name_listContext) any {
	return v.VisitChildren(ctx)
}

func (v *visitor) VisitQualified_table_name(ctx *sqliteparser.Qualified_table_nameContext) any {
	return v.VisitChildren(ctx)
}

func (v *visitor) VisitVacuum_stmt(ctx *sqliteparser.Vacuum_stmtContext) any {
	return v.VisitChildren(ctx)
}

func (v *visitor) VisitFilter_clause(ctx *sqliteparser.Filter_clauseContext) any {
	return v.VisitChildren(ctx)
}

func (v *visitor) VisitWindow_defn(ctx *sqliteparser.Window_defnContext) any {
	return v.VisitChildren(ctx)
}

func (v *visitor) VisitOver_clause(ctx *sqliteparser.Over_clauseContext) any {
	return v.VisitChildren(ctx)
}

func (v *visitor) VisitFrame_spec(ctx *sqliteparser.Frame_specContext) any {
	return v.VisitChildren(ctx)
}

func (v *visitor) VisitFrame_clause(ctx *sqliteparser.Frame_clauseContext) any {
	return v.VisitChildren(ctx)
}

func (v *visitor) VisitWith_clause(ctx *sqliteparser.With_clauseContext) any {
	if ctx.RECURSIVE_() != nil {
		v.mods.WriteString(`q.SetRecursive(true)`)
	}
	return v.VisitChildren(ctx)
}

func (v *visitor) VisitCommon_table_expression(ctx *sqliteparser.Common_table_expressionContext) any {
	columns, ok := ctx.Select_stmt().Accept(v).(returns)
	if v.err != nil {
		v.err = fmt.Errorf("CTE select stmt: %w", v.err)
		return nil
	}
	if !ok {
		v.err = fmt.Errorf("could not get stmt info")
		return nil
	}

	source := querySource{
		name:    getName(ctx.Table_name()),
		columns: columns,
		cte:     true,
	}

	columnNames := ctx.AllColumn_name()
	if len(columnNames) == 0 {
		v.sources = append(v.sources, source)
		return nil
	}

	if len(columnNames) != len(source.columns) {
		v.err = fmt.Errorf("column names do not match %d != %d", len(columnNames), len(source.columns))
		return nil
	}

	for i, column := range columnNames {
		source.columns[i].name = getName(column)
	}

	v.sources = append(v.sources, source)
	return nil
}

func (v *visitor) VisitWhere_stmt(ctx *sqliteparser.Where_stmtContext) any {
	v.addName(ctx, exprName{
		childRefs: map[nodeKey]exprChildNameRef{
			key(ctx.Expr()): func() ([]string, []string) {
				return []string{"where"}, nil
			},
		},
	})

	v.VisitChildren(ctx)
	if v.err != nil {
		v.err = fmt.Errorf("where stmt: %w", v.err)
		return nil
	}

	return nil
}

func (v *visitor) VisitOrder_by_stmt(ctx *sqliteparser.Order_by_stmtContext) any {
	return v.VisitChildren(ctx)
}

func (v *visitor) VisitGroup_by_stmt(ctx *sqliteparser.Group_by_stmtContext) any {
	return v.VisitChildren(ctx)
}

func (v *visitor) VisitWindow_stmt(ctx *sqliteparser.Window_stmtContext) any {
	return v.VisitChildren(ctx)
}

func (v *visitor) VisitLimit_stmt(ctx *sqliteparser.Limit_stmtContext) any {
	return v.VisitChildren(ctx)
}

func (v *visitor) VisitOrdering_term(ctx *sqliteparser.Ordering_termContext) any {
	return v.VisitChildren(ctx)
}

func (v *visitor) VisitAsc_desc(ctx *sqliteparser.Asc_descContext) any {
	return v.VisitChildren(ctx)
}

func (v *visitor) VisitFrame_left(ctx *sqliteparser.Frame_leftContext) any {
	return v.VisitChildren(ctx)
}

func (v *visitor) VisitFrame_right(ctx *sqliteparser.Frame_rightContext) any {
	return v.VisitChildren(ctx)
}

func (v *visitor) VisitFrame_single(ctx *sqliteparser.Frame_singleContext) any {
	return v.VisitChildren(ctx)
}

func (v *visitor) VisitOffset(ctx *sqliteparser.OffsetContext) any {
	return v.VisitChildren(ctx)
}

func (v *visitor) VisitDefault_value(ctx *sqliteparser.Default_valueContext) any {
	return v.VisitChildren(ctx)
}

func (v *visitor) VisitPartition_by(ctx *sqliteparser.Partition_byContext) any {
	return v.VisitChildren(ctx)
}

func (v *visitor) VisitOrder_by_expr(ctx *sqliteparser.Order_by_exprContext) any {
	return v.VisitChildren(ctx)
}

func (v *visitor) VisitOrder_by_expr_asc_desc(ctx *sqliteparser.Order_by_expr_asc_descContext) any {
	return v.VisitChildren(ctx)
}

func (v *visitor) VisitUnary_operator(ctx *sqliteparser.Unary_operatorContext) any {
	return v.VisitChildren(ctx)
}

func (v *visitor) VisitError_message(ctx *sqliteparser.Error_messageContext) any {
	return v.VisitChildren(ctx)
}

func (v *visitor) VisitModule_argument(ctx *sqliteparser.Module_argumentContext) any {
	return v.VisitChildren(ctx)
}

func (v *visitor) VisitKeyword(ctx *sqliteparser.KeywordContext) any {
	return v.VisitChildren(ctx)
}

func (v *visitor) VisitName(ctx *sqliteparser.NameContext) any {
	return v.VisitChildren(ctx)
}

func (v *visitor) VisitFunction_name(ctx *sqliteparser.Function_nameContext) any {
	return v.VisitChildren(ctx)
}

func (v *visitor) VisitSchema_name(ctx *sqliteparser.Schema_nameContext) any {
	return v.VisitChildren(ctx)
}

func (v *visitor) VisitTable_name(ctx *sqliteparser.Table_nameContext) any {
	return v.VisitChildren(ctx)
}

func (v *visitor) VisitTable_or_index_name(ctx *sqliteparser.Table_or_index_nameContext) any {
	return v.VisitChildren(ctx)
}

func (v *visitor) VisitColumn_name(ctx *sqliteparser.Column_nameContext) any {
	return v.VisitChildren(ctx)
}

func (v *visitor) VisitCollation_name(ctx *sqliteparser.Collation_nameContext) any {
	return v.VisitChildren(ctx)
}

func (v *visitor) VisitForeign_table(ctx *sqliteparser.Foreign_tableContext) any {
	return v.VisitChildren(ctx)
}

func (v *visitor) VisitIndex_name(ctx *sqliteparser.Index_nameContext) any {
	return v.VisitChildren(ctx)
}

func (v *visitor) VisitTrigger_name(ctx *sqliteparser.Trigger_nameContext) any {
	return v.VisitChildren(ctx)
}

func (v *visitor) VisitView_name(ctx *sqliteparser.View_nameContext) any {
	return v.VisitChildren(ctx)
}

func (v *visitor) VisitModule_name(ctx *sqliteparser.Module_nameContext) any {
	return v.VisitChildren(ctx)
}

func (v *visitor) VisitPragma_name(ctx *sqliteparser.Pragma_nameContext) any {
	return v.VisitChildren(ctx)
}

func (v *visitor) VisitSavepoint_name(ctx *sqliteparser.Savepoint_nameContext) any {
	return v.VisitChildren(ctx)
}

func (v *visitor) VisitTable_alias(ctx *sqliteparser.Table_aliasContext) any {
	return v.VisitChildren(ctx)
}

func (v *visitor) VisitTransaction_name(ctx *sqliteparser.Transaction_nameContext) any {
	return v.VisitChildren(ctx)
}

func (v *visitor) VisitWindow_name(ctx *sqliteparser.Window_nameContext) any {
	return v.VisitChildren(ctx)
}

func (v *visitor) VisitAlias(ctx *sqliteparser.AliasContext) any {
	return v.VisitChildren(ctx)
}

func (v *visitor) VisitFilename(ctx *sqliteparser.FilenameContext) any {
	return v.VisitChildren(ctx)
}

func (v *visitor) VisitBase_window_name(ctx *sqliteparser.Base_window_nameContext) any {
	return v.VisitChildren(ctx)
}

func (v *visitor) VisitSimple_func(ctx *sqliteparser.Simple_funcContext) any {
	return v.VisitChildren(ctx)
}

func (v *visitor) VisitAggregate_func(ctx *sqliteparser.Aggregate_funcContext) any {
	return v.VisitChildren(ctx)
}

func (v *visitor) VisitWindow_func(ctx *sqliteparser.Window_funcContext) any {
	return v.VisitChildren(ctx)
}

func (v *visitor) VisitTable_function_name(ctx *sqliteparser.Table_function_nameContext) any {
	return v.VisitChildren(ctx)
}

func (v *visitor) VisitIdentifier(ctx *sqliteparser.IdentifierContext) any {
	v.quoteIdentifier(ctx)
	return v.VisitChildren(ctx)
}
