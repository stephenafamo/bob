package parser

import (
	"fmt"
	"io"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"

	pg "github.com/pganalyze/pg_query_go/v6"
	"github.com/stephenafamo/bob/internal"
)

const (
	openParToken  = pg.Token_ASCII_40
	closeParToken = pg.Token_ASCII_41
)

func newNodeInfo() nodeInfo {
	return nodeInfo{
		start:    -1,
		end:      -1,
		children: make(map[string]nodeInfo),
	}
}

type nodeInfo struct {
	start, end int32
	children   map[string]nodeInfo
}

func (n nodeInfo) isValid() bool {
	return n.start >= 0 && n.end >= 0
}

func (info nodeInfo) addChild(name string, childInfo nodeInfo) nodeInfo {
	if !childInfo.isValid() {
		return info
	}
	if childInfo.start != -1 && (info.start == -1 || childInfo.start < info.start) {
		info.start = childInfo.start
	}
	if childInfo.end != -1 && (info.end == -1 || childInfo.end > info.end) {
		info.end = childInfo.end
	}
	info.children[name] = childInfo

	return info
}

func (n nodeInfo) String() string {
	return fmt.Sprintf("%d:%d -> (%d)", n.start, n.end, len(n.children))
}

func (n nodeInfo) position() position {
	return position{n.start, n.end}
}

type col struct {
	pos      position
	name     string
	nullable bool
}

func (c col) LitterDump(w io.Writer) {
	fmt.Fprintf(w, "(%s, %s, %t)", c.name, c.pos.String(), c.nullable)
}

type queryResult struct {
	schema          string
	name            string
	columns         []col
	mustBeQualified bool
}

type walker struct {
	db           tables
	sharedSchema string
	names        map[position]string
	nullability  map[position]nullable
	groups       map[argPos]struct{}
	multiple     map[[2]int]struct{}

	position int32
	input    string
	tokens   []*pg.ScanToken

	editRules []internal.EditRule
	imports   [][]string
	mods      *strings.Builder

	atom *atomic.Int64
	args [][]argPos

	errors []error
}

func (w *walker) matchNames(p1 [2]int32, p2 [2]int32) {
	w.maybeSetName(p1, w.names[p2])
	w.maybeSetName(p2, w.names[p1])
}

func (w *walker) maybeSetName(p [2]int32, name string) {
	_, ok := w.names[p]
	if ok {
		return
	}

	if len(name) >= 2 {
		switch {
		case name[0] == '"' && name[len(name)-1] == '"':
			name = name[1 : len(name)-1]
		case name[0] == '\'' && name[len(name)-1] == '\'':
			name = name[1 : len(name)-1]
		}
	}

	if name == "" {
		return
	}

	w.names[p] = name
}

func (w *walker) setNull(p [2]int32, n nullable) {
	if n == nil {
		return
	}

	if prev, ok := w.nullability[p]; ok {
		w.nullability[p] = makeAnyNullable(n, prev)
		return
	}

	w.nullability[p] = n
}

func (w *walker) setGroup(pos argPos) {
	w.groups[pos] = struct{}{}
}

func (w *walker) setMultiple(pos [2]int) {
	w.multiple[pos] = struct{}{}
}

func (w *walker) updatePosition(pos int32) {
	if pos <= 0 {
		return
	}
	w.position = pos
}

