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
	"github.com/stephenafamo/bob/gen/bobgen-helpers/parser/antlrhelpers"
	"github.com/stephenafamo/bob/internal"
	mysqlparser "github.com/stephenafamo/sqlparser/mysql"
)

var _ mysqlparser.MySqlParserVisitor = &visitor{}

func NewVisitor(db tables) *visitor {
	return &visitor{
		Visitor: antlrhelpers.Visitor[any, any]{
			DB:        db,
			Names:     make(map[NodeKey]string),
			Infos:     make(map[NodeKey]NodeInfo),
			Functions: defaultFunctions,
			Atom:      &atomic.Int64{},
		},
		querySources: make(map[antlrhelpers.NodeKey]QuerySource),
	}
}

type visitor struct {
	antlrhelpers.Visitor[any, any]
	querySources map[antlrhelpers.NodeKey]QuerySource
}

// Visit implements parser.MySqlParserVisitor.
func (v *visitor) Visit(tree antlr.ParseTree) any {
	return tree.Accept(v)
}

// VisitChildren implements parser.MySqlParserVisitor.
func (v *visitor) VisitChildren(node antlr.RuleNode) any {
	if v.Err != nil {
		v.Err = fmt.Errorf("visiting children: %w", v.Err)
		return nil
	}

	for i, child := range node.GetChildren() {
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

// VisitTerminal implements parser.MySqlParserVisitor.
func (v *visitor) VisitTerminal(node antlr.TerminalNode) any {
	token := node.GetSymbol()

	switch token.GetTokenType() {
	case mysqlparser.MySqlParserID:
		name := token.GetText()
		v.StmtRules = append(v.StmtRules, internal.Replace(
			token.GetStart(), token.GetStop(),
			fmt.Sprintf("`%s`", name),
		))
		v.MaybeSetName(antlrhelpers.NodeKey{
			Start: token.GetStart(), Stop: token.GetStop(),
		}, name)
	case mysqlparser.MySqlParserDOT_ID:
		name := token.GetText()[1:]
		v.StmtRules = append(v.StmtRules, internal.Replace(
			token.GetStart(), token.GetStop(),
			fmt.Sprintf(".`%s`", name),
		))
		v.MaybeSetName(antlrhelpers.NodeKey{
			Start: token.GetStart(), Stop: token.GetStop(),
		}, name)
	case mysqlparser.MySqlParserREVERSE_QUOTE_ID:
		name := token.GetText()
		name = name[1 : len(name)-1]
		v.MaybeSetName(antlrhelpers.NodeKey{
			Start: token.GetStart(), Stop: token.GetStop(),
		}, name)
	case mysqlparser.MySqlParserSTRING_LITERAL:
		name := token.GetText()
		name = name[1 : len(name)-1]
		v.MaybeSetName(antlrhelpers.NodeKey{
			Start: token.GetStart(), Stop: token.GetStop(),
		}, name)
	}

	literals := mysqlparser.MySqlLexerLexerStaticData.LiteralNames
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

// Visit a parse tree produced by MySqlParser#root.
func (v *visitor) VisitRoot(ctx *mysqlparser.RootContext) any {
	return ctx.SqlStatements().Accept(v)
}

// Visit a parse tree produced by MySqlParser#sqlStatements.
func (v *visitor) VisitSqlStatements(ctx *mysqlparser.SqlStatementsContext) any {
	stmts := ctx.AllSqlStatement()
	allresp := make([]StmtInfo, 0, len(stmts))

	for i, stmt := range stmts {
		dml := stmt.DmlStatement()
		if dml == nil {
			continue
		}

		onlyChild := dml.GetChild(0)

		v.Sources = nil
		v.StmtRules = slices.Clone(v.BaseRules)
		v.Atom = &atomic.Int64{}

		resp := onlyChild.(antlr.ParseTree).Accept(v)
		if v.Err != nil {
			v.Err = fmt.Errorf("stmt %d: %w", i, v.Err)
			return nil
		}

		var columns []ReturnColumn
		var imports [][]string
		queryType := bob.QueryTypeUnknown
		mods := &strings.Builder{}

		switch child := onlyChild.(type) {
		case *mysqlparser.InsertStatementContext:
			queryType = bob.QueryTypeInsert
			v.modInsertStatement(child, mods)
		case *mysqlparser.UpdateStatementContext:
			queryType = bob.QueryTypeUpdate
			v.modUpdateStatement(child, mods)
		case *mysqlparser.DeleteStatementContext:
			queryType = bob.QueryTypeDelete
			v.modDeleteStatement(child, mods)
		case mysqlparser.ISelectStatementContext:
			queryType = bob.QueryTypeSelect
			imports = v.modSelectStatement(child, mods)
			var ok bool
			columns, ok = resp.([]ReturnColumn)
			if !ok {
				v.Err = fmt.Errorf("stmt %d: could not get columns in select statement, got %T", i, resp)
				return nil
			}
		}

		allresp = append(allresp, StmtInfo{
			QueryType: queryType,
			Node:      stmt,
			Columns:   columns,
			EditRules: slices.Clone(v.StmtRules),
			Comment:   v.getCommentToLeft(stmt),
			Mods:      mods,
			Imports:   imports,
		})
	}

	return allresp
}

// Visit a parse tree produced by MySqlParser#sqlStatement.
func (v *visitor) VisitSqlStatement(ctx *mysqlparser.SqlStatementContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#emptyStatement_.
func (v *visitor) VisitEmptyStatement_(ctx *mysqlparser.EmptyStatement_Context) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#ddlStatement.
func (v *visitor) VisitDdlStatement(ctx *mysqlparser.DdlStatementContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#dmlStatement.
func (v *visitor) VisitDmlStatement(ctx *mysqlparser.DmlStatementContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#transactionStatement.
func (v *visitor) VisitTransactionStatement(ctx *mysqlparser.TransactionStatementContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#replicationStatement.
func (v *visitor) VisitReplicationStatement(ctx *mysqlparser.ReplicationStatementContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#preparedStatement.
func (v *visitor) VisitPreparedStatement(ctx *mysqlparser.PreparedStatementContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#compoundStatement.
func (v *visitor) VisitCompoundStatement(ctx *mysqlparser.CompoundStatementContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#administrationStatement.
func (v *visitor) VisitAdministrationStatement(ctx *mysqlparser.AdministrationStatementContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#utilityStatement.
func (v *visitor) VisitUtilityStatement(ctx *mysqlparser.UtilityStatementContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#createDatabase.
func (v *visitor) VisitCreateDatabase(ctx *mysqlparser.CreateDatabaseContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#createEvent.
func (v *visitor) VisitCreateEvent(ctx *mysqlparser.CreateEventContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#createIndex.
func (v *visitor) VisitCreateIndex(ctx *mysqlparser.CreateIndexContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#createLogfileGroup.
func (v *visitor) VisitCreateLogfileGroup(ctx *mysqlparser.CreateLogfileGroupContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#createProcedure.
func (v *visitor) VisitCreateProcedure(ctx *mysqlparser.CreateProcedureContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#createFunction.
func (v *visitor) VisitCreateFunction(ctx *mysqlparser.CreateFunctionContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#createRole.
func (v *visitor) VisitCreateRole(ctx *mysqlparser.CreateRoleContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#createServer.
func (v *visitor) VisitCreateServer(ctx *mysqlparser.CreateServerContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#copyCreateTable.
func (v *visitor) VisitCopyCreateTable(ctx *mysqlparser.CopyCreateTableContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#queryCreateTable.
func (v *visitor) VisitQueryCreateTable(ctx *mysqlparser.QueryCreateTableContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#columnCreateTable.
func (v *visitor) VisitColumnCreateTable(ctx *mysqlparser.ColumnCreateTableContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#createTablespaceInnodb.
func (v *visitor) VisitCreateTablespaceInnodb(ctx *mysqlparser.CreateTablespaceInnodbContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#createTablespaceNdb.
func (v *visitor) VisitCreateTablespaceNdb(ctx *mysqlparser.CreateTablespaceNdbContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#createTrigger.
func (v *visitor) VisitCreateTrigger(ctx *mysqlparser.CreateTriggerContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#withClause.
func (v *visitor) VisitWithClause(ctx *mysqlparser.WithClauseContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#commonTableExpression.
func (v *visitor) VisitCommonTableExpression(ctx *mysqlparser.CommonTableExpressionContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#cteName.
func (v *visitor) VisitCteName(ctx *mysqlparser.CteNameContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#cteColumnName.
func (v *visitor) VisitCteColumnName(ctx *mysqlparser.CteColumnNameContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#createView.
func (v *visitor) VisitCreateView(ctx *mysqlparser.CreateViewContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#createDatabaseOption.
func (v *visitor) VisitCreateDatabaseOption(ctx *mysqlparser.CreateDatabaseOptionContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#charSet.
func (v *visitor) VisitCharSet(ctx *mysqlparser.CharSetContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#ownerStatement.
func (v *visitor) VisitOwnerStatement(ctx *mysqlparser.OwnerStatementContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#preciseSchedule.
func (v *visitor) VisitPreciseSchedule(ctx *mysqlparser.PreciseScheduleContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#intervalSchedule.
func (v *visitor) VisitIntervalSchedule(ctx *mysqlparser.IntervalScheduleContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#timestampValue.
func (v *visitor) VisitTimestampValue(ctx *mysqlparser.TimestampValueContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#intervalExpr.
func (v *visitor) VisitIntervalExpr(ctx *mysqlparser.IntervalExprContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#intervalType.
func (v *visitor) VisitIntervalType(ctx *mysqlparser.IntervalTypeContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#enableType.
func (v *visitor) VisitEnableType(ctx *mysqlparser.EnableTypeContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#indexType.
func (v *visitor) VisitIndexType(ctx *mysqlparser.IndexTypeContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#indexOption.
func (v *visitor) VisitIndexOption(ctx *mysqlparser.IndexOptionContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#procedureParameter.
func (v *visitor) VisitProcedureParameter(ctx *mysqlparser.ProcedureParameterContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#functionParameter.
func (v *visitor) VisitFunctionParameter(ctx *mysqlparser.FunctionParameterContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#routineComment.
func (v *visitor) VisitRoutineComment(ctx *mysqlparser.RoutineCommentContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#routineLanguage.
func (v *visitor) VisitRoutineLanguage(ctx *mysqlparser.RoutineLanguageContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#routineBehavior.
func (v *visitor) VisitRoutineBehavior(ctx *mysqlparser.RoutineBehaviorContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#routineData.
func (v *visitor) VisitRoutineData(ctx *mysqlparser.RoutineDataContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#routineSecurity.
func (v *visitor) VisitRoutineSecurity(ctx *mysqlparser.RoutineSecurityContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#serverOption.
func (v *visitor) VisitServerOption(ctx *mysqlparser.ServerOptionContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#createDefinitions.
func (v *visitor) VisitCreateDefinitions(ctx *mysqlparser.CreateDefinitionsContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#columnDeclaration.
func (v *visitor) VisitColumnDeclaration(ctx *mysqlparser.ColumnDeclarationContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#constraintDeclaration.
func (v *visitor) VisitConstraintDeclaration(ctx *mysqlparser.ConstraintDeclarationContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#indexDeclaration.
func (v *visitor) VisitIndexDeclaration(ctx *mysqlparser.IndexDeclarationContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#columnDefinition.
func (v *visitor) VisitColumnDefinition(ctx *mysqlparser.ColumnDefinitionContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#nullColumnConstraint.
func (v *visitor) VisitNullColumnConstraint(ctx *mysqlparser.NullColumnConstraintContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#defaultColumnConstraint.
func (v *visitor) VisitDefaultColumnConstraint(ctx *mysqlparser.DefaultColumnConstraintContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#visibilityColumnConstraint.
func (v *visitor) VisitVisibilityColumnConstraint(ctx *mysqlparser.VisibilityColumnConstraintContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#invisibilityColumnConstraint.
func (v *visitor) VisitInvisibilityColumnConstraint(ctx *mysqlparser.InvisibilityColumnConstraintContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#autoIncrementColumnConstraint.
func (v *visitor) VisitAutoIncrementColumnConstraint(ctx *mysqlparser.AutoIncrementColumnConstraintContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#primaryKeyColumnConstraint.
func (v *visitor) VisitPrimaryKeyColumnConstraint(ctx *mysqlparser.PrimaryKeyColumnConstraintContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#uniqueKeyColumnConstraint.
func (v *visitor) VisitUniqueKeyColumnConstraint(ctx *mysqlparser.UniqueKeyColumnConstraintContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#commentColumnConstraint.
func (v *visitor) VisitCommentColumnConstraint(ctx *mysqlparser.CommentColumnConstraintContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#formatColumnConstraint.
func (v *visitor) VisitFormatColumnConstraint(ctx *mysqlparser.FormatColumnConstraintContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#storageColumnConstraint.
func (v *visitor) VisitStorageColumnConstraint(ctx *mysqlparser.StorageColumnConstraintContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#referenceColumnConstraint.
func (v *visitor) VisitReferenceColumnConstraint(ctx *mysqlparser.ReferenceColumnConstraintContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#collateColumnConstraint.
func (v *visitor) VisitCollateColumnConstraint(ctx *mysqlparser.CollateColumnConstraintContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#generatedColumnConstraint.
func (v *visitor) VisitGeneratedColumnConstraint(ctx *mysqlparser.GeneratedColumnConstraintContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#serialDefaultColumnConstraint.
func (v *visitor) VisitSerialDefaultColumnConstraint(ctx *mysqlparser.SerialDefaultColumnConstraintContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#checkColumnConstraint.
func (v *visitor) VisitCheckColumnConstraint(ctx *mysqlparser.CheckColumnConstraintContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#primaryKeyTableConstraint.
func (v *visitor) VisitPrimaryKeyTableConstraint(ctx *mysqlparser.PrimaryKeyTableConstraintContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#uniqueKeyTableConstraint.
func (v *visitor) VisitUniqueKeyTableConstraint(ctx *mysqlparser.UniqueKeyTableConstraintContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#foreignKeyTableConstraint.
func (v *visitor) VisitForeignKeyTableConstraint(ctx *mysqlparser.ForeignKeyTableConstraintContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#checkTableConstraint.
func (v *visitor) VisitCheckTableConstraint(ctx *mysqlparser.CheckTableConstraintContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#referenceDefinition.
func (v *visitor) VisitReferenceDefinition(ctx *mysqlparser.ReferenceDefinitionContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#referenceAction.
func (v *visitor) VisitReferenceAction(ctx *mysqlparser.ReferenceActionContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#referenceControlType.
func (v *visitor) VisitReferenceControlType(ctx *mysqlparser.ReferenceControlTypeContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#simpleIndexDeclaration.
func (v *visitor) VisitSimpleIndexDeclaration(ctx *mysqlparser.SimpleIndexDeclarationContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#specialIndexDeclaration.
func (v *visitor) VisitSpecialIndexDeclaration(ctx *mysqlparser.SpecialIndexDeclarationContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#tableOptionEngine.
func (v *visitor) VisitTableOptionEngine(ctx *mysqlparser.TableOptionEngineContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#tableOptionEngineAttribute.
func (v *visitor) VisitTableOptionEngineAttribute(ctx *mysqlparser.TableOptionEngineAttributeContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#tableOptionAutoextendSize.
func (v *visitor) VisitTableOptionAutoextendSize(ctx *mysqlparser.TableOptionAutoextendSizeContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#tableOptionAutoIncrement.
func (v *visitor) VisitTableOptionAutoIncrement(ctx *mysqlparser.TableOptionAutoIncrementContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#tableOptionAverage.
func (v *visitor) VisitTableOptionAverage(ctx *mysqlparser.TableOptionAverageContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#tableOptionCharset.
func (v *visitor) VisitTableOptionCharset(ctx *mysqlparser.TableOptionCharsetContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#tableOptionChecksum.
func (v *visitor) VisitTableOptionChecksum(ctx *mysqlparser.TableOptionChecksumContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#tableOptionCollate.
func (v *visitor) VisitTableOptionCollate(ctx *mysqlparser.TableOptionCollateContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#tableOptionComment.
func (v *visitor) VisitTableOptionComment(ctx *mysqlparser.TableOptionCommentContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#tableOptionCompression.
func (v *visitor) VisitTableOptionCompression(ctx *mysqlparser.TableOptionCompressionContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#tableOptionConnection.
func (v *visitor) VisitTableOptionConnection(ctx *mysqlparser.TableOptionConnectionContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#tableOptionDataDirectory.
func (v *visitor) VisitTableOptionDataDirectory(ctx *mysqlparser.TableOptionDataDirectoryContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#tableOptionDelay.
func (v *visitor) VisitTableOptionDelay(ctx *mysqlparser.TableOptionDelayContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#tableOptionEncryption.
func (v *visitor) VisitTableOptionEncryption(ctx *mysqlparser.TableOptionEncryptionContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#tableOptionPageCompressed.
func (v *visitor) VisitTableOptionPageCompressed(ctx *mysqlparser.TableOptionPageCompressedContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#tableOptionPageCompressionLevel.
func (v *visitor) VisitTableOptionPageCompressionLevel(ctx *mysqlparser.TableOptionPageCompressionLevelContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#tableOptionEncryptionKeyId.
func (v *visitor) VisitTableOptionEncryptionKeyId(ctx *mysqlparser.TableOptionEncryptionKeyIdContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#tableOptionIndexDirectory.
func (v *visitor) VisitTableOptionIndexDirectory(ctx *mysqlparser.TableOptionIndexDirectoryContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#tableOptionInsertMethod.
func (v *visitor) VisitTableOptionInsertMethod(ctx *mysqlparser.TableOptionInsertMethodContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#tableOptionKeyBlockSize.
func (v *visitor) VisitTableOptionKeyBlockSize(ctx *mysqlparser.TableOptionKeyBlockSizeContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#tableOptionMaxRows.
func (v *visitor) VisitTableOptionMaxRows(ctx *mysqlparser.TableOptionMaxRowsContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#tableOptionMinRows.
func (v *visitor) VisitTableOptionMinRows(ctx *mysqlparser.TableOptionMinRowsContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#tableOptionPackKeys.
func (v *visitor) VisitTableOptionPackKeys(ctx *mysqlparser.TableOptionPackKeysContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#tableOptionPassword.
func (v *visitor) VisitTableOptionPassword(ctx *mysqlparser.TableOptionPasswordContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#tableOptionRowFormat.
func (v *visitor) VisitTableOptionRowFormat(ctx *mysqlparser.TableOptionRowFormatContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#tableOptionStartTransaction.
func (v *visitor) VisitTableOptionStartTransaction(ctx *mysqlparser.TableOptionStartTransactionContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#tableOptionSecondaryEngineAttribute.
func (v *visitor) VisitTableOptionSecondaryEngineAttribute(ctx *mysqlparser.TableOptionSecondaryEngineAttributeContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#tableOptionRecalculation.
func (v *visitor) VisitTableOptionRecalculation(ctx *mysqlparser.TableOptionRecalculationContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#tableOptionPersistent.
func (v *visitor) VisitTableOptionPersistent(ctx *mysqlparser.TableOptionPersistentContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#tableOptionSamplePage.
func (v *visitor) VisitTableOptionSamplePage(ctx *mysqlparser.TableOptionSamplePageContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#tableOptionTablespace.
func (v *visitor) VisitTableOptionTablespace(ctx *mysqlparser.TableOptionTablespaceContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#tableOptionTableType.
func (v *visitor) VisitTableOptionTableType(ctx *mysqlparser.TableOptionTableTypeContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#tableOptionTransactional.
func (v *visitor) VisitTableOptionTransactional(ctx *mysqlparser.TableOptionTransactionalContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#tableOptionUnion.
func (v *visitor) VisitTableOptionUnion(ctx *mysqlparser.TableOptionUnionContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#tableType.
func (v *visitor) VisitTableType(ctx *mysqlparser.TableTypeContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#tablespaceStorage.
func (v *visitor) VisitTablespaceStorage(ctx *mysqlparser.TablespaceStorageContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#partitionDefinitions.
func (v *visitor) VisitPartitionDefinitions(ctx *mysqlparser.PartitionDefinitionsContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#partitionFunctionHash.
func (v *visitor) VisitPartitionFunctionHash(ctx *mysqlparser.PartitionFunctionHashContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#partitionFunctionKey.
func (v *visitor) VisitPartitionFunctionKey(ctx *mysqlparser.PartitionFunctionKeyContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#partitionFunctionRange.
func (v *visitor) VisitPartitionFunctionRange(ctx *mysqlparser.PartitionFunctionRangeContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#partitionFunctionList.
func (v *visitor) VisitPartitionFunctionList(ctx *mysqlparser.PartitionFunctionListContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#subPartitionFunctionHash.
func (v *visitor) VisitSubPartitionFunctionHash(ctx *mysqlparser.SubPartitionFunctionHashContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#subPartitionFunctionKey.
func (v *visitor) VisitSubPartitionFunctionKey(ctx *mysqlparser.SubPartitionFunctionKeyContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#partitionComparison.
func (v *visitor) VisitPartitionComparison(ctx *mysqlparser.PartitionComparisonContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#partitionListAtom.
func (v *visitor) VisitPartitionListAtom(ctx *mysqlparser.PartitionListAtomContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#partitionListVector.
func (v *visitor) VisitPartitionListVector(ctx *mysqlparser.PartitionListVectorContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#partitionSimple.
func (v *visitor) VisitPartitionSimple(ctx *mysqlparser.PartitionSimpleContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#partitionDefinerAtom.
func (v *visitor) VisitPartitionDefinerAtom(ctx *mysqlparser.PartitionDefinerAtomContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#partitionDefinerVector.
func (v *visitor) VisitPartitionDefinerVector(ctx *mysqlparser.PartitionDefinerVectorContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#subpartitionDefinition.
func (v *visitor) VisitSubpartitionDefinition(ctx *mysqlparser.SubpartitionDefinitionContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#partitionOptionEngine.
func (v *visitor) VisitPartitionOptionEngine(ctx *mysqlparser.PartitionOptionEngineContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#partitionOptionComment.
func (v *visitor) VisitPartitionOptionComment(ctx *mysqlparser.PartitionOptionCommentContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#partitionOptionDataDirectory.
func (v *visitor) VisitPartitionOptionDataDirectory(ctx *mysqlparser.PartitionOptionDataDirectoryContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#partitionOptionIndexDirectory.
func (v *visitor) VisitPartitionOptionIndexDirectory(ctx *mysqlparser.PartitionOptionIndexDirectoryContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#partitionOptionMaxRows.
func (v *visitor) VisitPartitionOptionMaxRows(ctx *mysqlparser.PartitionOptionMaxRowsContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#partitionOptionMinRows.
func (v *visitor) VisitPartitionOptionMinRows(ctx *mysqlparser.PartitionOptionMinRowsContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#partitionOptionTablespace.
func (v *visitor) VisitPartitionOptionTablespace(ctx *mysqlparser.PartitionOptionTablespaceContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#partitionOptionNodeGroup.
func (v *visitor) VisitPartitionOptionNodeGroup(ctx *mysqlparser.PartitionOptionNodeGroupContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#alterSimpleDatabase.
func (v *visitor) VisitAlterSimpleDatabase(ctx *mysqlparser.AlterSimpleDatabaseContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#alterUpgradeName.
func (v *visitor) VisitAlterUpgradeName(ctx *mysqlparser.AlterUpgradeNameContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#alterEvent.
func (v *visitor) VisitAlterEvent(ctx *mysqlparser.AlterEventContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#alterFunction.
func (v *visitor) VisitAlterFunction(ctx *mysqlparser.AlterFunctionContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#alterInstance.
func (v *visitor) VisitAlterInstance(ctx *mysqlparser.AlterInstanceContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#alterLogfileGroup.
func (v *visitor) VisitAlterLogfileGroup(ctx *mysqlparser.AlterLogfileGroupContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#alterProcedure.
func (v *visitor) VisitAlterProcedure(ctx *mysqlparser.AlterProcedureContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#alterServer.
func (v *visitor) VisitAlterServer(ctx *mysqlparser.AlterServerContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#alterTable.
func (v *visitor) VisitAlterTable(ctx *mysqlparser.AlterTableContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#alterTablespace.
func (v *visitor) VisitAlterTablespace(ctx *mysqlparser.AlterTablespaceContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#alterView.
func (v *visitor) VisitAlterView(ctx *mysqlparser.AlterViewContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#alterByTableOption.
func (v *visitor) VisitAlterByTableOption(ctx *mysqlparser.AlterByTableOptionContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#alterByAddColumn.
func (v *visitor) VisitAlterByAddColumn(ctx *mysqlparser.AlterByAddColumnContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#alterByAddColumns.
func (v *visitor) VisitAlterByAddColumns(ctx *mysqlparser.AlterByAddColumnsContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#alterByAddIndex.
func (v *visitor) VisitAlterByAddIndex(ctx *mysqlparser.AlterByAddIndexContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#alterByAddPrimaryKey.
func (v *visitor) VisitAlterByAddPrimaryKey(ctx *mysqlparser.AlterByAddPrimaryKeyContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#alterByAddUniqueKey.
func (v *visitor) VisitAlterByAddUniqueKey(ctx *mysqlparser.AlterByAddUniqueKeyContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#alterByAddSpecialIndex.
func (v *visitor) VisitAlterByAddSpecialIndex(ctx *mysqlparser.AlterByAddSpecialIndexContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#alterByAddForeignKey.
func (v *visitor) VisitAlterByAddForeignKey(ctx *mysqlparser.AlterByAddForeignKeyContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#alterByAddCheckTableConstraint.
func (v *visitor) VisitAlterByAddCheckTableConstraint(ctx *mysqlparser.AlterByAddCheckTableConstraintContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#alterByAlterCheckTableConstraint.
func (v *visitor) VisitAlterByAlterCheckTableConstraint(ctx *mysqlparser.AlterByAlterCheckTableConstraintContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#alterBySetAlgorithm.
func (v *visitor) VisitAlterBySetAlgorithm(ctx *mysqlparser.AlterBySetAlgorithmContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#alterByChangeDefault.
func (v *visitor) VisitAlterByChangeDefault(ctx *mysqlparser.AlterByChangeDefaultContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#alterByChangeColumn.
func (v *visitor) VisitAlterByChangeColumn(ctx *mysqlparser.AlterByChangeColumnContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#alterByRenameColumn.
func (v *visitor) VisitAlterByRenameColumn(ctx *mysqlparser.AlterByRenameColumnContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#alterByLock.
func (v *visitor) VisitAlterByLock(ctx *mysqlparser.AlterByLockContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#alterByModifyColumn.
func (v *visitor) VisitAlterByModifyColumn(ctx *mysqlparser.AlterByModifyColumnContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#alterByDropColumn.
func (v *visitor) VisitAlterByDropColumn(ctx *mysqlparser.AlterByDropColumnContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#alterByDropConstraintCheck.
func (v *visitor) VisitAlterByDropConstraintCheck(ctx *mysqlparser.AlterByDropConstraintCheckContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#alterByDropPrimaryKey.
func (v *visitor) VisitAlterByDropPrimaryKey(ctx *mysqlparser.AlterByDropPrimaryKeyContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#alterByDropIndex.
func (v *visitor) VisitAlterByDropIndex(ctx *mysqlparser.AlterByDropIndexContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#alterByRenameIndex.
func (v *visitor) VisitAlterByRenameIndex(ctx *mysqlparser.AlterByRenameIndexContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#alterByAlterColumnDefault.
func (v *visitor) VisitAlterByAlterColumnDefault(ctx *mysqlparser.AlterByAlterColumnDefaultContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#alterByAlterIndexVisibility.
func (v *visitor) VisitAlterByAlterIndexVisibility(ctx *mysqlparser.AlterByAlterIndexVisibilityContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#alterByDropForeignKey.
func (v *visitor) VisitAlterByDropForeignKey(ctx *mysqlparser.AlterByDropForeignKeyContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#alterByDisableKeys.
func (v *visitor) VisitAlterByDisableKeys(ctx *mysqlparser.AlterByDisableKeysContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#alterByEnableKeys.
func (v *visitor) VisitAlterByEnableKeys(ctx *mysqlparser.AlterByEnableKeysContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#alterByRename.
func (v *visitor) VisitAlterByRename(ctx *mysqlparser.AlterByRenameContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#alterByOrder.
func (v *visitor) VisitAlterByOrder(ctx *mysqlparser.AlterByOrderContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#alterByConvertCharset.
func (v *visitor) VisitAlterByConvertCharset(ctx *mysqlparser.AlterByConvertCharsetContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#alterByDefaultCharset.
func (v *visitor) VisitAlterByDefaultCharset(ctx *mysqlparser.AlterByDefaultCharsetContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#alterByDiscardTablespace.
func (v *visitor) VisitAlterByDiscardTablespace(ctx *mysqlparser.AlterByDiscardTablespaceContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#alterByImportTablespace.
func (v *visitor) VisitAlterByImportTablespace(ctx *mysqlparser.AlterByImportTablespaceContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#alterByForce.
func (v *visitor) VisitAlterByForce(ctx *mysqlparser.AlterByForceContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#alterByValidate.
func (v *visitor) VisitAlterByValidate(ctx *mysqlparser.AlterByValidateContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#alterByAddPartition.
func (v *visitor) VisitAlterByAddPartition(ctx *mysqlparser.AlterByAddPartitionContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#alterByDropPartition.
func (v *visitor) VisitAlterByDropPartition(ctx *mysqlparser.AlterByDropPartitionContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#alterByDiscardPartition.
func (v *visitor) VisitAlterByDiscardPartition(ctx *mysqlparser.AlterByDiscardPartitionContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#alterByImportPartition.
func (v *visitor) VisitAlterByImportPartition(ctx *mysqlparser.AlterByImportPartitionContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#alterByTruncatePartition.
func (v *visitor) VisitAlterByTruncatePartition(ctx *mysqlparser.AlterByTruncatePartitionContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#alterByCoalescePartition.
func (v *visitor) VisitAlterByCoalescePartition(ctx *mysqlparser.AlterByCoalescePartitionContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#alterByReorganizePartition.
func (v *visitor) VisitAlterByReorganizePartition(ctx *mysqlparser.AlterByReorganizePartitionContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#alterByExchangePartition.
func (v *visitor) VisitAlterByExchangePartition(ctx *mysqlparser.AlterByExchangePartitionContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#alterByAnalyzePartition.
func (v *visitor) VisitAlterByAnalyzePartition(ctx *mysqlparser.AlterByAnalyzePartitionContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#alterByCheckPartition.
func (v *visitor) VisitAlterByCheckPartition(ctx *mysqlparser.AlterByCheckPartitionContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#alterByOptimizePartition.
func (v *visitor) VisitAlterByOptimizePartition(ctx *mysqlparser.AlterByOptimizePartitionContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#alterByRebuildPartition.
func (v *visitor) VisitAlterByRebuildPartition(ctx *mysqlparser.AlterByRebuildPartitionContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#alterByRepairPartition.
func (v *visitor) VisitAlterByRepairPartition(ctx *mysqlparser.AlterByRepairPartitionContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#alterByRemovePartitioning.
func (v *visitor) VisitAlterByRemovePartitioning(ctx *mysqlparser.AlterByRemovePartitioningContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#alterByUpgradePartitioning.
func (v *visitor) VisitAlterByUpgradePartitioning(ctx *mysqlparser.AlterByUpgradePartitioningContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#alterByAddDefinitions.
func (v *visitor) VisitAlterByAddDefinitions(ctx *mysqlparser.AlterByAddDefinitionsContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#dropDatabase.
func (v *visitor) VisitDropDatabase(ctx *mysqlparser.DropDatabaseContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#dropEvent.
func (v *visitor) VisitDropEvent(ctx *mysqlparser.DropEventContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#dropIndex.
func (v *visitor) VisitDropIndex(ctx *mysqlparser.DropIndexContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#dropLogfileGroup.
func (v *visitor) VisitDropLogfileGroup(ctx *mysqlparser.DropLogfileGroupContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#dropProcedure.
func (v *visitor) VisitDropProcedure(ctx *mysqlparser.DropProcedureContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#dropFunction.
func (v *visitor) VisitDropFunction(ctx *mysqlparser.DropFunctionContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#dropServer.
func (v *visitor) VisitDropServer(ctx *mysqlparser.DropServerContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#dropTable.
func (v *visitor) VisitDropTable(ctx *mysqlparser.DropTableContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#dropTablespace.
func (v *visitor) VisitDropTablespace(ctx *mysqlparser.DropTablespaceContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#dropTrigger.
func (v *visitor) VisitDropTrigger(ctx *mysqlparser.DropTriggerContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#dropView.
func (v *visitor) VisitDropView(ctx *mysqlparser.DropViewContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#dropRole.
func (v *visitor) VisitDropRole(ctx *mysqlparser.DropRoleContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#setRole.
func (v *visitor) VisitSetRole(ctx *mysqlparser.SetRoleContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#renameTable.
func (v *visitor) VisitRenameTable(ctx *mysqlparser.RenameTableContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#renameTableClause.
func (v *visitor) VisitRenameTableClause(ctx *mysqlparser.RenameTableClauseContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#truncateTable.
func (v *visitor) VisitTruncateTable(ctx *mysqlparser.TruncateTableContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#callStatement.
func (v *visitor) VisitCallStatement(ctx *mysqlparser.CallStatementContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#deleteStatement.
func (v *visitor) VisitDeleteStatement(ctx *mysqlparser.DeleteStatementContext) any {
	if single := ctx.SingleDeleteStatement(); single != nil {
		return single.Accept(v)
	}

	if multi := ctx.MultipleDeleteStatement(); multi != nil {
		return multi.Accept(v)
	}

	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#doStatement.
func (v *visitor) VisitDoStatement(ctx *mysqlparser.DoStatementContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#handlerStatement.
func (v *visitor) VisitHandlerStatement(ctx *mysqlparser.HandlerStatementContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#insertStatement.
func (v *visitor) VisitInsertStatement(ctx *mysqlparser.InsertStatementContext) any {
	// Defer reset the source list
	defer func(l int) {
		v.Sources = v.Sources[:l]
	}(len(v.Sources))

	tableName := getFullIDName(ctx.TableName().FullId())
	tableSource := v.getSourceFromTable(ctx)
	v.Sources = append(v.Sources, tableSource)

	v.VisitChildren(ctx)
	if v.Err != nil {
		v.Err = fmt.Errorf("insert stmt: %w", v.Err)
		return nil
	}

	var colNames []string
	if full := ctx.FullColumnNameList(); full != nil {
		columns := full.AllFullColumnName()
		colNames = make([]string, len(columns))
		for i := range columns {
			colNames[i] = getFullColumnName(columns[i])
		}
	}

	if len(colNames) == 0 {
		colNames = make([]string, len(tableSource.Columns))
		for i := range tableSource.Columns {
			colNames[i] = tableSource.Columns[i].Name
		}
	}

	if values := ctx.InsertStatementValue(); values != nil {
		rows := values.AllExpressionsWithDefaults()
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
			for valIndex, value := range row.AllExpressionOrDefault() {
				v.UpdateInfo(NodeInfo{
					Node:            value,
					ExprDescription: "ROW Value",
					Type: []NodeType{getColumnType(
						v.DB,
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

	return nil
}

// Visit a parse tree produced by MySqlParser#loadDataStatement.
func (v *visitor) VisitLoadDataStatement(ctx *mysqlparser.LoadDataStatementContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#loadXmlStatement.
func (v *visitor) VisitLoadXmlStatement(ctx *mysqlparser.LoadXmlStatementContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#replaceStatement.
func (v *visitor) VisitReplaceStatement(ctx *mysqlparser.ReplaceStatementContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#simpleSelect.
func (v *visitor) VisitSelectStatement(ctx *mysqlparser.SelectStatementContext) any {
	// Defer reset the source list
	defer func(l int) {
		v.Sources = v.Sources[:l]
	}(len(v.Sources))

	v.addSourcesFromWithClause(ctx.WithClause())
	if v.Err != nil {
		v.Err = fmt.Errorf("with clause: %w", v.Err)
		return nil
	}

	var source QuerySource

	if base := ctx.SelectStatementBase(); base != nil {
		if from := base.FromClause(); from != nil {
			v.addSourcesFromTableSources(from.TableSources())
			if v.Err != nil {
				v.Err = fmt.Errorf("add base sources: %w", v.Err)
				return nil
			}
		}
	}

	v.VisitChildren(ctx)
	if v.Err != nil {
		v.Err = fmt.Errorf("select stmt: %w", v.Err)
		return nil
	}

	if setQuery := ctx.SetQuery(); setQuery != nil {
		source = v.querySources[antlrhelpers.Key(setQuery)]
	}

	// Getting the source should come after visiting children
	// so that the types are correctly set
	if base := ctx.SelectStatementBase(); base != nil {
		source = v.getSourceFromSelectElements(base.SelectElements())
		if v.Err != nil {
			v.Err = fmt.Errorf("get base source: %w", v.Err)
			return nil
		}
	}

	return source.Columns
}

// Visit a parse tree produced by MySqlParser#updateStatement.
func (v *visitor) VisitUpdateStatement(ctx *mysqlparser.UpdateStatementContext) any {
	if single := ctx.SingleUpdateStatement(); single != nil {
		return single.Accept(v)
	}

	if multi := ctx.MultipleUpdateStatement(); multi != nil {
		return multi.Accept(v)
	}

	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#valuesStatement.
func (v *visitor) VisitValuesStatement(ctx *mysqlparser.ValuesStatementContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#insertStatementValue.
func (v *visitor) VisitInsertStatementValue(ctx *mysqlparser.InsertStatementValueContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#updatedElement.
func (v *visitor) VisitUpdatedElement(ctx *mysqlparser.UpdatedElementContext) any {
	v.VisitChildren(ctx)
	if v.Err != nil {
		return nil
	}

	v.UpdateInfo(NodeInfo{
		Node:                 ctx.FullColumnName(),
		ExprDescription:      "Update Elem Col",
		ExprRef:              ctx.ExpressionOrDefault(),
		IgnoreRefNullability: true,
	})

	v.UpdateInfo(NodeInfo{
		Node:                 ctx.ExpressionOrDefault(),
		ExprDescription:      "Update Elem Expr",
		ExprRef:              ctx.FullColumnName(),
		IgnoreRefNullability: true,
	})

	v.MatchNodeNames(ctx.FullColumnName(), ctx.ExpressionOrDefault())

	return nil
}

// Visit a parse tree produced by MySqlParser#assignmentField.
func (v *visitor) VisitAssignmentField(ctx *mysqlparser.AssignmentFieldContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#lockClause.
func (v *visitor) VisitLockClause(ctx *mysqlparser.LockClauseContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#singleDeleteStatement.
func (v *visitor) VisitSingleDeleteStatement(ctx *mysqlparser.SingleDeleteStatementContext) any {
	// Defer reset the source list
	defer func(l int) {
		v.Sources = v.Sources[:l]
	}(len(v.Sources))

	v.addSourcesFromWithClause(ctx.WithClause())
	if v.Err != nil {
		v.Err = fmt.Errorf("with clause: %w", v.Err)
		return nil
	}

	tableSource := v.getSourceFromTable(ctx)
	v.Sources = append(v.Sources, tableSource)

	v.VisitChildren(ctx)
	if v.Err != nil {
		v.Err = fmt.Errorf("update stmt: %w", v.Err)
		return nil
	}

	return nil
}

// Visit a parse tree produced by MySqlParser#multipleDeleteStatement.
func (v *visitor) VisitMultipleDeleteStatement(ctx *mysqlparser.MultipleDeleteStatementContext) any {
	if using := ctx.USING(); using == nil {
		v.Err = fmt.Errorf("only the USING form is supported in DELETE statements")
		return nil
	}

	// Defer reset the source list
	defer func(l int) {
		v.Sources = v.Sources[:l]
	}(len(v.Sources))

	v.addSourcesFromWithClause(ctx.WithClause())
	if v.Err != nil {
		v.Err = fmt.Errorf("with clause: %w", v.Err)
		return nil
	}

	for _, table := range ctx.AllMultipleDeleteTable() {
		v.Sources = append(v.Sources, v.getSourceFromTableName(table.TableName()))
		if v.Err != nil {
			v.Err = fmt.Errorf("table name: %w", v.Err)
			return nil
		}
	}

	v.addSourcesFromTableSources(ctx.TableSources())
	if v.Err != nil {
		v.Err = fmt.Errorf("with clause: %w", v.Err)
		return nil
	}

	v.VisitChildren(ctx)
	if v.Err != nil {
		v.Err = fmt.Errorf("update stmt: %w", v.Err)
		return nil
	}

	return nil
}

// VisitMultipleDeleteTable implements parser.MySqlParserVisitor.
func (v *visitor) VisitMultipleDeleteTable(ctx *mysqlparser.MultipleDeleteTableContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#handlerOpenStatement.
func (v *visitor) VisitHandlerOpenStatement(ctx *mysqlparser.HandlerOpenStatementContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#handlerReadIndexStatement.
func (v *visitor) VisitHandlerReadIndexStatement(ctx *mysqlparser.HandlerReadIndexStatementContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#handlerReadStatement.
func (v *visitor) VisitHandlerReadStatement(ctx *mysqlparser.HandlerReadStatementContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#handlerCloseStatement.
func (v *visitor) VisitHandlerCloseStatement(ctx *mysqlparser.HandlerCloseStatementContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#singleUpdateStatement.
func (v *visitor) VisitSingleUpdateStatement(ctx *mysqlparser.SingleUpdateStatementContext) any {
	// Defer reset the source list
	defer func(l int) {
		v.Sources = v.Sources[:l]
	}(len(v.Sources))

	v.addSourcesFromWithClause(ctx.WithClause())
	if v.Err != nil {
		v.Err = fmt.Errorf("with clause: %w", v.Err)
		return nil
	}

	tableSource := v.getSourceFromTable(ctx)
	v.Sources = append(v.Sources, tableSource)

	v.VisitChildren(ctx)
	if v.Err != nil {
		v.Err = fmt.Errorf("update stmt: %w", v.Err)
		return nil
	}

	return nil
}

// Visit a parse tree produced by MySqlParser#multipleUpdateStatement.
func (v *visitor) VisitMultipleUpdateStatement(ctx *mysqlparser.MultipleUpdateStatementContext) any {
	// Defer reset the source list
	defer func(l int) {
		v.Sources = v.Sources[:l]
	}(len(v.Sources))

	v.addSourcesFromWithClause(ctx.WithClause())
	if v.Err != nil {
		v.Err = fmt.Errorf("with clause: %w", v.Err)
		return nil
	}

	v.addSourcesFromTableSources(ctx.TableSources())
	if v.Err != nil {
		v.Err = fmt.Errorf("with clause: %w", v.Err)
		return nil
	}

	v.VisitChildren(ctx)
	if v.Err != nil {
		v.Err = fmt.Errorf("update stmt: %w", v.Err)
		return nil
	}

	return nil
}

// Visit a parse tree produced by MySqlParser#orderByClause.
func (v *visitor) VisitOrderByClause(ctx *mysqlparser.OrderByClauseContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#orderByExpression.
func (v *visitor) VisitOrderByExpression(ctx *mysqlparser.OrderByExpressionContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#tableSources.
func (v *visitor) VisitTableSources(ctx *mysqlparser.TableSourcesContext) any {
	return v.VisitChildren(ctx)
}

// VisitTableSource implements parser.MySqlParserVisitor.
func (v *visitor) VisitTableSource(ctx *mysqlparser.TableSourceContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#atomTableItem.
func (v *visitor) VisitAtomTableItem(ctx *mysqlparser.AtomTableItemContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#subqueryTableItem.
func (v *visitor) VisitSubqueryTableItem(ctx *mysqlparser.SubqueryTableItemContext) any {
	return v.VisitChildren(ctx)
}

// VisitJsonTableItem implements parser.MySqlParserVisitor.
func (v *visitor) VisitJsonTableItem(ctx *mysqlparser.JsonTableItemContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#tableSourcesItem.
func (v *visitor) VisitTableSourcesItem(ctx *mysqlparser.TableSourcesItemContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#indexHint.
func (v *visitor) VisitIndexHint(ctx *mysqlparser.IndexHintContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#indexHintType.
func (v *visitor) VisitIndexHintType(ctx *mysqlparser.IndexHintTypeContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#innerJoin.
func (v *visitor) VisitInnerJoin(ctx *mysqlparser.InnerJoinContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#straightJoin.
func (v *visitor) VisitStraightJoin(ctx *mysqlparser.StraightJoinContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#outerJoin.
func (v *visitor) VisitOuterJoin(ctx *mysqlparser.OuterJoinContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#naturalJoin.
func (v *visitor) VisitNaturalJoin(ctx *mysqlparser.NaturalJoinContext) any {
	return v.VisitChildren(ctx)
}

// VisitJoinSpecification implements parser.MySqlParserVisitor.
func (v *visitor) VisitJoinSpecification(ctx *mysqlparser.JoinSpecificationContext) any {
	return v.VisitChildren(ctx)
}

// VisitSelectStatementBase implements parser.MySqlParserVisitor.
func (v *visitor) VisitSelectStatementBase(ctx *mysqlparser.SelectStatementBaseContext) any {
	return v.VisitChildren(ctx)
}

// VisitSelectStatementFinish implements parser.MySqlParserVisitor.
func (v *visitor) VisitSelectStatementFinish(ctx *mysqlparser.SelectStatementFinishContext) any {
	return v.VisitChildren(ctx)
}

// VisitSetQuery implements parser.MySqlParserVisitor.
func (v *visitor) VisitSetQuery(ctx *mysqlparser.SetQueryContext) any {
	return v.VisitChildren(ctx)
}

// VisitSetQueryBase implements parser.MySqlParserVisitor.
func (v *visitor) VisitSetQueryBase(ctx *mysqlparser.SetQueryBaseContext) any {
	// Defer reset the source list
	defer func(l int) {
		v.Sources = v.Sources[:l]
	}(len(v.Sources))

	base := ctx.SelectStatementBase()
	if base == nil {
		return v.VisitChildren(ctx)
	}

	if from := base.FromClause(); from != nil {
		v.addSourcesFromTableSources(from.TableSources())
	}

	v.VisitChildren(ctx)
	if v.Err != nil {
		v.Err = fmt.Errorf("children: %w", v.Err)
		return nil
	}

	source := v.getSourceFromSelectElements(base.SelectElements())
	if v.Err != nil {
		v.Err = fmt.Errorf("get set base source: %w", v.Err)
		return nil
	}

	v.querySources[antlrhelpers.Key(ctx)] = source

	return nil
}

// VisitSetQueryInParenthesis implements parser.MySqlParserVisitor.
func (v *visitor) VisitSetQueryInParenthesis(ctx *mysqlparser.SetQueryInParenthesisContext) any {
	base := ctx.SelectStatementBase()
	if base == nil {
		return v.VisitChildren(ctx)
	}

	if from := base.FromClause(); from != nil {
		v.addSourcesFromTableSources(from.TableSources())
	}

	v.VisitChildren(ctx)
	if v.Err != nil {
		v.Err = fmt.Errorf("children: %w", v.Err)
		return nil
	}

	source := v.getSourceFromSelectElements(base.SelectElements())
	if v.Err != nil {
		v.Err = fmt.Errorf("get set base source: %w", v.Err)
		return nil
	}

	v.querySources[antlrhelpers.Key(ctx)] = source

	return nil
}

// VisitSetQueryPart implements parser.MySqlParserVisitor.
func (v *visitor) VisitSetQueryPart(ctx *mysqlparser.SetQueryPartContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#jsonTable.
func (v *visitor) VisitJsonTable(ctx *mysqlparser.JsonTableContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#jsonColumnList.
func (v *visitor) VisitJsonColumnList(ctx *mysqlparser.JsonColumnListContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#jsonColumn.
func (v *visitor) VisitJsonColumn(ctx *mysqlparser.JsonColumnContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#jsonOnEmpty.
func (v *visitor) VisitJsonOnEmpty(ctx *mysqlparser.JsonOnEmptyContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#jsonOnError.
func (v *visitor) VisitJsonOnError(ctx *mysqlparser.JsonOnErrorContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#selectSpec.
func (v *visitor) VisitSelectSpec(ctx *mysqlparser.SelectSpecContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#selectElements.
func (v *visitor) VisitSelectElements(ctx *mysqlparser.SelectElementsContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#selectStarElement.
func (v *visitor) VisitSelectStarElement(ctx *mysqlparser.SelectStarElementContext) any {
	return v.VisitChildren(ctx)
}

// VisitSelectTableElement implements parser.MySqlParserVisitor.
func (v *visitor) VisitSelectTableElement(ctx *mysqlparser.SelectTableElementContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#selectColumnElement.
func (v *visitor) VisitSelectColumnElement(ctx *mysqlparser.SelectColumnElementContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#selectFunctionElement.
func (v *visitor) VisitSelectFunctionElement(ctx *mysqlparser.SelectFunctionElementContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#selectExpressionElement.
func (v *visitor) VisitSelectExpressionElement(ctx *mysqlparser.SelectExpressionElementContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#selectIntoVariables.
func (v *visitor) VisitSelectIntoVariables(ctx *mysqlparser.SelectIntoVariablesContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#selectIntoDumpFile.
func (v *visitor) VisitSelectIntoDumpFile(ctx *mysqlparser.SelectIntoDumpFileContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#selectIntoTextFile.
func (v *visitor) VisitSelectIntoTextFile(ctx *mysqlparser.SelectIntoTextFileContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#selectFieldsInto.
func (v *visitor) VisitSelectFieldsInto(ctx *mysqlparser.SelectFieldsIntoContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#selectLinesInto.
func (v *visitor) VisitSelectLinesInto(ctx *mysqlparser.SelectLinesIntoContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#fromClause.
func (v *visitor) VisitFromClause(ctx *mysqlparser.FromClauseContext) any {
	return v.VisitChildren(ctx)
}

// VisitWhereClause implements parser.MySqlParserVisitor.
func (v *visitor) VisitWhereClause(ctx *mysqlparser.WhereClauseContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#groupByClause.
func (v *visitor) VisitGroupByClause(ctx *mysqlparser.GroupByClauseContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#havingClause.
func (v *visitor) VisitHavingClause(ctx *mysqlparser.HavingClauseContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#windowClause.
func (v *visitor) VisitWindowClause(ctx *mysqlparser.WindowClauseContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#groupByItem.
func (v *visitor) VisitGroupByItem(ctx *mysqlparser.GroupByItemContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#limitClause.
func (v *visitor) VisitLimitClause(ctx *mysqlparser.LimitClauseContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#limitClauseAtom.
func (v *visitor) VisitLimitClauseAtom(ctx *mysqlparser.LimitClauseAtomContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#startTransaction.
func (v *visitor) VisitStartTransaction(ctx *mysqlparser.StartTransactionContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#beginWork.
func (v *visitor) VisitBeginWork(ctx *mysqlparser.BeginWorkContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#commitWork.
func (v *visitor) VisitCommitWork(ctx *mysqlparser.CommitWorkContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#rollbackWork.
func (v *visitor) VisitRollbackWork(ctx *mysqlparser.RollbackWorkContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#savepointStatement.
func (v *visitor) VisitSavepointStatement(ctx *mysqlparser.SavepointStatementContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#rollbackStatement.
func (v *visitor) VisitRollbackStatement(ctx *mysqlparser.RollbackStatementContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#releaseStatement.
func (v *visitor) VisitReleaseStatement(ctx *mysqlparser.ReleaseStatementContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#lockTables.
func (v *visitor) VisitLockTables(ctx *mysqlparser.LockTablesContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#unlockTables.
func (v *visitor) VisitUnlockTables(ctx *mysqlparser.UnlockTablesContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#setAutocommitStatement.
func (v *visitor) VisitSetAutocommitStatement(ctx *mysqlparser.SetAutocommitStatementContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#setTransactionStatement.
func (v *visitor) VisitSetTransactionStatement(ctx *mysqlparser.SetTransactionStatementContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#transactionMode.
func (v *visitor) VisitTransactionMode(ctx *mysqlparser.TransactionModeContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#lockTableElement.
func (v *visitor) VisitLockTableElement(ctx *mysqlparser.LockTableElementContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#lockAction.
func (v *visitor) VisitLockAction(ctx *mysqlparser.LockActionContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#transactionOption.
func (v *visitor) VisitTransactionOption(ctx *mysqlparser.TransactionOptionContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#transactionLevel.
func (v *visitor) VisitTransactionLevel(ctx *mysqlparser.TransactionLevelContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#changeMaster.
func (v *visitor) VisitChangeMaster(ctx *mysqlparser.ChangeMasterContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#changeReplicationFilter.
func (v *visitor) VisitChangeReplicationFilter(ctx *mysqlparser.ChangeReplicationFilterContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#purgeBinaryLogs.
func (v *visitor) VisitPurgeBinaryLogs(ctx *mysqlparser.PurgeBinaryLogsContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#resetMaster.
func (v *visitor) VisitResetMaster(ctx *mysqlparser.ResetMasterContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#resetSlave.
func (v *visitor) VisitResetSlave(ctx *mysqlparser.ResetSlaveContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#startSlave.
func (v *visitor) VisitStartSlave(ctx *mysqlparser.StartSlaveContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#stopSlave.
func (v *visitor) VisitStopSlave(ctx *mysqlparser.StopSlaveContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#startGroupReplication.
func (v *visitor) VisitStartGroupReplication(ctx *mysqlparser.StartGroupReplicationContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#stopGroupReplication.
func (v *visitor) VisitStopGroupReplication(ctx *mysqlparser.StopGroupReplicationContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#masterStringOption.
func (v *visitor) VisitMasterStringOption(ctx *mysqlparser.MasterStringOptionContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#masterDecimalOption.
func (v *visitor) VisitMasterDecimalOption(ctx *mysqlparser.MasterDecimalOptionContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#masterBoolOption.
func (v *visitor) VisitMasterBoolOption(ctx *mysqlparser.MasterBoolOptionContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#masterRealOption.
func (v *visitor) VisitMasterRealOption(ctx *mysqlparser.MasterRealOptionContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#masterUidListOption.
func (v *visitor) VisitMasterUidListOption(ctx *mysqlparser.MasterUidListOptionContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#stringMasterOption.
func (v *visitor) VisitStringMasterOption(ctx *mysqlparser.StringMasterOptionContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#decimalMasterOption.
func (v *visitor) VisitDecimalMasterOption(ctx *mysqlparser.DecimalMasterOptionContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#boolMasterOption.
func (v *visitor) VisitBoolMasterOption(ctx *mysqlparser.BoolMasterOptionContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#channelOption.
func (v *visitor) VisitChannelOption(ctx *mysqlparser.ChannelOptionContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#doDbReplication.
func (v *visitor) VisitDoDbReplication(ctx *mysqlparser.DoDbReplicationContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#ignoreDbReplication.
func (v *visitor) VisitIgnoreDbReplication(ctx *mysqlparser.IgnoreDbReplicationContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#doTableReplication.
func (v *visitor) VisitDoTableReplication(ctx *mysqlparser.DoTableReplicationContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#ignoreTableReplication.
func (v *visitor) VisitIgnoreTableReplication(ctx *mysqlparser.IgnoreTableReplicationContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#wildDoTableReplication.
func (v *visitor) VisitWildDoTableReplication(ctx *mysqlparser.WildDoTableReplicationContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#wildIgnoreTableReplication.
func (v *visitor) VisitWildIgnoreTableReplication(ctx *mysqlparser.WildIgnoreTableReplicationContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#rewriteDbReplication.
func (v *visitor) VisitRewriteDbReplication(ctx *mysqlparser.RewriteDbReplicationContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#tablePair.
func (v *visitor) VisitTablePair(ctx *mysqlparser.TablePairContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#threadType.
func (v *visitor) VisitThreadType(ctx *mysqlparser.ThreadTypeContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#gtidsUntilOption.
func (v *visitor) VisitGtidsUntilOption(ctx *mysqlparser.GtidsUntilOptionContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#masterLogUntilOption.
func (v *visitor) VisitMasterLogUntilOption(ctx *mysqlparser.MasterLogUntilOptionContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#relayLogUntilOption.
func (v *visitor) VisitRelayLogUntilOption(ctx *mysqlparser.RelayLogUntilOptionContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#sqlGapsUntilOption.
func (v *visitor) VisitSqlGapsUntilOption(ctx *mysqlparser.SqlGapsUntilOptionContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#userConnectionOption.
func (v *visitor) VisitUserConnectionOption(ctx *mysqlparser.UserConnectionOptionContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#passwordConnectionOption.
func (v *visitor) VisitPasswordConnectionOption(ctx *mysqlparser.PasswordConnectionOptionContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#defaultAuthConnectionOption.
func (v *visitor) VisitDefaultAuthConnectionOption(ctx *mysqlparser.DefaultAuthConnectionOptionContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#pluginDirConnectionOption.
func (v *visitor) VisitPluginDirConnectionOption(ctx *mysqlparser.PluginDirConnectionOptionContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#gtuidSet.
func (v *visitor) VisitGtuidSet(ctx *mysqlparser.GtuidSetContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#xaStartTransaction.
func (v *visitor) VisitXaStartTransaction(ctx *mysqlparser.XaStartTransactionContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#xaEndTransaction.
func (v *visitor) VisitXaEndTransaction(ctx *mysqlparser.XaEndTransactionContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#xaPrepareStatement.
func (v *visitor) VisitXaPrepareStatement(ctx *mysqlparser.XaPrepareStatementContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#xaCommitWork.
func (v *visitor) VisitXaCommitWork(ctx *mysqlparser.XaCommitWorkContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#xaRollbackWork.
func (v *visitor) VisitXaRollbackWork(ctx *mysqlparser.XaRollbackWorkContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#xaRecoverWork.
func (v *visitor) VisitXaRecoverWork(ctx *mysqlparser.XaRecoverWorkContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#prepareStatement.
func (v *visitor) VisitPrepareStatement(ctx *mysqlparser.PrepareStatementContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#executeStatement.
func (v *visitor) VisitExecuteStatement(ctx *mysqlparser.ExecuteStatementContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#deallocatePrepare.
func (v *visitor) VisitDeallocatePrepare(ctx *mysqlparser.DeallocatePrepareContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#routineBody.
func (v *visitor) VisitRoutineBody(ctx *mysqlparser.RoutineBodyContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#blockStatement.
func (v *visitor) VisitBlockStatement(ctx *mysqlparser.BlockStatementContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#caseStatement.
func (v *visitor) VisitCaseStatement(ctx *mysqlparser.CaseStatementContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#ifStatement.
func (v *visitor) VisitIfStatement(ctx *mysqlparser.IfStatementContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#iterateStatement.
func (v *visitor) VisitIterateStatement(ctx *mysqlparser.IterateStatementContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#leaveStatement.
func (v *visitor) VisitLeaveStatement(ctx *mysqlparser.LeaveStatementContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#loopStatement.
func (v *visitor) VisitLoopStatement(ctx *mysqlparser.LoopStatementContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#repeatStatement.
func (v *visitor) VisitRepeatStatement(ctx *mysqlparser.RepeatStatementContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#returnStatement.
func (v *visitor) VisitReturnStatement(ctx *mysqlparser.ReturnStatementContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#whileStatement.
func (v *visitor) VisitWhileStatement(ctx *mysqlparser.WhileStatementContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#CloseCursor.
func (v *visitor) VisitCloseCursor(ctx *mysqlparser.CloseCursorContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#FetchCursor.
func (v *visitor) VisitFetchCursor(ctx *mysqlparser.FetchCursorContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#OpenCursor.
func (v *visitor) VisitOpenCursor(ctx *mysqlparser.OpenCursorContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#declareVariable.
func (v *visitor) VisitDeclareVariable(ctx *mysqlparser.DeclareVariableContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#declareCondition.
func (v *visitor) VisitDeclareCondition(ctx *mysqlparser.DeclareConditionContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#declareCursor.
func (v *visitor) VisitDeclareCursor(ctx *mysqlparser.DeclareCursorContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#declareHandler.
func (v *visitor) VisitDeclareHandler(ctx *mysqlparser.DeclareHandlerContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#handlerConditionCode.
func (v *visitor) VisitHandlerConditionCode(ctx *mysqlparser.HandlerConditionCodeContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#handlerConditionState.
func (v *visitor) VisitHandlerConditionState(ctx *mysqlparser.HandlerConditionStateContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#handlerConditionName.
func (v *visitor) VisitHandlerConditionName(ctx *mysqlparser.HandlerConditionNameContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#handlerConditionWarning.
func (v *visitor) VisitHandlerConditionWarning(ctx *mysqlparser.HandlerConditionWarningContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#handlerConditionNotfound.
func (v *visitor) VisitHandlerConditionNotfound(ctx *mysqlparser.HandlerConditionNotfoundContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#handlerConditionException.
func (v *visitor) VisitHandlerConditionException(ctx *mysqlparser.HandlerConditionExceptionContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#procedureSqlStatement.
func (v *visitor) VisitProcedureSqlStatement(ctx *mysqlparser.ProcedureSqlStatementContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#caseAlternative.
func (v *visitor) VisitCaseAlternative(ctx *mysqlparser.CaseAlternativeContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#elifAlternative.
func (v *visitor) VisitElifAlternative(ctx *mysqlparser.ElifAlternativeContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#alterUserMysqlV56.
func (v *visitor) VisitAlterUserMysqlV56(ctx *mysqlparser.AlterUserMysqlV56Context) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#alterUserMysqlV80.
func (v *visitor) VisitAlterUserMysqlV80(ctx *mysqlparser.AlterUserMysqlV80Context) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#createUserMysqlV56.
func (v *visitor) VisitCreateUserMysqlV56(ctx *mysqlparser.CreateUserMysqlV56Context) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#createUserMysqlV80.
func (v *visitor) VisitCreateUserMysqlV80(ctx *mysqlparser.CreateUserMysqlV80Context) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#dropUser.
func (v *visitor) VisitDropUser(ctx *mysqlparser.DropUserContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#grantStatement.
func (v *visitor) VisitGrantStatement(ctx *mysqlparser.GrantStatementContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#roleOption.
func (v *visitor) VisitRoleOption(ctx *mysqlparser.RoleOptionContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#grantProxy.
func (v *visitor) VisitGrantProxy(ctx *mysqlparser.GrantProxyContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#renameUser.
func (v *visitor) VisitRenameUser(ctx *mysqlparser.RenameUserContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#detailRevoke.
func (v *visitor) VisitDetailRevoke(ctx *mysqlparser.DetailRevokeContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#shortRevoke.
func (v *visitor) VisitShortRevoke(ctx *mysqlparser.ShortRevokeContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#roleRevoke.
func (v *visitor) VisitRoleRevoke(ctx *mysqlparser.RoleRevokeContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#revokeProxy.
func (v *visitor) VisitRevokeProxy(ctx *mysqlparser.RevokeProxyContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#setPasswordStatement.
func (v *visitor) VisitSetPasswordStatement(ctx *mysqlparser.SetPasswordStatementContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#userSpecification.
func (v *visitor) VisitUserSpecification(ctx *mysqlparser.UserSpecificationContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#hashAuthOption.
func (v *visitor) VisitHashAuthOption(ctx *mysqlparser.HashAuthOptionContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#stringAuthOption.
func (v *visitor) VisitStringAuthOption(ctx *mysqlparser.StringAuthOptionContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#moduleAuthOption.
func (v *visitor) VisitModuleAuthOption(ctx *mysqlparser.ModuleAuthOptionContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#simpleAuthOption.
func (v *visitor) VisitSimpleAuthOption(ctx *mysqlparser.SimpleAuthOptionContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#module.
func (v *visitor) VisitModule(ctx *mysqlparser.ModuleContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#passwordModuleOption.
func (v *visitor) VisitPasswordModuleOption(ctx *mysqlparser.PasswordModuleOptionContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#tlsOption.
func (v *visitor) VisitTlsOption(ctx *mysqlparser.TlsOptionContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#userResourceOption.
func (v *visitor) VisitUserResourceOption(ctx *mysqlparser.UserResourceOptionContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#userPasswordOption.
func (v *visitor) VisitUserPasswordOption(ctx *mysqlparser.UserPasswordOptionContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#userLockOption.
func (v *visitor) VisitUserLockOption(ctx *mysqlparser.UserLockOptionContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#privelegeClause.
func (v *visitor) VisitPrivelegeClause(ctx *mysqlparser.PrivelegeClauseContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#privilege.
func (v *visitor) VisitPrivilege(ctx *mysqlparser.PrivilegeContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#currentSchemaPriviLevel.
func (v *visitor) VisitCurrentSchemaPriviLevel(ctx *mysqlparser.CurrentSchemaPriviLevelContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#globalPrivLevel.
func (v *visitor) VisitGlobalPrivLevel(ctx *mysqlparser.GlobalPrivLevelContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#definiteSchemaPrivLevel.
func (v *visitor) VisitDefiniteSchemaPrivLevel(ctx *mysqlparser.DefiniteSchemaPrivLevelContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#definiteFullTablePrivLevel.
func (v *visitor) VisitDefiniteFullTablePrivLevel(ctx *mysqlparser.DefiniteFullTablePrivLevelContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#definiteFullTablePrivLevel2.
func (v *visitor) VisitDefiniteFullTablePrivLevel2(ctx *mysqlparser.DefiniteFullTablePrivLevel2Context) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#definiteTablePrivLevel.
func (v *visitor) VisitDefiniteTablePrivLevel(ctx *mysqlparser.DefiniteTablePrivLevelContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#renameUserClause.
func (v *visitor) VisitRenameUserClause(ctx *mysqlparser.RenameUserClauseContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#analyzeTable.
func (v *visitor) VisitAnalyzeTable(ctx *mysqlparser.AnalyzeTableContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#checkTable.
func (v *visitor) VisitCheckTable(ctx *mysqlparser.CheckTableContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#checksumTable.
func (v *visitor) VisitChecksumTable(ctx *mysqlparser.ChecksumTableContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#optimizeTable.
func (v *visitor) VisitOptimizeTable(ctx *mysqlparser.OptimizeTableContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#repairTable.
func (v *visitor) VisitRepairTable(ctx *mysqlparser.RepairTableContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#checkTableOption.
func (v *visitor) VisitCheckTableOption(ctx *mysqlparser.CheckTableOptionContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#createUdfunction.
func (v *visitor) VisitCreateUdfunction(ctx *mysqlparser.CreateUdfunctionContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#installPlugin.
func (v *visitor) VisitInstallPlugin(ctx *mysqlparser.InstallPluginContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#uninstallPlugin.
func (v *visitor) VisitUninstallPlugin(ctx *mysqlparser.UninstallPluginContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#setVariable.
func (v *visitor) VisitSetVariable(ctx *mysqlparser.SetVariableContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#setCharset.
func (v *visitor) VisitSetCharset(ctx *mysqlparser.SetCharsetContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#setNames.
func (v *visitor) VisitSetNames(ctx *mysqlparser.SetNamesContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#setPassword.
func (v *visitor) VisitSetPassword(ctx *mysqlparser.SetPasswordContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#setTransaction.
func (v *visitor) VisitSetTransaction(ctx *mysqlparser.SetTransactionContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#setAutocommit.
func (v *visitor) VisitSetAutocommit(ctx *mysqlparser.SetAutocommitContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#setNewValueInsideTrigger.
func (v *visitor) VisitSetNewValueInsideTrigger(ctx *mysqlparser.SetNewValueInsideTriggerContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#showMasterLogs.
func (v *visitor) VisitShowMasterLogs(ctx *mysqlparser.ShowMasterLogsContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#showLogEvents.
func (v *visitor) VisitShowLogEvents(ctx *mysqlparser.ShowLogEventsContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#showObjectFilter.
func (v *visitor) VisitShowObjectFilter(ctx *mysqlparser.ShowObjectFilterContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#showColumns.
func (v *visitor) VisitShowColumns(ctx *mysqlparser.ShowColumnsContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#showCreateDb.
func (v *visitor) VisitShowCreateDb(ctx *mysqlparser.ShowCreateDbContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#showCreateFullIdObject.
func (v *visitor) VisitShowCreateFullIdObject(ctx *mysqlparser.ShowCreateFullIdObjectContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#showCreateUser.
func (v *visitor) VisitShowCreateUser(ctx *mysqlparser.ShowCreateUserContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#showEngine.
func (v *visitor) VisitShowEngine(ctx *mysqlparser.ShowEngineContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#showGlobalInfo.
func (v *visitor) VisitShowGlobalInfo(ctx *mysqlparser.ShowGlobalInfoContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#showErrors.
func (v *visitor) VisitShowErrors(ctx *mysqlparser.ShowErrorsContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#showCountErrors.
func (v *visitor) VisitShowCountErrors(ctx *mysqlparser.ShowCountErrorsContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#showSchemaFilter.
func (v *visitor) VisitShowSchemaFilter(ctx *mysqlparser.ShowSchemaFilterContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#showRoutine.
func (v *visitor) VisitShowRoutine(ctx *mysqlparser.ShowRoutineContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#showGrants.
func (v *visitor) VisitShowGrants(ctx *mysqlparser.ShowGrantsContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#showIndexes.
func (v *visitor) VisitShowIndexes(ctx *mysqlparser.ShowIndexesContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#showOpenTables.
func (v *visitor) VisitShowOpenTables(ctx *mysqlparser.ShowOpenTablesContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#showProfile.
func (v *visitor) VisitShowProfile(ctx *mysqlparser.ShowProfileContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#showSlaveStatus.
func (v *visitor) VisitShowSlaveStatus(ctx *mysqlparser.ShowSlaveStatusContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#variableClause.
func (v *visitor) VisitVariableClause(ctx *mysqlparser.VariableClauseContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#showCommonEntity.
func (v *visitor) VisitShowCommonEntity(ctx *mysqlparser.ShowCommonEntityContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#showFilter.
func (v *visitor) VisitShowFilter(ctx *mysqlparser.ShowFilterContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#showGlobalInfoClause.
func (v *visitor) VisitShowGlobalInfoClause(ctx *mysqlparser.ShowGlobalInfoClauseContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#showSchemaEntity.
func (v *visitor) VisitShowSchemaEntity(ctx *mysqlparser.ShowSchemaEntityContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#showProfileType.
func (v *visitor) VisitShowProfileType(ctx *mysqlparser.ShowProfileTypeContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#binlogStatement.
func (v *visitor) VisitBinlogStatement(ctx *mysqlparser.BinlogStatementContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#cacheIndexStatement.
func (v *visitor) VisitCacheIndexStatement(ctx *mysqlparser.CacheIndexStatementContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#flushStatement.
func (v *visitor) VisitFlushStatement(ctx *mysqlparser.FlushStatementContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#killStatement.
func (v *visitor) VisitKillStatement(ctx *mysqlparser.KillStatementContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#loadIndexIntoCache.
func (v *visitor) VisitLoadIndexIntoCache(ctx *mysqlparser.LoadIndexIntoCacheContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#resetStatement.
func (v *visitor) VisitResetStatement(ctx *mysqlparser.ResetStatementContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#shutdownStatement.
func (v *visitor) VisitShutdownStatement(ctx *mysqlparser.ShutdownStatementContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#tableIndexes.
func (v *visitor) VisitTableIndexes(ctx *mysqlparser.TableIndexesContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#simpleFlushOption.
func (v *visitor) VisitSimpleFlushOption(ctx *mysqlparser.SimpleFlushOptionContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#channelFlushOption.
func (v *visitor) VisitChannelFlushOption(ctx *mysqlparser.ChannelFlushOptionContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#tableFlushOption.
func (v *visitor) VisitTableFlushOption(ctx *mysqlparser.TableFlushOptionContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#flushTableOption.
func (v *visitor) VisitFlushTableOption(ctx *mysqlparser.FlushTableOptionContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#loadedTableIndexes.
func (v *visitor) VisitLoadedTableIndexes(ctx *mysqlparser.LoadedTableIndexesContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#simpleDescribeStatement.
func (v *visitor) VisitSimpleDescribeStatement(ctx *mysqlparser.SimpleDescribeStatementContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#fullDescribeStatement.
func (v *visitor) VisitFullDescribeStatement(ctx *mysqlparser.FullDescribeStatementContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#helpStatement.
func (v *visitor) VisitHelpStatement(ctx *mysqlparser.HelpStatementContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#useStatement.
func (v *visitor) VisitUseStatement(ctx *mysqlparser.UseStatementContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#signalStatement.
func (v *visitor) VisitSignalStatement(ctx *mysqlparser.SignalStatementContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#resignalStatement.
func (v *visitor) VisitResignalStatement(ctx *mysqlparser.ResignalStatementContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#signalConditionInformation.
func (v *visitor) VisitSignalConditionInformation(ctx *mysqlparser.SignalConditionInformationContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#diagnosticsStatement.
func (v *visitor) VisitDiagnosticsStatement(ctx *mysqlparser.DiagnosticsStatementContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#diagnosticsConditionInformationName.
func (v *visitor) VisitDiagnosticsConditionInformationName(ctx *mysqlparser.DiagnosticsConditionInformationNameContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#describeStatements.
func (v *visitor) VisitDescribeStatements(ctx *mysqlparser.DescribeStatementsContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#describeConnection.
func (v *visitor) VisitDescribeConnection(ctx *mysqlparser.DescribeConnectionContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#fullId.
func (v *visitor) VisitFullId(ctx *mysqlparser.FullIdContext) any {
	v.VisitChildren(ctx)
	if v.Err != nil {
		return nil
	}

	var name string
	if dotted := ctx.DottedId(); dotted != nil {
		name = getDottedIDName(dotted)
	} else {
		name = getUIDName(ctx.Uid())
	}

	v.MaybeSetNodeName(ctx, name)
	return nil
}

// Visit a parse tree produced by MySqlParser#tableName.
func (v *visitor) VisitTableName(ctx *mysqlparser.TableNameContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#roleName.
func (v *visitor) VisitRoleName(ctx *mysqlparser.RoleNameContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#fullColumnName.
func (v *visitor) VisitFullColumnName(ctx *mysqlparser.FullColumnNameContext) any {
	v.VisitChildren(ctx)
	if v.Err != nil {
		return nil
	}

	var table, column string

	allDotted := ctx.AllDottedId()
	switch len(allDotted) {
	case 0:
		column = getUIDName(ctx.Uid())
	case 1:
		column = getDottedIDName(allDotted[0])
		table = getUIDName(ctx.Uid())
	case 2:
		column = getDottedIDName(allDotted[1])
		table = getDottedIDName(allDotted[0])
	}

	v.MaybeSetNodeName(ctx, column)
	v.UpdateInfo(NodeInfo{
		Node:            ctx,
		ExprDescription: "Column Name",
		Type:            makeRef(v.Sources, table, column),
	})

	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#indexColumnName.
func (v *visitor) VisitIndexColumnName(ctx *mysqlparser.IndexColumnNameContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#userName.
func (v *visitor) VisitUserName(ctx *mysqlparser.UserNameContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#mysqlVariable.
func (v *visitor) VisitMysqlVariable(ctx *mysqlparser.MysqlVariableContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#charsetName.
func (v *visitor) VisitCharsetName(ctx *mysqlparser.CharsetNameContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#collationName.
func (v *visitor) VisitCollationName(ctx *mysqlparser.CollationNameContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#engineName.
func (v *visitor) VisitEngineName(ctx *mysqlparser.EngineNameContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#uuidSet.
func (v *visitor) VisitUuidSet(ctx *mysqlparser.UuidSetContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#xid.
func (v *visitor) VisitXid(ctx *mysqlparser.XidContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#xuidStringId.
func (v *visitor) VisitXuidStringId(ctx *mysqlparser.XuidStringIdContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#authPlugin.
func (v *visitor) VisitAuthPlugin(ctx *mysqlparser.AuthPluginContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#uid.
func (v *visitor) VisitUid(ctx *mysqlparser.UidContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#simpleId.
func (v *visitor) VisitSimpleId(ctx *mysqlparser.SimpleIdContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#dottedId.
func (v *visitor) VisitDottedId(ctx *mysqlparser.DottedIdContext) any {
	v.VisitChildren(ctx)
	if v.Err != nil {
		return nil
	}

	v.MaybeSetNodeName(ctx, getDottedIDName(ctx)) // remove leading dot

	return nil
}

// Visit a parse tree produced by MySqlParser#decimalLiteral.
func (v *visitor) VisitDecimalLiteral(ctx *mysqlparser.DecimalLiteralContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#fileSizeLiteral.
func (v *visitor) VisitFileSizeLiteral(ctx *mysqlparser.FileSizeLiteralContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#stringLiteral.
func (v *visitor) VisitStringLiteral(ctx *mysqlparser.StringLiteralContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#booleanLiteral.
func (v *visitor) VisitBooleanLiteral(ctx *mysqlparser.BooleanLiteralContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#hexadecimalLiteral.
func (v *visitor) VisitHexadecimalLiteral(ctx *mysqlparser.HexadecimalLiteralContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#nullNotnull.
func (v *visitor) VisitNullNotnull(ctx *mysqlparser.NullNotnullContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#constant.
func (v *visitor) VisitConstant(ctx *mysqlparser.ConstantContext) any {
	v.VisitChildren(ctx)
	if v.Err != nil {
		return nil
	}

	var DBType NodeType

	switch {
	case ctx.StringLiteral() != nil:
		DBType = knownTypeNotNull("varchar")
		txt := ctx.GetText()
		txt = txt[1 : len(txt)-1] // remove quotes
		v.MaybeSetNodeName(ctx, txt)

	case ctx.DecimalLiteral() != nil: // decimal number
		typ := ""
		txt := ctx.DecimalLiteral().GetText()
		switch {
		case strings.ContainsAny(ctx.GetText(), "eE"): // scientific notation
			typ = "float"
		case strings.Contains(txt, "."): // decimal number
			typ = "double"
		default: // integer number
			typ = "int"
			_, err := strconv.ParseInt(txt, 10, 32)
			if errors.Is(err, strconv.ErrRange) { // too large number
				typ = "bigint"
			}
		}

		if ctx.MINUS() == nil { // signed number
			typ += " unsigned"
		}

		DBType = knownTypeNotNull(typ)

	case ctx.BIT_STRING() != nil: // bit string
		DBType = knownTypeNotNull("varbinary") // Seen as a bit string

	case ctx.HexadecimalLiteral() != nil: // hexadecimal number
		DBType = knownTypeNotNull("varbinary") // Seen as a bit string

	case ctx.BooleanLiteral() != nil: // boolean number
		DBType = knownTypeNotNull("boolean")
		v.MaybeSetNodeName(ctx, ctx.GetText())

	case ctx.GetNullLiteral() != nil: // null
		DBType = knownTypeNull("")
		v.MaybeSetNodeName(ctx, "NULL")
	}

	info := NodeInfo{
		Node:            ctx,
		ExprDescription: "Constant",
	}

	if len(DBType.DBType) > 0 {
		info.Type = []NodeType{DBType}
	}

	v.UpdateInfo(info)
	return nil
}

// Visit a parse tree produced by MySqlParser#stringDataType.
func (v *visitor) VisitStringDataType(ctx *mysqlparser.StringDataTypeContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#nationalVaryingStringDataType.
func (v *visitor) VisitNationalVaryingStringDataType(ctx *mysqlparser.NationalVaryingStringDataTypeContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#nationalStringDataType.
func (v *visitor) VisitNationalStringDataType(ctx *mysqlparser.NationalStringDataTypeContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#dimensionDataType.
func (v *visitor) VisitDimensionDataType(ctx *mysqlparser.DimensionDataTypeContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#simpleDataType.
func (v *visitor) VisitSimpleDataType(ctx *mysqlparser.SimpleDataTypeContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#collectionDataType.
func (v *visitor) VisitCollectionDataType(ctx *mysqlparser.CollectionDataTypeContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#spatialDataType.
func (v *visitor) VisitSpatialDataType(ctx *mysqlparser.SpatialDataTypeContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#longVarcharDataType.
func (v *visitor) VisitLongVarcharDataType(ctx *mysqlparser.LongVarcharDataTypeContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#longVarbinaryDataType.
func (v *visitor) VisitLongVarbinaryDataType(ctx *mysqlparser.LongVarbinaryDataTypeContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#collectionOptions.
func (v *visitor) VisitCollectionOptions(ctx *mysqlparser.CollectionOptionsContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#convertedDataType.
func (v *visitor) VisitConvertedDataType(ctx *mysqlparser.ConvertedDataTypeContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#lengthOneDimension.
func (v *visitor) VisitLengthOneDimension(ctx *mysqlparser.LengthOneDimensionContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#lengthTwoDimension.
func (v *visitor) VisitLengthTwoDimension(ctx *mysqlparser.LengthTwoDimensionContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#lengthTwoOptionalDimension.
func (v *visitor) VisitLengthTwoOptionalDimension(ctx *mysqlparser.LengthTwoOptionalDimensionContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#uidList.
func (v *visitor) VisitUidList(ctx *mysqlparser.UidListContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#fullColumnNameList.
func (v *visitor) VisitFullColumnNameList(ctx *mysqlparser.FullColumnNameListContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#tables.
func (v *visitor) VisitTables(ctx *mysqlparser.TablesContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#indexColumnNames.
func (v *visitor) VisitIndexColumnNames(ctx *mysqlparser.IndexColumnNamesContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#expressions.
func (v *visitor) VisitExpressions(ctx *mysqlparser.ExpressionsContext) any {
	v.VisitChildren(ctx)
	if v.Err != nil {
		return nil
	}

	return nil
}

// Visit a parse tree produced by MySqlParser#expressionsWithDefaults.
func (v *visitor) VisitExpressionsWithDefaults(ctx *mysqlparser.ExpressionsWithDefaultsContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#constants.
func (v *visitor) VisitConstants(ctx *mysqlparser.ConstantsContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#simpleStrings.
func (v *visitor) VisitSimpleStrings(ctx *mysqlparser.SimpleStringsContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#userVariables.
func (v *visitor) VisitUserVariables(ctx *mysqlparser.UserVariablesContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#defaultValue.
func (v *visitor) VisitDefaultValue(ctx *mysqlparser.DefaultValueContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#currentTimestamp.
func (v *visitor) VisitCurrentTimestamp(ctx *mysqlparser.CurrentTimestampContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#expressionOrDefault.
func (v *visitor) VisitExpressionOrDefault(ctx *mysqlparser.ExpressionOrDefaultContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#ifExists.
func (v *visitor) VisitIfExists(ctx *mysqlparser.IfExistsContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#ifNotExists.
func (v *visitor) VisitIfNotExists(ctx *mysqlparser.IfNotExistsContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#orReplace.
func (v *visitor) VisitOrReplace(ctx *mysqlparser.OrReplaceContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#waitNowaitClause.
func (v *visitor) VisitWaitNowaitClause(ctx *mysqlparser.WaitNowaitClauseContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#specificFunctionCall.
func (v *visitor) VisitSpecificFunctionCall(ctx *mysqlparser.SpecificFunctionCallContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#aggregateFunctionCall.
func (v *visitor) VisitAggregateFunctionCall(ctx *mysqlparser.AggregateFunctionCallContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#nonAggregateFunctionCall.
func (v *visitor) VisitNonAggregateFunctionCall(ctx *mysqlparser.NonAggregateFunctionCallContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#scalarFunctionCall.
func (v *visitor) VisitScalarFunctionCall(ctx *mysqlparser.ScalarFunctionCallContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#udfFunctionCall.
func (v *visitor) VisitUdfFunctionCall(ctx *mysqlparser.UdfFunctionCallContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#passwordFunctionCall.
func (v *visitor) VisitPasswordFunctionCall(ctx *mysqlparser.PasswordFunctionCallContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#simpleFunctionCall.
func (v *visitor) VisitSimpleFunctionCall(ctx *mysqlparser.SimpleFunctionCallContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#dataTypeFunctionCall.
func (v *visitor) VisitDataTypeFunctionCall(ctx *mysqlparser.DataTypeFunctionCallContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#valuesFunctionCall.
func (v *visitor) VisitValuesFunctionCall(ctx *mysqlparser.ValuesFunctionCallContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#caseExpressionFunctionCall.
func (v *visitor) VisitCaseExpressionFunctionCall(ctx *mysqlparser.CaseExpressionFunctionCallContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#caseFunctionCall.
func (v *visitor) VisitCaseFunctionCall(ctx *mysqlparser.CaseFunctionCallContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#charFunctionCall.
func (v *visitor) VisitCharFunctionCall(ctx *mysqlparser.CharFunctionCallContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#positionFunctionCall.
func (v *visitor) VisitPositionFunctionCall(ctx *mysqlparser.PositionFunctionCallContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#substrFunctionCall.
func (v *visitor) VisitSubstrFunctionCall(ctx *mysqlparser.SubstrFunctionCallContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#trimFunctionCall.
func (v *visitor) VisitTrimFunctionCall(ctx *mysqlparser.TrimFunctionCallContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#weightFunctionCall.
func (v *visitor) VisitWeightFunctionCall(ctx *mysqlparser.WeightFunctionCallContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#extractFunctionCall.
func (v *visitor) VisitExtractFunctionCall(ctx *mysqlparser.ExtractFunctionCallContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#getFormatFunctionCall.
func (v *visitor) VisitGetFormatFunctionCall(ctx *mysqlparser.GetFormatFunctionCallContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#jsonValueFunctionCall.
func (v *visitor) VisitJsonValueFunctionCall(ctx *mysqlparser.JsonValueFunctionCallContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#caseFuncAlternative.
func (v *visitor) VisitCaseFuncAlternative(ctx *mysqlparser.CaseFuncAlternativeContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#levelWeightList.
func (v *visitor) VisitLevelWeightList(ctx *mysqlparser.LevelWeightListContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#levelWeightRange.
func (v *visitor) VisitLevelWeightRange(ctx *mysqlparser.LevelWeightRangeContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#levelInWeightListElement.
func (v *visitor) VisitLevelInWeightListElement(ctx *mysqlparser.LevelInWeightListElementContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#aggregateWindowedFunction.
func (v *visitor) VisitAggregateWindowedFunction(ctx *mysqlparser.AggregateWindowedFunctionContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#nonAggregateWindowedFunction.
func (v *visitor) VisitNonAggregateWindowedFunction(ctx *mysqlparser.NonAggregateWindowedFunctionContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#overClause.
func (v *visitor) VisitOverClause(ctx *mysqlparser.OverClauseContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#windowSpec.
func (v *visitor) VisitWindowSpec(ctx *mysqlparser.WindowSpecContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#windowName.
func (v *visitor) VisitWindowName(ctx *mysqlparser.WindowNameContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#frameClause.
func (v *visitor) VisitFrameClause(ctx *mysqlparser.FrameClauseContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#frameUnits.
func (v *visitor) VisitFrameUnits(ctx *mysqlparser.FrameUnitsContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#frameExtent.
func (v *visitor) VisitFrameExtent(ctx *mysqlparser.FrameExtentContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#frameBetween.
func (v *visitor) VisitFrameBetween(ctx *mysqlparser.FrameBetweenContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#frameRange.
func (v *visitor) VisitFrameRange(ctx *mysqlparser.FrameRangeContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#partitionClause.
func (v *visitor) VisitPartitionClause(ctx *mysqlparser.PartitionClauseContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#scalarFunctionName.
func (v *visitor) VisitScalarFunctionName(ctx *mysqlparser.ScalarFunctionNameContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#passwordFunctionClause.
func (v *visitor) VisitPasswordFunctionClause(ctx *mysqlparser.PasswordFunctionClauseContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#functionArgs.
func (v *visitor) VisitFunctionArgs(ctx *mysqlparser.FunctionArgsContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#functionArg.
func (v *visitor) VisitFunctionArg(ctx *mysqlparser.FunctionArgContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#isExpression.
func (v *visitor) VisitIsExpression(ctx *mysqlparser.IsExpressionContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#notExpression.
func (v *visitor) VisitNotExpression(ctx *mysqlparser.NotExpressionContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#logicalExpression.
func (v *visitor) VisitLogicalExpression(ctx *mysqlparser.LogicalExpressionContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#predicateExpression.
func (v *visitor) VisitPredicateExpression(ctx *mysqlparser.PredicateExpressionContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#soundsLikePredicate.
func (v *visitor) VisitSoundsLikePredicate(ctx *mysqlparser.SoundsLikePredicateContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#expressionAtomPredicate.
func (v *visitor) VisitExpressionAtomPredicate(ctx *mysqlparser.ExpressionAtomPredicateContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#subqueryComparisonPredicate.
func (v *visitor) VisitSubqueryComparisonPredicate(ctx *mysqlparser.SubqueryComparisonPredicateContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#jsonMemberOfPredicate.
func (v *visitor) VisitJsonMemberOfPredicate(ctx *mysqlparser.JsonMemberOfPredicateContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#binaryComparisonPredicate.
func (v *visitor) VisitBinaryComparisonPredicate(ctx *mysqlparser.BinaryComparisonPredicateContext) any {
	v.VisitChildren(ctx)
	if v.Err != nil {
		return nil
	}

	v.UpdateInfo(NodeInfo{
		Node:            ctx,
		ExprDescription: "Comparison",
		Type:            []NodeType{knownTypeNull("boolean")},
	})

	v.equateTypesAndNames(ctx.GetLeft(), ctx.GetRight())

	return nil
}

// Visit a parse tree produced by MySqlParser#inExpressions
func (v *visitor) VisitInExpressions(ctx *mysqlparser.InExpressionsContext) any {
	v.VisitChildren(ctx)
	if v.Err != nil {
		return nil
	}

	v.UpdateInfo(NodeInfo{
		Node:            ctx,
		ExprDescription: "IN",
		Type:            []NodeType{knownTypeNull("boolean")},
	})

	expressions := ctx.ExpressionList().Expressions()

	for _, expression := range expressions.AllExpression() {
		v.equateTypesAndNames(ctx.Predicate(), expression)
	}

	if all := expressions.AllExpression(); len(all) == 1 {
		v.StmtRules = append(v.StmtRules, internal.RecordPoints(
			all[0].GetStart().GetStart(),
			all[0].GetStop().GetStop(),
			func(start, end int) error {
				v.UpdateInfo(NodeInfo{
					Node:            expressions,
					ExprDescription: "Expressions",
					EditedPosition:  [2]int{start, end},
					CanBeMultiple:   true,
				})
				return nil
			},
		)...)
	}

	return nil
}

// Visit a parse tree produced by MySqlParser#inSubselect
func (v *visitor) VisitInSubSelect(ctx *mysqlparser.InSubSelectContext) any {
	v.VisitChildren(ctx)
	if v.Err != nil {
		return nil
	}

	return nil
}

// Visit a parse tree produced by MySqlParser#betweenPredicate.
func (v *visitor) VisitBetweenPredicate(ctx *mysqlparser.BetweenPredicateContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#isNullPredicate.
func (v *visitor) VisitIsNullPredicate(ctx *mysqlparser.IsNullPredicateContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#likePredicate.
func (v *visitor) VisitLikePredicate(ctx *mysqlparser.LikePredicateContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#regexpPredicate.
func (v *visitor) VisitRegexpPredicate(ctx *mysqlparser.RegexpPredicateContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#unaryExpressionAtom.
func (v *visitor) VisitUnaryExpressionAtom(ctx *mysqlparser.UnaryExpressionAtomContext) any {
	v.VisitChildren(ctx)
	if v.Err != nil {
		return nil
	}

	tokenTyp := ctx.UnaryOperator().GetOperator().GetTokenType()
	switch tokenTyp {
	case mysqlparser.MySqlParserPLUS:
		// Returns the same type as the operand
		v.UpdateInfo(NodeInfo{
			Node:            ctx,
			ExprDescription: "Unary Plus",
			ExprRef:         ctx.ExpressionAtom(),
		})

		v.UpdateInfo(NodeInfo{
			Node:            ctx.ExpressionAtom(),
			ExprDescription: "Unary Plus Expr",
			ExprRef:         ctx,
		})

	case mysqlparser.MySqlParserMINUS:
		// Always INTEGER, should be used with a numeric literal
		v.UpdateInfo(NodeInfo{
			Node:            ctx,
			ExprDescription: "Unary Minus",
			ExprRef:         ctx.ExpressionAtom(),
		})

		v.UpdateInfo(NodeInfo{
			Node:            ctx.ExpressionAtom(),
			ExprDescription: "Unary Minus Expr",
			Type: []NodeType{
				knownTypeNotNull("bigint"),
				knownTypeNotNull("decimal"),
				knownTypeNotNull("double"),
			},
		})

	case mysqlparser.MySqlParserNOT, mysqlparser.MySqlParserEXCLAMATION_SYMBOL:
		// Returns a BOOLEAN (should technically only be used with a boolean expression)
		v.UpdateInfo(NodeInfo{
			Node:            ctx,
			ExprDescription: "Unary NOT",
			Type:            []NodeType{knownTypeNotNull("boolean")},
		})

		v.UpdateInfo(NodeInfo{
			Node:            ctx.ExpressionAtom(),
			ExprDescription: "Unary NOT Expr",
			Type:            []NodeType{knownTypeNotNull("boolean")},
		})

	case mysqlparser.MySqlParserBIT_NOT_OP:
		// Bitwise NOT
		// Always INTEGER, should be used with a numeric literal
		v.UpdateInfo(NodeInfo{
			Node:            ctx,
			ExprDescription: "Unary Tilde",
			Type:            []NodeType{knownTypeNotNull("bigint")},
		})

		v.UpdateInfo(NodeInfo{
			Node:            ctx.ExpressionAtom(),
			ExprDescription: "Unary Tilde Expr",
			Type: []NodeType{
				knownTypeNotNull("varbinary"),
				knownTypeNotNull("bigint"),
			},
		})
	}

	return nil
}

// Visit a parse tree produced by MySqlParser#collateExpressionAtom.
func (v *visitor) VisitCollateExpressionAtom(ctx *mysqlparser.CollateExpressionAtomContext) any {
	v.UpdateInfo(NodeInfo{
		Node:            ctx,
		ExprDescription: "COLLATE",
		Type:            []NodeType{knownTypeNotNull("varchar")},
	})

	v.UpdateInfo(NodeInfo{
		Node:            ctx.ExpressionAtom(),
		ExprDescription: "COLLATE Expr",
		Type:            []NodeType{knownTypeNotNull("varchar")},
	})

	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#bindExpressionAtom.
func (v *visitor) VisitBindExpressionAtom(ctx *mysqlparser.BindExpressionAtomContext) any {
	v.VisitChildren(ctx)
	if v.Err != nil {
		return nil
	}

	info := NodeInfo{
		Node:            ctx,
		ExprDescription: "Bind",
	}

	v.SetArg(ctx)
	v.UpdateInfo(info)
	v.StmtRules = append(v.StmtRules, internal.RecordPoints(
		ctx.GetStart().GetStart(), ctx.GetStop().GetStop(),
		func(start, end int) error {
			v.UpdateInfo(NodeInfo{
				Node:           ctx,
				EditedPosition: [2]int{start, end},
			})
			return nil
		})...,
	)

	return nil
}

// Visit a parse tree produced by MySqlParser#mysqlVariableExpressionAtom.
func (v *visitor) VisitMysqlVariableExpressionAtom(ctx *mysqlparser.MysqlVariableExpressionAtomContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#nestedExpressionAtom.
func (v *visitor) VisitNestedExpressionAtom(ctx *mysqlparser.NestedExpressionAtomContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#nestedRowExpressionAtom.
func (v *visitor) VisitNestedRowExpressionAtom(ctx *mysqlparser.NestedRowExpressionAtomContext) any {
	v.VisitChildren(ctx)
	if v.Err != nil {
		return nil
	}

	v.StmtRules = append(v.StmtRules, internal.RecordPoints(
		ctx.GetStart().GetStart(),
		ctx.GetStop().GetStop(),
		func(start, end int) error {
			v.SetGroup(ctx)
			v.UpdateInfo(NodeInfo{
				Node:            ctx,
				ExprDescription: "Row",
				EditedPosition:  [2]int{start, end},
			})
			return nil
		},
	)...)

	return nil
}

// Visit a parse tree produced by MySqlParser#mathExpressionAtom.
func (v *visitor) VisitMathExpressionAtom(ctx *mysqlparser.MathExpressionAtomContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#existsExpressionAtom.
func (v *visitor) VisitExistsExpressionAtom(ctx *mysqlparser.ExistsExpressionAtomContext) any {
	v.UpdateInfo(NodeInfo{
		Node:            ctx,
		ExprDescription: "Exists",
		Type:            []NodeType{knownTypeNull("boolean")},
	})
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#intervalExpressionAtom.
func (v *visitor) VisitIntervalExpressionAtom(ctx *mysqlparser.IntervalExpressionAtomContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#jsonExpressionAtom.
func (v *visitor) VisitJsonExpressionAtom(ctx *mysqlparser.JsonExpressionAtomContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#subqueryExpressionAtom.
func (v *visitor) VisitSubqueryExpressionAtom(ctx *mysqlparser.SubqueryExpressionAtomContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#constantExpressionAtom.
func (v *visitor) VisitConstantExpressionAtom(ctx *mysqlparser.ConstantExpressionAtomContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#functionCallExpressionAtom.
func (v *visitor) VisitFunctionCallExpressionAtom(ctx *mysqlparser.FunctionCallExpressionAtomContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#binaryExpressionAtom.
func (v *visitor) VisitBinaryExpressionAtom(ctx *mysqlparser.BinaryExpressionAtomContext) any {
	v.VisitChildren(ctx)
	if v.Err != nil {
		return nil
	}

	v.UpdateInfo(NodeInfo{
		Node:            ctx,
		ExprDescription: "BINARY",
		Type:            []NodeType{knownTypeNotNull("varbinary")},
	})

	v.UpdateInfo(NodeInfo{
		Node:            ctx.ExpressionAtom(),
		ExprDescription: "BINARY Expr",
		Type:            []NodeType{knownTypeNotNull("varchar")},
	})

	return nil
}

// Visit a parse tree produced by MySqlParser#fullColumnNameExpressionAtom.
func (v *visitor) VisitFullColumnNameExpressionAtom(ctx *mysqlparser.FullColumnNameExpressionAtomContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#bitExpressionAtom.
func (v *visitor) VisitBitExpressionAtom(ctx *mysqlparser.BitExpressionAtomContext) any {
	return v.VisitChildren(ctx)
}

// VisitExpressionList implements parser.MySqlParserVisitor.
func (v *visitor) VisitExpressionList(ctx *mysqlparser.ExpressionListContext) any {
	v.VisitChildren(ctx)
	if v.Err != nil {
		return nil
	}

	v.StmtRules = append(v.StmtRules, internal.RecordPoints(
		ctx.GetStart().GetStart(),
		ctx.GetStop().GetStop(),
		func(start, end int) error {
			v.SetGroup(ctx)
			v.UpdateInfo(NodeInfo{
				Node:            ctx,
				ExprDescription: "List",
				EditedPosition:  [2]int{start, end},
			})
			return nil
		},
	)...)

	return nil
}

// VisitSubSelect implements parser.MySqlParserVisitor.
func (v *visitor) VisitSubSelect(ctx *mysqlparser.SubSelectContext) any {
	v.VisitChildren(ctx)
	if v.Err != nil {
		return nil
	}

	v.StmtRules = append(v.StmtRules, internal.RecordPoints(
		ctx.GetStart().GetStart(),
		ctx.GetStop().GetStop(),
		func(start, end int) error {
			v.SetGroup(ctx)
			v.UpdateInfo(NodeInfo{
				Node:            ctx,
				ExprDescription: "SubSelect",
				EditedPosition:  [2]int{start, end},
			})
			return nil
		},
	)...)

	v.querySources[antlrhelpers.Key(ctx)] = v.querySources[antlrhelpers.Key(ctx.SelectStatement())]

	return nil
}

// Visit a parse tree produced by MySqlParser#unaryOperator.
func (v *visitor) VisitUnaryOperator(ctx *mysqlparser.UnaryOperatorContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#comparisonOperator.
func (v *visitor) VisitComparisonOperator(ctx *mysqlparser.ComparisonOperatorContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#logicalOperator.
func (v *visitor) VisitLogicalOperator(ctx *mysqlparser.LogicalOperatorContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#bitOperator.
func (v *visitor) VisitBitOperator(ctx *mysqlparser.BitOperatorContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#mathOperator.
func (v *visitor) VisitMathOperator(ctx *mysqlparser.MathOperatorContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#jsonOperator.
func (v *visitor) VisitJsonOperator(ctx *mysqlparser.JsonOperatorContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#charsetNameBase.
func (v *visitor) VisitCharsetNameBase(ctx *mysqlparser.CharsetNameBaseContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#transactionLevelBase.
func (v *visitor) VisitTransactionLevelBase(ctx *mysqlparser.TransactionLevelBaseContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#privilegesBase.
func (v *visitor) VisitPrivilegesBase(ctx *mysqlparser.PrivilegesBaseContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#intervalTypeBase.
func (v *visitor) VisitIntervalTypeBase(ctx *mysqlparser.IntervalTypeBaseContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#dataTypeBase.
func (v *visitor) VisitDataTypeBase(ctx *mysqlparser.DataTypeBaseContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#keywordsCanBeId.
func (v *visitor) VisitKeywordsCanBeId(ctx *mysqlparser.KeywordsCanBeIdContext) any {
	return v.VisitChildren(ctx)
}

// Visit a parse tree produced by MySqlParser#functionNameBase.
func (v *visitor) VisitFunctionNameBase(ctx *mysqlparser.FunctionNameBaseContext) any {
	return v.VisitChildren(ctx)
}
