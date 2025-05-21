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
	antlrhelpers "github.com/stephenafamo/bob/gen/bobgen-helpers/parser/antlrhelpers"
	"github.com/stephenafamo/bob/internal"
	sqliteparser "github.com/stephenafamo/sqlparser/sqlite"
)

var _ sqliteparser.SQLiteParserVisitor = &visitor{}

func NewVisitor(db tables) *visitor {
	return &visitor{
		Visitor: antlrhelpers.Visitor[any, IndexExtra]{
			DB:        db,
			Names:     make(map[NodeKey]string),
			Infos:     make(map[NodeKey]NodeInfo),
			Functions: defaultFunctions,
			Atom:      &atomic.Int64{},
		},
	}
}

type visitor struct {
	antlrhelpers.Visitor[any, IndexExtra]
}

func (v *visitor) Visit(tree antlr.ParseTree) any { return tree.Accept(v) }

func (v *visitor) VisitChildren(ctx antlr.RuleNode) any {
	if v.Err != nil {
		v.Err = fmt.Errorf("visiting children: %w", v.Err)
		return nil
	}

	for i, child := range ctx.GetChildren() {
		if tree, ok := child.(antlr.ParseTree); ok {
			tree.Accept(v)
		}

		if v.Err != nil {
			v.Err = fmt.Errorf("visiting child %d: %w", i, v.Err)
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
	v.StmtRules = append(v.StmtRules, internal.Replace(
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
	allresp := make([]StmtInfo, len(stmts))

	for i, stmt := range stmts {
		for _, child := range stmt.GetChildren() {
			if _, isTerminal := child.(antlr.TerminalNode); isTerminal {
				continue
			}

			v.Sources = nil
			v.StmtRules = slices.Clone(v.BaseRules)
			v.Atom = &atomic.Int64{}

			resp := child.(antlr.ParseTree).Accept(v)
			if v.Err != nil {
				v.Err = fmt.Errorf("stmt %d: %w", i, v.Err)
				return nil
			}

			info, ok := resp.([]ReturnColumn)
			if !ok {
				v.Err = fmt.Errorf("stmt %d: could not get columns, got %T", i, resp)
				return nil
			}

			var imports [][]string
			queryType := bob.QueryTypeUnknown
			mods := &strings.Builder{}

			switch child := child.(type) {
			case *sqliteparser.Select_stmtContext:
				queryType = bob.QueryTypeSelect
				imports = v.modSelect_stmt(child, mods)
			case *sqliteparser.Insert_stmtContext:
				queryType = bob.QueryTypeInsert
				v.modInsert_stmt(child, mods)
			case *sqliteparser.Update_stmtContext:
				queryType = bob.QueryTypeUpdate
				v.modUpdate_stmt(child, mods)
			case *sqliteparser.Delete_stmtContext:
				queryType = bob.QueryTypeDelete
				v.modDelete_stmt(child, mods)
			}

			allresp[i] = StmtInfo{
				QueryType: queryType,
				Node:      stmt,
				Columns:   info,
				EditRules: slices.Clone(v.StmtRules),
				Comment:   v.getCommentToLeft(stmt),
				Mods:      mods,
				Imports:   imports,
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
	// Defer reset the source list
	initialLen := len(v.Sources)
	defer func(l int) {
		v.Sources = v.Sources[:l]
	}(len(v.Sources))

	v.addSourcesFromWithClause(ctx.With_clause())
	if v.Err != nil {
		v.Err = fmt.Errorf("with clause: %w", v.Err)
		return nil
	}

	table := ctx.Qualified_table_name()
	tableName := getName(table.Table_name())
	tableSource := v.getSourceFromTable(table)
	v.Sources = append(v.Sources, tableSource)

	v.VisitChildren(ctx)
	if v.Err != nil {
		v.Err = fmt.Errorf("insert stmt: %w", v.Err)
		return nil
	}

	returning := ctx.Returning_clause()
	if returning == nil {
		return []ReturnColumn{}
	}

	// Reset the sources to the original length
	v.Sources = v.Sources[:initialLen]
	// Only add the table source for the returning clause
	tableSource.Name = tableName
	v.Sources = append(v.Sources, tableSource)

	return v.getSourceFromColumns(returning.AllResult_column()).Columns
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
	v.VisitChildren(ctx)
	if v.Err != nil {
		return nil
	}

	lhsType := v.Infos[antlrhelpers.Key(ctx.GetLhs())].Type
	rhsType := v.Infos[antlrhelpers.Key(ctx.GetRhs())].Type

	typ := []NodeType{knownTypeNotNull("INTEGER"), knownTypeNotNull("REAL")}

	switch {
	case len(lhsType) == 1 && len(rhsType) == 1:
		typ = []NodeType{knownTypeNotNull("REAL")}
		lhs := lhsType[0]
		rhs := rhsType[0]
		if lhs.DBType == "INTEGER" &&
			rhs.DBType == "INTEGER" {
			typ = []NodeType{knownType("INTEGER", antlrhelpers.AnyNullable(lhs.Nullable, rhs.Nullable))}
		}

	case len(lhsType) == 1 && len(rhsType) == 0:
		typ = []NodeType{knownTypeNotNull("REAL")}
		lhs := lhsType[0]
		if lhs.DBType == "INTEGER" {
			typ = []NodeType{knownType("INTEGER", lhs.Nullable)}
		}

	case len(lhsType) == 0 && len(rhsType) == 1:
		typ = []NodeType{knownTypeNotNull("REAL")}
		rhs := rhsType[0]
		if rhs.DBType == "INTEGER" {
			typ = []NodeType{knownType("INTEGER", rhs.Nullable)}
		}
	}

	v.UpdateInfo(NodeInfo{
		Node:            ctx,
		ExprDescription: "Arithmetic",
		Type:            typ,
	})

	v.UpdateInfo(NodeInfo{
		Node:            ctx.GetLhs(),
		ExprDescription: "Arithmetic LHS",
		Type:            typ,
	})

	v.UpdateInfo(NodeInfo{
		Node:            ctx.GetRhs(),
		ExprDescription: "Arithmetic RHS",
		Type:            typ,
	})

	return nil
}

func (v *visitor) VisitExpr_json_extract_string(ctx *sqliteparser.Expr_json_extract_stringContext) any {
	v.VisitChildren(ctx)
	if v.Err != nil {
		return nil
	}

	v.UpdateInfo(NodeInfo{
		Node:            ctx,
		ExprDescription: "JSON->>",
		Type:            []NodeType{knownTypeNull("")},
	})

	v.UpdateInfo(NodeInfo{
		Node:            ctx.GetLhs(),
		ExprDescription: "JSON->> LHS",
		Type:            []NodeType{knownTypeNotNull("JSON")},
	})

	v.UpdateInfo(NodeInfo{
		Node:            ctx.GetRhs(),
		ExprDescription: "JSON->> RHS",
		Type: []NodeType{
			knownTypeNotNull("TEXT"),
			knownTypeNotNull("INTEGER"),
		},
	})

	return nil
}

func (v *visitor) VisitExpr_raise(ctx *sqliteparser.Expr_raiseContext) any {
	return v.VisitChildren(ctx)
}

func (v *visitor) VisitExpr_bool(ctx *sqliteparser.Expr_boolContext) any {
	v.VisitChildren(ctx)
	if v.Err != nil {
		return nil
	}

	v.UpdateInfo(NodeInfo{
		Node:            ctx,
		ExprDescription: "AND/OR",
		Type:            []NodeType{knownTypeNotNull("BOOLEAN")},
	})

	v.UpdateInfo(NodeInfo{
		Node:            ctx.GetLhs(),
		ExprDescription: "AND/OR LHS",
		Type:            []NodeType{knownTypeNull("BOOLEAN")},
	})

	v.UpdateInfo(NodeInfo{
		Node:            ctx.GetRhs(),
		ExprDescription: "AND/OR RHS",
		Type:            []NodeType{knownTypeNull("BOOLEAN")},
	})

	return nil
}

func (v *visitor) VisitExpr_is(ctx *sqliteparser.Expr_isContext) any {
	v.VisitChildren(ctx)
	if v.Err != nil {
		return nil
	}

	v.UpdateInfo(NodeInfo{
		Node:            ctx,
		ExprDescription: "IS",
		Type:            []NodeType{knownTypeNotNull("BOOLEAN")},
	})

	v.UpdateInfo(NodeInfo{
		Node:            ctx.GetLhs(),
		ExprDescription: "IS LHS",
		ExprRef:         ctx.GetRhs(),
		Type:            []NodeType{knownTypeNull("")},
	})

	v.UpdateInfo(NodeInfo{
		Node:            ctx.GetRhs(),
		ExprDescription: "IS RHS",
		ExprRef:         ctx.GetLhs(),
		Type:            []NodeType{knownTypeNull("")},
	})

	return nil
}

func (v *visitor) VisitExpr_concat(ctx *sqliteparser.Expr_concatContext) any {
	v.VisitChildren(ctx)
	if v.Err != nil {
		return nil
	}

	v.UpdateInfo(NodeInfo{
		Node:            ctx,
		ExprDescription: "Concat",
		Type:            []NodeType{knownTypeNotNull("TEXT")},
	})

	v.UpdateInfo(NodeInfo{
		Node:            ctx.GetLhs(),
		ExprDescription: "Concat LHS",
		Type:            []NodeType{knownTypeNotNull("TEXT")},
	})

	v.UpdateInfo(NodeInfo{
		Node:            ctx.GetRhs(),
		ExprDescription: "Concat RHS",
		Type:            []NodeType{knownTypeNotNull("TEXT")},
	})

	return nil
}

func (v *visitor) VisitExpr_list(ctx *sqliteparser.Expr_listContext) any {
	v.VisitChildren(ctx)
	if v.Err != nil {
		return nil
	}

	exprs := ctx.AllExpr()
	if len(exprs) == 1 {
		v.MaybeSetNodeName(ctx, v.GetName(exprs[0]))
	}

	v.StmtRules = append(v.StmtRules, internal.RecordPoints(
		ctx.GetStart().GetStart(),
		ctx.GetStop().GetStop(),
		func(start, end int) error {
			v.SetGroup(ctx)
			v.UpdateInfo(NodeInfo{
				Node:            ctx,
				ExprDescription: "LIST",
				EditedPosition:  [2]int{start, end},
			})
			return nil
		},
	)...)

	return nil
}

func (v *visitor) VisitExpr_in(ctx *sqliteparser.Expr_inContext) any {
	v.VisitChildren(ctx)
	if v.Err != nil {
		return nil
	}

	rhsExpr := ctx.GetRhsExpr()
	rhsList, rhsIsList := rhsExpr.(*sqliteparser.Expr_listContext)
	if !rhsIsList {
		return nil
	}
	rhsChildren := rhsList.AllExpr()

	// If there is only one child, it can be multiple
	canBeMultiple := len(rhsChildren) == 1

	lhs := ctx.GetLhs()
	lhsList, lhsIsList := lhs.(*sqliteparser.Expr_listContext)
	var lhsChildren []sqliteparser.IExprContext
	if lhsIsList {
		lhsChildren = lhsList.AllExpr()
	}
	lhsChildNames := make([]string, len(lhsChildren))
	for i, child := range lhsChildren {
		lhsChildNames[i] = v.GetName(child)
	}

	lhsName := v.GetName(lhs)

	for _, child := range rhsChildren {
		v.UpdateInfo(NodeInfo{
			Node:                 child,
			ExprDescription:      "IN RHS",
			ExprRef:              lhs,
			IgnoreRefNullability: true,
			CanBeMultiple:        canBeMultiple,
		})
		v.MaybeSetNodeName(child, lhsName)

		childList, childIsList := child.(*sqliteparser.Expr_listContext)
		if !childIsList {
			continue
		}
		grandChildren := childList.AllExpr()

		if len(grandChildren) != len(lhsChildren) {
			v.Err = fmt.Errorf("IN: list length mismatch %d != %d", len(grandChildren), len(lhsChildren))
			return nil
		}

		for i, grandChild := range grandChildren {
			v.UpdateInfo(NodeInfo{
				Node:                 grandChild,
				ExprDescription:      "IN RHS List",
				ExprRef:              lhsChildren[i],
				IgnoreRefNullability: true,
			})
			v.MaybeSetNodeName(grandChild, lhsChildNames[i])
		}

		v.StmtRules = append(v.StmtRules, internal.RecordPoints(
			childList.GetStart().GetStart(),
			childList.GetStop().GetStop(),
			func(start, end int) error {
				v.UpdateInfo(NodeInfo{
					Node:           childList,
					EditedPosition: [2]int{start, end},
				})
				return nil
			},
		)...)
	}

	return nil
}

func (v *visitor) VisitExpr_collate(ctx *sqliteparser.Expr_collateContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by SQLiteParser#expr_modulo.
func (v *visitor) VisitExpr_modulo(ctx *sqliteparser.Expr_moduloContext) any {
	v.VisitChildren(ctx)
	if v.Err != nil {
		return nil
	}

	v.UpdateInfo(NodeInfo{
		Node:            ctx,
		ExprDescription: "Modulo",
		Type:            []NodeType{knownTypeNotNull("INTEGER")},
	})

	v.UpdateInfo(NodeInfo{
		Node:            ctx.GetLhs(),
		ExprDescription: "Modulo LHS",
		Type:            []NodeType{knownTypeNotNull("INTEGER")},
	})

	v.UpdateInfo(NodeInfo{
		Node:            ctx.GetRhs(),
		ExprDescription: "Modulo RHS",
		Type:            []NodeType{knownTypeNotNull("INTEGER")},
	})

	return nil
}

func (v *visitor) VisitExpr_qualified_column_name(ctx *sqliteparser.Expr_qualified_column_nameContext) any {
	v.MaybeSetNodeName(
		ctx,
		getName(ctx.Column_name()),
	)

	v.VisitChildren(ctx)
	if v.Err != nil {
		return nil
	}

	v.UpdateInfo(NodeInfo{
		Node:            ctx,
		ExprDescription: "Qualified",
		Type:            makeRef(v.Sources, ctx),
	})

	return nil
}

func (v *visitor) VisitExpr_match(ctx *sqliteparser.Expr_matchContext) any {
	v.VisitChildren(ctx)
	if v.Err != nil {
		return nil
	}

	v.UpdateInfo(NodeInfo{
		Node:            ctx,
		ExprDescription: "Match",
		Type:            []NodeType{knownTypeNotNull("BOOLEAN")},
	})

	v.UpdateInfo(NodeInfo{
		Node:            ctx.GetRhs(),
		ExprDescription: "Modulo RHS",
		Type:            []NodeType{knownTypeNotNull("TEXT")},
	})

	return nil
}

func (v *visitor) VisitExpr_like(ctx *sqliteparser.Expr_likeContext) any {
	v.VisitChildren(ctx)
	if v.Err != nil {
		return nil
	}

	v.UpdateInfo(NodeInfo{
		Node:            ctx,
		ExprDescription: "LIKE",
		Type:            []NodeType{knownTypeNotNull("BOOLEAN")},
	})

	v.UpdateInfo(NodeInfo{
		Node:            ctx.GetLhs(),
		ExprDescription: "Like LHS",
		Type:            []NodeType{knownTypeNotNull("TEXT")},
	})

	v.UpdateInfo(NodeInfo{
		Node:            ctx.GetRhs(),
		ExprDescription: "Like RHS",
		Type:            []NodeType{knownTypeNotNull("TEXT")},
	})

	return nil
}

func (v *visitor) VisitExpr_null_comp(ctx *sqliteparser.Expr_null_compContext) any {
	v.VisitChildren(ctx)
	if v.Err != nil {
		return nil
	}

	v.UpdateInfo(NodeInfo{
		Node:            ctx,
		ExprDescription: "NULL Comparison",
		Type:            []NodeType{knownTypeNotNull("BOOLEAN")},
	})

	v.UpdateInfo(NodeInfo{
		Node:            ctx.Expr(),
		ExprDescription: "NULL Comparison Expr",
		Type:            []NodeType{knownTypeNotNull("")},
	})

	return nil
}

func (v *visitor) VisitExpr_json_extract_json(ctx *sqliteparser.Expr_json_extract_jsonContext) any {
	v.VisitChildren(ctx)
	if v.Err != nil {
		return nil
	}

	v.UpdateInfo(NodeInfo{
		Node:            ctx,
		ExprDescription: "JSON->",
		Type:            []NodeType{knownTypeNotNull("JSON")},
	})

	v.UpdateInfo(NodeInfo{
		Node:            ctx.GetLhs(),
		ExprDescription: "JSON-> LHS",
		Type:            []NodeType{knownTypeNotNull("JSON")},
	})

	v.UpdateInfo(NodeInfo{
		Node:            ctx.GetRhs(),
		ExprDescription: "JSON-> RHS",
		Type:            []NodeType{knownTypeNotNull("TEXT")},
	})

	return nil
}

func (v *visitor) VisitExpr_exists_select(ctx *sqliteparser.Expr_exists_selectContext) any {
	return v.VisitChildren(ctx)
}

func (v *visitor) VisitExpr_comparison(ctx *sqliteparser.Expr_comparisonContext) any {
	v.VisitChildren(ctx)
	if v.Err != nil {
		return nil
	}

	v.UpdateInfo(NodeInfo{
		Node:            ctx,
		ExprDescription: "Comparison",
		Type:            []NodeType{knownTypeNotNull("BOOLEAN")},
	})

	v.UpdateInfo(NodeInfo{
		Node:                 ctx.GetLhs(),
		ExprDescription:      "Comparison LHS",
		ExprRef:              ctx.GetRhs(),
		IgnoreRefNullability: true,
	})

	v.UpdateInfo(NodeInfo{
		Node:                 ctx.GetRhs(),
		ExprDescription:      "Comparison RHS",
		ExprRef:              ctx.GetLhs(),
		IgnoreRefNullability: true,
	})

	v.MatchNodeNames(ctx.GetLhs(), ctx.GetRhs())

	return nil
}

func (v *visitor) VisitExpr_literal(ctx *sqliteparser.Expr_literalContext) any {
	v.VisitChildren(ctx)
	if v.Err != nil {
		return nil
	}

	var DBType NodeType

	typ := ctx.Literal_value().GetLiteralType().GetTokenType()
	switch typ {
	case sqliteparser.SQLiteParserNUMERIC_LITERAL:
		v.MaybeSetNodeName(ctx, ctx.GetText())

		if strings.ContainsAny(ctx.GetText(), ".eE") {
			DBType = knownTypeNotNull("REAL")
			break
		}

		text := strings.ReplaceAll(ctx.GetText(), "_", "")
		if len(text) < 2 {
			DBType = knownTypeNotNull("INTEGER")
			break
		}

		base := 10

		if strings.EqualFold(text[0:2], "0x") {
			text = text[2:]
			base = 16
		}

		_, err := strconv.ParseInt(text, base, 64)
		if err == nil {
			DBType = knownTypeNotNull("INTEGER")
			break
		}

		if errors.Is(err, strconv.ErrRange) {
			DBType = knownTypeNotNull("REAL")
			break
		}

		v.Err = fmt.Errorf("cannot parse numeric integer: %s", ctx.GetText())
		return nil

	case sqliteparser.SQLiteParserSTRING_LITERAL:
		DBType = knownTypeNotNull("TEXT")
		txt := strings.ReplaceAll(ctx.GetText(), "'", "")
		v.MaybeSetNodeName(ctx, txt)

	case sqliteparser.SQLiteParserBLOB_LITERAL:
		DBType = knownTypeNotNull("BLOB")
		v.MaybeSetNodeName(ctx, "BLOB")

	case sqliteparser.SQLiteParserNULL_:
		DBType = knownTypeNull("")
		v.MaybeSetNodeName(ctx, "NULL")

	case sqliteparser.SQLiteParserTRUE_,
		sqliteparser.SQLiteParserFALSE_:
		DBType = knownTypeNotNull("BOOLEAN")
		v.MaybeSetNodeName(ctx, ctx.GetText())

	case sqliteparser.SQLiteParserCURRENT_TIME_,
		sqliteparser.SQLiteParserCURRENT_DATE_,
		sqliteparser.SQLiteParserCURRENT_TIMESTAMP_:
		DBType = knownTypeNotNull("DATETIME")
		v.MaybeSetNodeName(ctx, ctx.GetText()[:len(ctx.GetText())-1])

	default:
		v.Err = fmt.Errorf("unknown literal type: %d", typ)
		return nil
	}

	info := NodeInfo{
		Node:            ctx,
		ExprDescription: "Literal",
	}

	if len(DBType.DBType) > 0 {
		info.Type = []NodeType{DBType}
	}

	v.UpdateInfo(info)

	return nil
}

func (v *visitor) VisitExpr_cast(ctx *sqliteparser.Expr_castContext) any {
	v.VisitChildren(ctx)
	if v.Err != nil {
		return nil
	}

	v.UpdateInfo(NodeInfo{
		Node:            ctx,
		ExprDescription: "CAST",
		Type:            []NodeType{knownType(ctx.Type_name().GetText(), antlrhelpers.NotNullable)},
	})

	return nil
}

func (v *visitor) VisitExpr_string_op(ctx *sqliteparser.Expr_string_opContext) any {
	v.VisitChildren(ctx)
	if v.Err != nil {
		return nil
	}

	v.UpdateInfo(NodeInfo{
		Node:            ctx,
		ExprDescription: "String OP",
		Type:            []NodeType{knownTypeNotNull("BOOLEAN")},
	})

	v.UpdateInfo(NodeInfo{
		Node:            ctx.GetLhs(),
		ExprDescription: "String OP LHS",
		Type:            []NodeType{knownTypeNotNull("TEXT")},
	})

	v.UpdateInfo(NodeInfo{
		Node:            ctx.GetRhs(),
		ExprDescription: "String OP RHS",
		Type:            []NodeType{knownTypeNotNull("TEXT")},
	})

	return nil
}

func (v *visitor) VisitExpr_between(ctx *sqliteparser.Expr_betweenContext) any {
	return v.VisitChildren(ctx)
}

func (v *visitor) VisitExpr_bitwise(ctx *sqliteparser.Expr_bitwiseContext) any {
	v.VisitChildren(ctx)
	if v.Err != nil {
		return nil
	}

	v.UpdateInfo(NodeInfo{
		Node:            ctx,
		ExprDescription: "Bitwise",
		Type:            []NodeType{knownTypeNotNull("INTEGER")},
	})

	v.UpdateInfo(NodeInfo{
		Node:            ctx.GetLhs(),
		ExprDescription: "Bitwise LHS",
		Type:            []NodeType{knownTypeNotNull("INTEGER")},
	})

	v.UpdateInfo(NodeInfo{
		Node:            ctx.GetRhs(),
		ExprDescription: "Bitwise RHS",
		Type:            []NodeType{knownTypeNotNull("INTEGER")},
	})

	return nil
}

func (v *visitor) VisitExpr_unary(ctx *sqliteparser.Expr_unaryContext) any {
	v.VisitChildren(ctx)
	if v.Err != nil {
		return nil
	}

	tokenTyp := ctx.Unary_operator().GetOperator().GetTokenType()
	switch tokenTyp {
	case sqliteparser.SQLiteParserPLUS:
		// Returns the same type as the operand
		v.UpdateInfo(NodeInfo{
			Node:            ctx,
			ExprDescription: "Unary Plus",
			ExprRef:         ctx.Expr(),
		})

		v.UpdateInfo(NodeInfo{
			Node:            ctx.Expr(),
			ExprDescription: "Unary Plus Expr",
			ExprRef:         ctx,
		})

	case sqliteparser.SQLiteParserMINUS:
		// Always INTEGER, should be used with a numeric literal
		v.UpdateInfo(NodeInfo{
			Node:            ctx,
			ExprDescription: "Unary Minus",
			Type:            []NodeType{knownTypeNotNull("INTEGER"), knownTypeNotNull("REAL")},
		})

		v.UpdateInfo(NodeInfo{
			Node:            ctx.Expr(),
			ExprDescription: "Unary Minus Expr",
			Type:            []NodeType{knownTypeNotNull("INTEGER"), knownTypeNotNull("REAL")},
		})

	case sqliteparser.SQLiteParserTILDE:
		// Bitwise NOT
		// Always INTEGER, should be used with a numeric literal
		v.UpdateInfo(NodeInfo{
			Node:            ctx,
			ExprDescription: "Unary Tilde",
			Type:            []NodeType{knownTypeNotNull("INTEGER")},
		})

		v.UpdateInfo(NodeInfo{
			Node:            ctx.Expr(),
			ExprDescription: "Unary Tilde Expr",
			Type:            []NodeType{knownTypeNotNull("INTEGER")},
		})

	case sqliteparser.SQLiteParserNOT_:
		// Returns a BOOLEAN (should technically only be used with a boolean expression)
		v.UpdateInfo(NodeInfo{
			Node:            ctx,
			ExprDescription: "Unary NOT",
			Type:            []NodeType{knownTypeNotNull("BOOLEAN")},
		})

		v.UpdateInfo(NodeInfo{
			Node:            ctx.Expr(),
			ExprDescription: "Unary NOT Expr",
			Type:            []NodeType{knownTypeNotNull("BOOLEAN")},
		})
	}

	return nil
}

func (v *visitor) VisitExpr_bind(ctx *sqliteparser.Expr_bindContext) any {
	v.VisitChildren(ctx)
	if v.Err != nil {
		return nil
	}

	v.SetArg(ctx)
	info := NodeInfo{
		Node:            ctx,
		ExprDescription: "Bind",
		ArgKey:          ctx.GetText()[1:],
	}

	parent, ok := ctx.GetParent().(*sqliteparser.Expr_castContext)
	if ok {
		info.ExprRef = parent
	}

	v.SetArg(ctx)
	v.UpdateInfo(info)

	// So it does not refer to the same atomic
	a := v.Atom
	v.StmtRules = append(v.StmtRules, internal.EditCallback(
		internal.ReplaceFromFunc(
			ctx.GetStart().GetStart(), ctx.GetStop().GetStop(),
			func() string {
				return fmt.Sprintf("?%d", a.Add(1))
			},
		),
		func(start, end int, _, _ string) error {
			v.UpdateInfo(NodeInfo{
				Node:           ctx,
				EditedPosition: [2]int{start, end},
			})
			return nil
		}),
	)

	return nil
}

func (v *visitor) VisitExpr_simple_func(ctx *sqliteparser.Expr_simple_funcContext) any {
	v.VisitChildren(ctx)
	if v.Err != nil {
		v.Err = fmt.Errorf("simple function invocation: %w", v.Err)
		return nil
	}

	args := ctx.AllExpr()
	argTypes := make([]string, len(args))
	missingTypes := make([]bool, len(args))
	nullable := make([]func() bool, len(args))
	for i, arg := range args {
		argTypes[i] = v.Infos[antlrhelpers.Key(arg)].Type.ConfirmedDBType()
		nullable[i] = v.Infos[antlrhelpers.Key(arg)].Type.Nullable
		if argTypes[i] == "" {
			missingTypes[i] = true
		}
	}

	funcName := getName(ctx.Simple_func())
	funcDef, err := antlrhelpers.GetFunctionType(v.Functions, funcName, argTypes)
	if err != nil {
		v.Err = fmt.Errorf("simple function invocation: %w", err)
		return nil
	}

	for i, arg := range args {
		if missingTypes[i] {
			v.UpdateInfo(NodeInfo{
				Node:            arg,
				ExprDescription: "Function Arg",
				Type: []NodeType{knownType(
					funcDef.ArgType(i),
					func() bool { return funcDef.ShouldArgsBeNullable },
				)},
			})
		}
	}

	info := NodeInfo{
		Node:            ctx,
		ExprDescription: "Function Arg",
		Type: []NodeType{knownType(
			funcDef.ReturnType,
			antlrhelpers.AnyNullable(nullable...),
		)},
	}

	if funcDef.CalcNullable != nil {
		info.Type[0].NullableF = funcDef.CalcNullable(nullable...)
	}

	v.UpdateInfo(info)
	v.MaybeSetNodeName(ctx, funcName)

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
	// Defer reset the source list
	initialLen := len(v.Sources)
	defer func(l int) {
		v.Sources = v.Sources[:l]
	}(len(v.Sources))

	v.addSourcesFromWithClause(ctx.With_clause())
	if v.Err != nil {
		v.Err = fmt.Errorf("with clause: %w", v.Err)
		return nil
	}

	tableName := getName(ctx.Table_name())
	tableSource := v.getSourceFromTable(ctx)
	v.Sources = append(v.Sources, tableSource)

	v.VisitChildren(ctx)
	if v.Err != nil {
		v.Err = fmt.Errorf("insert stmt: %w", v.Err)
		return nil
	}

	columns := ctx.AllColumn_name()
	colNames := make([]string, len(columns))
	for i := range columns {
		colNames[i] = getName(columns[i])
	}
	if len(colNames) == 0 {
		colNames = make([]string, len(tableSource.Columns))
		for i := range tableSource.Columns {
			colNames[i] = tableSource.Columns[i].Name
		}
	}

	if values := ctx.Values_clause(); values != nil {
		rows := values.AllValue_row()
		for _, row := range rows {
			v.MaybeSetNodeName(row, tableName)
			v.StmtRules = append(v.StmtRules, internal.RecordPoints(
				row.GetStart().GetStart(),
				row.GetStop().GetStop(),
				func(start, end int) error {
					v.SetGroup(row)
					v.UpdateInfo(NodeInfo{
						Node:            row,
						ExprDescription: "ROW",
						EditedPosition:  [2]int{start, end},
						CanBeMultiple:   len(rows) == 1,
					})
					return nil
				},
			)...)

			for valIndex, value := range row.AllExpr() {
				v.UpdateInfo(NodeInfo{
					Node:            value,
					ExprDescription: "ROW Value",
					Type: []NodeType{getColumnType(
						v.DB,
						getName(ctx.Schema_name()),
						tableName,
						colNames[valIndex],
					)},
				})

				if valIndex < len(colNames) {
					v.MaybeSetNodeName(value, colNames[valIndex])
				}
			}
		}
	}

	returning := ctx.Returning_clause()
	if returning == nil {
		return []ReturnColumn{}
	}

	// Reset the sources to the original length
	v.Sources = v.Sources[:initialLen]
	// Only add the table source for the returning clause
	tableSource.Name = tableName
	v.Sources = append(v.Sources, tableSource)

	return v.getSourceFromColumns(returning.AllResult_column()).Columns
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
	// Defer reset the source list
	defer func(l int) {
		v.Sources = v.Sources[:l]
	}(len(v.Sources))

	v.addSourcesFromWithClause(ctx.With_clause())
	if v.Err != nil {
		v.Err = fmt.Errorf("with clause: %w", v.Err)
		return nil
	}

	// Should return a source
	// Use the first select core to get the columns
	source := ctx.Select_core().Accept(v).(QuerySource)
	if v.Err != nil {
		v.Err = fmt.Errorf("select core: %w", v.Err)
		return nil
	}

	for i, compound := range ctx.AllCompound_select() {
		coreSource := compound.Select_core().Accept(v).(QuerySource)
		if v.Err != nil {
			v.Err = fmt.Errorf("compound core %d: %w", i, v.Err)
			return nil
		}

		if len(source.Columns) != len(coreSource.Columns) {
			v.Err = fmt.Errorf(
				"select core %d: column count mismatch %d != %d",
				i, len(source.Columns), len(coreSource.Columns))
			return nil
		}

		for i, col := range source.Columns {
			matchingTypes := col.Type.Match(coreSource.Columns[i].Type)

			if len(source.Columns[i].Type) == 0 {
				v.Err = fmt.Errorf(
					"select core %d: column %d type mismatch:\n%v\n%v",
					i, i, col.Type, coreSource.Columns[i].Type)
				return nil
			}

			source.Columns[i].Type = matchingTypes
		}
	}

	v.Sources = append(v.Sources, source) // needed for order by and limit
	if order := ctx.Order_by_stmt(); order != nil {
		order.Accept(v)
		if v.Err != nil {
			v.Err = fmt.Errorf("order by: %w", v.Err)
			return nil
		}
	}

	if limit := ctx.Limit_stmt(); limit != nil {
		limit.Accept(v)
		if v.Err != nil {
			v.Err = fmt.Errorf("limit: %w", v.Err)
			return nil
		}
	}

	return source.Columns
}

// Should return a query source
func (v *visitor) VisitSelect_core(ctx *sqliteparser.Select_coreContext) any {
	defer func(l int) {
		v.Sources = v.Sources[:l]
	}(len(v.Sources))

	v.addSourcesFromFrom_item(ctx.From_item())
	if v.Err != nil {
		v.Err = fmt.Errorf("from item: %w", v.Err)
		return QuerySource{}
	}

	v.VisitChildren(ctx)
	if v.Err != nil {
		v.Err = fmt.Errorf("select core: %w", v.Err)
		return QuerySource{}
	}

	return v.getSourceFromColumns(ctx.AllResult_column())
}

func (v *visitor) VisitResult_column(ctx *sqliteparser.Result_columnContext) any {
	return v.VisitChildren(ctx)
}

func (v *visitor) VisitFrom_item(ctx *sqliteparser.From_itemContext) any {
	return nil // do not visit children automatically
}

func (v *visitor) VisitJoin_operator(ctx *sqliteparser.Join_operatorContext) any {
	return v.VisitChildren(ctx)
}

func (v *visitor) VisitJoin_constraint(ctx *sqliteparser.Join_constraintContext) any {
	return v.VisitChildren(ctx)
}

func (v *visitor) VisitTable_or_subquery(ctx *sqliteparser.Table_or_subqueryContext) any {
	return v.VisitChildren(ctx)
}

func (v *visitor) VisitCompound_select(ctx *sqliteparser.Compound_selectContext) any {
	return v.VisitChildren(ctx)
}

func (v *visitor) VisitCompound_operator(ctx *sqliteparser.Compound_operatorContext) any {
	return v.VisitChildren(ctx)
}

func (v *visitor) VisitUpdate_stmt(ctx *sqliteparser.Update_stmtContext) any {
	// Defer reset the source list
	initialLen := len(v.Sources)
	defer func(l int) {
		v.Sources = v.Sources[:l]
	}(len(v.Sources))

	v.addSourcesFromWithClause(ctx.With_clause())
	if v.Err != nil {
		v.Err = fmt.Errorf("with clause: %w", v.Err)
		return nil
	}

	table := ctx.Qualified_table_name()
	tableName := getName(table.Table_name())
	tableSource := v.getSourceFromTable(table)
	v.Sources = append(v.Sources, tableSource)

	v.addSourcesFromFrom_item(ctx.From_item())
	if v.Err != nil {
		v.Err = fmt.Errorf("from item: %w", v.Err)
		return QuerySource{}
	}

	v.VisitChildren(ctx)
	if v.Err != nil {
		v.Err = fmt.Errorf("update stmt: %w", v.Err)
		return nil
	}

	exprs := ctx.AllExpr()
	for i, nameOrList := range ctx.AllColumn_name_or_list() {
		nameExpr := nameOrList.Column_name()
		if nameExpr == nil {
			continue
		}
		colName := getName(nameExpr)
		expr := exprs[i]
		v.UpdateInfo(NodeInfo{
			Node:            expr,
			ExprDescription: "SET Expr",
			Type: []NodeType{getColumnType(
				v.DB,
				getName(table.Schema_name()),
				tableName,
				colName,
			)},
		})

		v.MaybeSetNodeName(expr, colName)
	}

	returning := ctx.Returning_clause()
	if returning == nil {
		return []ReturnColumn{}
	}

	// Reset the sources to the original length
	v.Sources = v.Sources[:initialLen]
	// Only add the table source for the returning clause
	tableSource.Name = tableName
	v.Sources = append(v.Sources, tableSource)

	return v.getSourceFromColumns(returning.AllResult_column()).Columns
}

func (v *visitor) VisitColumn_name_or_list(ctx *sqliteparser.Column_name_or_listContext) any {
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
	return v.VisitChildren(ctx)
}

func (v *visitor) VisitCommon_table_expression(ctx *sqliteparser.Common_table_expressionContext) any {
	return v.VisitChildren(ctx)
}

func (v *visitor) VisitWhere_stmt(ctx *sqliteparser.Where_stmtContext) any {
	return v.VisitChildren(ctx)
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