func (w *walker) walk(a any) nodeInfo {
	if a == nil {
		return newNodeInfo()
	}

	info := newNodeInfo()

	switch a := a.(type) {
	case *pg.Node:
		if a != nil {
			info = w.reflectWalk(reflect.ValueOf(a.Node))
		}

	case *pg.NullTest:
		info = w.walkNullTest(a)

	case *pg.A_Const:
		info = w.walkAConst(a)

	case *pg.A_Star:
		info = w.findTokenAfter(w.position, pg.Token_ASCII_42)
		if info.isValid() {
			w.maybeSetName(info.position(), w.input[info.start:info.end])
		}

	case *pg.CoalesceExpr:
		info = w.reflectWalk(reflect.ValueOf(a))
		for _, argInfo := range info.children["Args"].children {
			w.setNull(argInfo.position(), alwaysNullable{})
		}

	case *pg.SelectStmt:
		info = w.walkSelectStmt(a)

	case *pg.InsertStmt:
		info = w.walkInsertStmt(a)

	case *pg.UpdateStmt:
		info = w.walkUpdateStmt(a)

	case *pg.DeleteStmt:
		info = w.walkDeleteStmt(a)

	case *pg.ParamRef:
		info = w.walkParamRef(a)

	case *pg.A_Expr:
		info = w.walkAExpr(a)

	case *pg.ColumnRef:
		info = w.walkColumnRef(a)

	case *pg.ResTarget:
		info = w.walkResTarget(a)

	case *pg.RangeVar:
		info = w.walkRangeVar(a)

	case *pg.Alias:
		info = w.walkAlias(a)

	case *pg.String:
		info = w.walkString(a)

	case *pg.SortBy:
		info = w.walkSortBy(a)

	case *pg.FuncCall:
		info = w.walkFuncCall(a)

	case *pg.List:
		info = w.walkList(a)

	case *pg.RowExpr:
		info = w.walkRowExpr(a)

	case *pg.A_ArrayExpr:
		info = w.walkAArrayExpr(a)

	case *pg.OnConflictClause:
		info = w.walkOnConflictClause(a)

	case reflect.Value:
		info = w.reflectWalk(a)

	default:
		info = w.reflectWalk(reflect.ValueOf(a))
	}

	w.updatePosition(info.end)

	return info
}

func (w *walker) reflectWalk(reflected reflect.Value) nodeInfo {
	if !reflected.IsValid() {
		return newNodeInfo()
	}

	if reflected.Kind() == reflect.Slice {
		info := newNodeInfo()
		for i := range reflected.Len() {
			childInfo := w.walk(reflected.Index(i).Interface())
			info = info.addChild(strconv.Itoa(i), childInfo)
		}
		return info
	}

	refStruct := reflected

	if reflected.Kind() == reflect.Ptr {
		if reflected.IsNil() {
			return newNodeInfo()
		}
		refStruct = reflected.Elem()
	}

	if refStruct.Kind() != reflect.Struct {
		return newNodeInfo()
	}

	info := newNodeInfo()

	LocationField := refStruct.FieldByName("Location")
	if LocationField.IsValid() && LocationField.Kind() == reflect.Int32 {
		info.start = int32(LocationField.Int())
		w.updatePosition(info.start)
		info.end = w.getEnd(info.start)
	}

	for i := range refStruct.NumField() {
		fieldType := refStruct.Type().Field(i)
		if !fieldType.IsExported() {
			continue
		}

		if fieldType.Name == "Location" {
			continue
		}

		field := refStruct.Field(i)

		childInfo := w.walk(field.Interface())
		if childInfo.start == -1 || childInfo.end == -1 {
			continue
		}
		info = info.addChild(fieldType.Name, childInfo)
	}

	return w.balanceParenthesis(info)
}

func (w *walker) walkNullTest(a *pg.NullTest) nodeInfo {
	info := w.reflectWalk(reflect.ValueOf(a))
	nullInfo := w.findTokenAfter(info.end, pg.Token_NULL_P)
	if nullInfo.end != -1 {
		info.end = nullInfo.end
	}
	w.setNull(info.children["Arg"].position(), alwaysNullable{})

	return info
}

func (w *walker) walkAConst(a *pg.A_Const) nodeInfo {
	w.updatePosition(a.Location)
	info := nodeInfo{
		start:    a.Location,
		end:      w.getEnd(a.Location),
		children: map[string]nodeInfo{},
	}
	w.maybeSetName(info.position(), w.input[info.start:info.end])
	return info
}

func (w *walker) walkSelectStmt(a *pg.SelectStmt) nodeInfo {
	if a == nil {
		return newNodeInfo()
	}

	info := w.reflectWalk(reflect.ValueOf(a))
	info.start = w.getStartOfTokenBefore(info.start, pg.Token_SELECT, pg.Token_VALUES)

	if err := verifySelectStatement(a, info); err != nil {
		w.errors = append(w.errors, err)
	}

	valsInfo := info.children["ValuesLists"]
	for i := range a.ValuesLists {
		valInfo := valsInfo.children[strconv.Itoa(i)]
		w.editRules = append(w.editRules, internal.RecordPoints(
			int(valInfo.start), int(valInfo.end-1),
			func(start, end int) error {
				w.setGroup(argPos{
					original: valInfo.position(),
					edited:   [2]int{start, end},
				})
				if len(a.ValuesLists) == 1 {
					w.setMultiple([2]int{start, end})
				}
				return nil
			},
		)...)
	}

	return info
}

func (w *walker) walkInsertStmt(a *pg.InsertStmt) nodeInfo {
	info := w.reflectWalk(reflect.ValueOf(a))
	info.start = w.getStartOfTokenBefore(info.start, pg.Token_INSERT)

	vals := a.GetSelectStmt().GetSelectStmt().GetValuesLists()
	if len(vals) == 0 {
		return info
	}

	colNames := make([]string, len(a.Cols))
	for i := range a.Cols {
		colNameInfo := info.children["Cols"].children[strconv.Itoa(i)]
		colNames[i] = w.names[colNameInfo.position()]
	}

	table := w.getTableSource(a.Relation, info.children["Relation"])
	if len(a.Cols) == 0 {
		colNames = make([]string, len(table.columns))
		for i := range table.columns {
			colNames[i] = table.columns[i].name
		}
	}

	valsInfo := info.
		children["SelectStmt"].
		children["SelectStmt"].
		children["ValuesLists"]

	for i := range vals {
		itemsInfo := valsInfo.
			children[strconv.Itoa(i)].
			children["List"].
			children["Items"]

		for colIndex := range colNames {
			itemInfo, hasInfo := itemsInfo.children[strconv.Itoa(colIndex)]
			if !hasInfo {
				continue
			}
			name := colNames[colIndex]
			w.maybeSetName(itemInfo.position(), name)
			for _, col := range table.columns {
				if col.name == name && col.nullable {
					w.setNull(itemInfo.position(), alwaysNullable{})
					break
				}
			}
		}
	}

	return info
}

func (w *walker) walkUpdateStmt(a *pg.UpdateStmt) nodeInfo {
	info := w.reflectWalk(reflect.ValueOf(a))
	info.start = w.getStartOfTokenBefore(info.start, pg.Token_UPDATE)

	if err := verifyUpdateStatement(a, info); err != nil {
		w.errors = append(w.errors, err)
	}

	return info
}

func (w *walker) walkDeleteStmt(a *pg.DeleteStmt) nodeInfo {
	info := w.reflectWalk(reflect.ValueOf(a))
	info.start = w.getStartOfTokenBefore(info.start, pg.Token_DELETE_P)

	if err := verifyDeleteStatement(a, info); err != nil {
		w.errors = append(w.errors, err)
	}

	return info
}

func (w *walker) walkParamRef(a *pg.ParamRef) nodeInfo {
	w.updatePosition(a.Location)
	info := nodeInfo{
		start:    a.Location,
		end:      w.getEnd(a.Location),
		children: map[string]nodeInfo{},
	}
	if len(w.args) < int(a.Number) {
		w.args = append(w.args, make([][]argPos, int(a.Number)-len(w.args))...)
	}
	w.editRules = append(w.editRules, internal.EditCallback(
		internal.ReplaceFromFunc(
			int(info.start), int(info.end-1),
			func() string {
				return fmt.Sprintf("$%d", w.atom.Add(1))
			},
		),
		func(start, end int, _, _ string) error {
			w.args[a.Number-1] = append(w.args[a.Number-1], argPos{
				original: info.position(),
				edited:   [2]int{start, end},
			})
			return nil
		}),
	)

	return info
}

func (w *walker) walkAExpr(a *pg.A_Expr) nodeInfo {
	info := w.reflectWalk(reflect.ValueOf(a))
	lInfo := info.children["Lexpr"]
	rInfo := info.children["Rexpr"]
	switch a.Kind {
	case pg.A_Expr_Kind_AEXPR_OP,
		pg.A_Expr_Kind_AEXPR_DISTINCT,
		pg.A_Expr_Kind_AEXPR_NOT_DISTINCT:
		w.matchNames(lInfo.position(), rInfo.position())
	case pg.A_Expr_Kind_AEXPR_OP_ANY, pg.A_Expr_Kind_AEXPR_OP_ALL:
		for _, argInfo := range rInfo.children["AArrayExpr"].children {
			w.matchNames(lInfo.position(), argInfo.position())
		}
	case pg.A_Expr_Kind_AEXPR_IN:
		lRow, isRow := lInfo.children["RowExpr"].children["Args"]
		for _, argInfo := range rInfo.children["List"].children["Items"].children {
			w.matchNames(lInfo.position(), argInfo.position())
			if !isRow {
				continue
			}
			for key, rowItem := range argInfo.children["RowExpr"].children["Args"].children {
				w.matchNames(lRow.children[key].position(), rowItem.position())
			}
		}
	}

	return info
}

func (w *walker) walkColumnRef(a *pg.ColumnRef) nodeInfo {
	info := w.reflectWalk(reflect.ValueOf(a))
	if len(a.Fields) > 0 {
		lastInfo := info.children["Fields"].children[strconv.Itoa(len(a.Fields)-1)]
		if lastInfo.isValid() {
			w.maybeSetName(info.position(), w.names[lastInfo.position()])
		}
	}
	w.setNull(info.position(), columnNullable{a, info})

	return info
}

func (w *walker) walkResTarget(a *pg.ResTarget) nodeInfo {
	w.updatePosition(a.Location)
	info := w.reflectWalk(reflect.ValueOf(a))

	if a.Name != "" {
		nameInfo := newNodeInfo()
		index := sort.Search(len(w.tokens)-1, func(i int) bool {
			return w.tokens[i].End > info.end
		})

	IndexLoop:
		for i := index; i < len(w.tokens); i++ {
			switch {
			case w.tokens[i].Token == pg.Token_IDENT ||
				w.tokens[i].KeywordKind == pg.KeywordKind_UNRESERVED_KEYWORD:
				nameInfo = nodeInfo{
					start: w.tokens[i].Start,
					end:   w.tokens[i].End,
				}

			case w.tokens[i].Token != pg.Token_AS:
				break IndexLoop
			}
		}

		info = info.addChild("Name", nameInfo)
		w.maybeSetName(info.position(), a.Name)
	}

	valPos := info.children["Val"].position()
	w.maybeSetName(info.position(), w.names[valPos])
	w.setNull(info.position(), w.nullability[valPos])

	if a.Name != "" {
		w.maybeSetName(valPos, a.Name)
	}

	return info
}

func (w *walker) walkRangeVar(a *pg.RangeVar) nodeInfo {
	w.updatePosition(a.Location)
	info := newNodeInfo()
	firstInfo := nodeInfo{
		start:    a.Location,
		end:      w.getEnd(a.Location),
		children: map[string]nodeInfo{},
	}
	switch {
	case a.Catalogname != "":
		catalogInfo := firstInfo
		schemaInfo := w.findIdentOrUnreserved(catalogInfo.end)
		relInfo := w.findIdentOrUnreserved(schemaInfo.end)

		w.maybeSetName(catalogInfo.position(), a.Catalogname)
		w.maybeSetName(schemaInfo.position(), a.Schemaname)
		w.maybeSetName(relInfo.position(), a.Relname)

		info = info.addChild("Catalogname", catalogInfo)
		info = info.addChild("Schemaname", schemaInfo)
		info = info.addChild("Relname", relInfo)

	case a.Schemaname != "":
		schemaInfo := firstInfo
		relInfo := w.findIdentOrUnreserved(schemaInfo.end)

		w.maybeSetName(schemaInfo.position(), a.Schemaname)
		w.maybeSetName(relInfo.position(), a.Relname)

		info = info.addChild("Schemaname", schemaInfo)
		info = info.addChild("Relname", relInfo)
	default:
		w.maybeSetName(firstInfo.position(), a.Relname)
		info = info.addChild("Relname", firstInfo)
	}

	w.position = info.end
	info = info.addChild("Alias", w.walk(a.Alias))

	return info
}

func (w *walker) walkAlias(a *pg.Alias) nodeInfo {
	if a == nil {
		return newNodeInfo()
	}
	aliasNameInfo := w.findIdentOrUnreserved(w.position)
	info := newNodeInfo().addChild("Aliasname", aliasNameInfo)
	w.maybeSetName(aliasNameInfo.position(), a.Aliasname)

	// Update position so that the col name identifiers can be found
	w.updatePosition(aliasNameInfo.end)
	info = info.addChild("Colnames", w.walk(a.GetColnames()))
	info = w.balanceParenthesis(info)

	return info
}

func (w *walker) walkString(a *pg.String) nodeInfo {
	info := newNodeInfo()
	identifierInfo := w.findIdentOrUnreserved(w.position)

	if !identifierInfo.isValid() {
		return info
	}

	quoted := w.input[identifierInfo.start:identifierInfo.end]
	unquoted, err := strconv.Unquote(quoted)
	if err != nil {
		unquoted = quoted
	}
	if strings.EqualFold(a.GetSval(), unquoted) {
		info = identifierInfo
		w.maybeSetName(info.position(), unquoted)
	}

	return info
}

func (w *walker) walkSortBy(a *pg.SortBy) nodeInfo {
	w.updatePosition(a.Location)
	info := w.reflectWalk(reflect.ValueOf(a))
	hasSortDir := a.SortbyDir > pg.SortByDir_SORTBY_DEFAULT
	hasSortNulls := a.SortbyNulls > pg.SortByNulls_SORTBY_NULLS_DEFAULT
	switch {
	case hasSortNulls:
		info.end = w.getEndOfTokenAfter(
			info.start, pg.Token_FIRST_P, pg.Token_LAST_P)
	case hasSortDir && a.SortbyDir != pg.SortByDir_SORTBY_USING:
		info.end = w.getEndOfTokenAfter(
			info.start, pg.Token_ASC, pg.Token_DESC)
	}
	return info
}

func (w *walker) walkFuncCall(a *pg.FuncCall) nodeInfo {
	info := w.reflectWalk(reflect.ValueOf(a))
	if len(a.Funcname) > 0 {
		funcNameInfo := info.children["Funcname"].children["0"]
		if funcNameInfo.isValid() {
			w.maybeSetName(info.position(), w.names[funcNameInfo.position()])
		}
	}

	return info
}

func (w *walker) walkList(a *pg.List) nodeInfo {
	info := w.reflectWalk(reflect.ValueOf(a))
	info.start = w.getStartOfTokenBefore(info.start, openParToken)
	info.end = w.getEndOfTokenAfter(info.end, closeParToken)

	w.editRules = append(w.editRules, internal.RecordPoints(
		int(info.start), int(info.end-1),
		func(start, end int) error {
			w.setGroup(argPos{
				original: info.position(),
				edited:   [2]int{start, end},
			})
			return nil
		},
	)...)

	itemsInfo := info.children["Items"]
	if len(a.Items) == 1 {
		w.editRules = append(w.editRules, internal.RecordPoints(
			int(itemsInfo.start), int(itemsInfo.end-1),
			func(start, end int) error {
				w.setMultiple([2]int{start, end})
				return nil
			},
		)...)
	}

	return info
}

func (w *walker) walkRowExpr(a *pg.RowExpr) nodeInfo {
	info := w.reflectWalk(reflect.ValueOf(a))
	w.editRules = append(w.editRules,
		internal.RecordPoints(int(info.start), int(info.end-1),
			func(start, end int) error {
				w.setGroup(argPos{
					original: info.position(),
					edited:   [2]int{start, end},
				})
				return nil
			},
		)...)

	return info
}

func (w *walker) walkAArrayExpr(a *pg.A_ArrayExpr) nodeInfo {
	info := w.reflectWalk(reflect.ValueOf(a))
	info.end = w.getEndOfTokenAfter(info.end, pg.Token_ASCII_93)

	elementsInfo := info.children["Elements"]
	if len(a.Elements) == 1 {
		w.editRules = append(w.editRules, internal.RecordPoints(
			int(elementsInfo.start), int(elementsInfo.end-1),
			func(start, end int) error {
				w.setMultiple([2]int{start, end})
				w.setGroup(argPos{
					original: elementsInfo.position(),
					edited:   [2]int{start, end},
				})
				return nil
			},
		)...)
	} else {
		w.editRules = append(w.editRules, internal.RecordPoints(
			int(info.start), int(info.end-1),
			func(start, end int) error {
				w.setGroup(argPos{
					original: info.position(),
					edited:   [2]int{start, end},
				})
				return nil
			},
		)...)
	}

	return info
}

func (w *walker) walkOnConflictClause(a *pg.OnConflictClause) nodeInfo {
	info := w.reflectWalk(reflect.ValueOf(a))

	var doIndex, actionIndex int
	lastInfo := w.findTokenAfterFunc(info.start, func(index int, t *pg.ScanToken) bool {
		switch t.Token {
		case pg.Token_DO:
			if t.KeywordKind == pg.KeywordKind_RESERVED_KEYWORD {
				doIndex = index
			}
		case pg.Token_UPDATE, pg.Token_NOTHING:
			if t.KeywordKind == pg.KeywordKind_UNRESERVED_KEYWORD {
				actionIndex = index
			}
		}

		return actionIndex-doIndex == 1
	})

	if lastInfo.isValid() {
		info = info.addChild("Action", nodeInfo{w.tokens[doIndex].Start, w.tokens[actionIndex].End, nil})
	}

	return info
}
