package driver

import (
	"context"
	"database/sql"
	sqlDriver "database/sql/driver"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"

	helpers "github.com/stephenafamo/bob/gen/bobgen-helpers"
	"github.com/stephenafamo/bob/gen/bobgen-sqlite/driver/parser"
	"github.com/stephenafamo/bob/gen/drivers"
	"github.com/stephenafamo/scan"
	"github.com/stephenafamo/scan/stdscan"
	"github.com/volatiletech/strmangle"
	"modernc.org/sqlite"
)

const (
	mattnDriver    = "github.com/mattn/go-sqlite3"
	morderncDriver = "modernc.org/sqlite"
	ncrucesDriver  = "github.com/ncruces/go-sqlite3"
	libsqlDriver   = "github.com/tursodatabase/libsql-client-go/libsql"
	defaultDriver  = morderncDriver
)

type (
	Interface  = drivers.Interface[any, any, IndexExtra]
	DBInfo     = drivers.DBInfo[any, any, IndexExtra]
	IndexExtra = parser.IndexExtra
)

func init() {
	if err := registerRegexpFunction(); err != nil {
		panic(fmt.Sprintf("failed to register regexp function: %v", err))
	}
}

func New(config Config) Interface {
	if config.Driver == "" {
		config.Driver = defaultDriver
	}

	switch config.Driver {
	// These are the only supported drivers
	case mattnDriver, morderncDriver, ncrucesDriver, libsqlDriver:
	default:
		panic(fmt.Sprintf(
			"unsupported driver %q, supported drivers are: %q, %q, %q, %q",
			config.Driver,
			mattnDriver, morderncDriver,
			ncrucesDriver, libsqlDriver,
		))
	}
	return &driver{config: config}
}

type Config struct {
	helpers.Config `yaml:",squash"`
	// The database schemas to generate models for
	// a map of the schema name to the DSN
	Attach map[string]string
	// The name of this schema will not be included in the generated models
	// a context value can then be used to set the schema at runtime
	// useful for multi-tenant setups
	SharedSchema string `yaml:"shared_schema"`
}

func (c Config) AttachQueries() []string {
	driverName := inferDriver(c)
	queries := make([]string, 0, len(c.Attach))
	for schema, dsn := range c.Attach {
		if driverName == "sqlite" {
			dsn = strconv.Quote(dsn)
		}
		queries = append(queries, fmt.Sprintf("attach database %s as %s", dsn, schema))
	}

	return queries
}

// driver holds the database connection string and a handle
// to the database connection.
type driver struct {
	config Config
	conn   *sql.DB
}

func (d *driver) Dialect() string {
	return "sqlite"
}

func (d *driver) Destination() string {
	return d.config.Output
}

func (d *driver) PackageName() string {
	return d.config.Pkgname
}

func (d *driver) Types() drivers.Types {
	return helpers.Types()
}

func attach(ctx context.Context, db *sql.DB, config Config) error {
	for _, query := range config.AttachQueries() {
		_, err := db.ExecContext(ctx, query)
		if err != nil {
			return fmt.Errorf("error running query %s: %w", query, err)
		}
	}

	return nil
}

// Assemble all the information we need to provide back to the driver
func (d *driver) Assemble(ctx context.Context) (*DBInfo, error) {
	var err error

	if d.config.SharedSchema == "" {
		d.config.SharedSchema = "main"
	}

	if d.config.Dsn == "" {
		return nil, fmt.Errorf("database dsn is not set")
	}

	driverName := inferDriver(d.config)
	d.conn, err = sql.Open(driverName, d.config.Dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	defer d.conn.Close()

	for schema, dsn := range d.config.Attach {
		if driverName == "sqlite" {
			dsn = strconv.Quote(dsn)
		}
		_, err = d.conn.ExecContext(ctx, fmt.Sprintf("attach database %s as %s", dsn, schema))
		if err != nil {
			return nil, fmt.Errorf("could not attach %q: %w", schema, err)
		}
	}

	tables, err := d.tables(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting tables: %w", err)
	}

	queries, err := parser.New(tables, driverName).ParseFolders(ctx, d.config.Queries...)
	if err != nil {
		return nil, fmt.Errorf("parse query folders: %w", err)
	}

	if driverName == "libsql" {
		d.config.Driver = "github.com/tursodatabase/libsql-client-go/libsql"
	}
	dbinfo := &DBInfo{
		Driver:       d.config.Driver,
		Tables:       tables,
		QueryFolders: queries,
	}

	return dbinfo, nil
}

func inferDriver(config Config) string {
	driverName := "sqlite"
	if !strings.Contains(config.Dsn, "://") {
		return driverName
	}
	dsn, _ := url.Parse(config.Dsn)
	if dsn == nil {
		return driverName
	}
	libsqlSchemes := map[string]bool{
		"libsql": true,
		"file":   true,
		"https":  true,
		"http":   true,
		"wss":    true,
		"ws":     true,
	}
	if libsqlSchemes[dsn.Scheme] {
		driverName = "libsql"
	}
	return driverName
}

func (d *driver) buildQuery(schema string) (string, []any) {
	var args []any
	query := fmt.Sprintf(`SELECT name FROM %q.sqlite_schema WHERE name NOT LIKE 'sqlite_%%' AND type IN ('table', 'view')`, schema)

	tableFilter := drivers.ParseTableFilter(d.config.Only, d.config.Except)

	if len(tableFilter.Only) > 0 {
		var subqueries []string
		stringPatterns, regexPatterns := tableFilter.ClassifyPatterns(tableFilter.Only)
		include := make([]string, 0, len(stringPatterns))
		for _, name := range stringPatterns {
			if (schema == "main" && !strings.Contains(name, ".")) || strings.HasPrefix(name, schema+".") {
				include = append(include, strings.TrimPrefix(name, schema+"."))
			}
		}
		if len(include) > 0 {
			subqueries = append(subqueries, fmt.Sprintf("name in (%s)", strmangle.Placeholders(false, len(include), 1, 1)))
			for _, w := range include {
				args = append(args, w)
			}
		}
		if len(regexPatterns) > 0 {
			subqueries = append(subqueries, fmt.Sprintf("name regexp (%s)", strmangle.Placeholders(false, 1, len(args)+1, 1)))
			args = append(args, strings.Join(regexPatterns, "|"))
		}
		if len(subqueries) > 0 {
			query += fmt.Sprintf(" and (%s)", strings.Join(subqueries, " or "))
		}
	}

	if len(tableFilter.Except) > 0 {
		var subqueries []string
		stringPatterns, regexPatterns := tableFilter.ClassifyPatterns(tableFilter.Except)
		exclude := make([]string, 0, len(tableFilter.Except))
		for _, name := range stringPatterns {
			if (schema == "main" && !strings.Contains(name, ".")) || strings.HasPrefix(name, schema+".") {
				exclude = append(exclude, strings.TrimPrefix(name, schema+"."))
			}
		}
		if len(exclude) > 0 {
			subqueries = append(subqueries, fmt.Sprintf("name not in (%s)", strmangle.Placeholders(false, len(exclude), 1+len(args), 1)))
			for _, w := range exclude {
				args = append(args, w)
			}
		}
		if len(regexPatterns) > 0 {
			subqueries = append(subqueries, fmt.Sprintf("name not regexp (%s)", strmangle.Placeholders(false, 1, len(args)+1, 1)))
			args = append(args, strings.Join(regexPatterns, "|"))
		}
		if len(subqueries) > 0 {
			query += fmt.Sprintf(" and (%s)", strings.Join(subqueries, " and "))
		}
	}

	query += ` ORDER BY type, name`

	return query, args
}

func (d *driver) tables(ctx context.Context) (drivers.Tables[any, IndexExtra], error) {
	mainQuery, mainArgs := d.buildQuery("main")
	mainTables, err := stdscan.All(ctx, d.conn, scan.SingleColumnMapper[string], mainQuery, mainArgs...)
	if err != nil {
		return nil, err
	}

	colFilter := drivers.ParseColumnFilter(mainTables, d.config.Only, d.config.Except)
	allTables := make(drivers.Tables[any, IndexExtra], len(mainTables))
	for i, name := range mainTables {
		allTables[i], err = d.getTable(ctx, "main", name, colFilter)
		if err != nil {
			return nil, err
		}
	}

	for schema := range d.config.Attach {
		schemaQuery, schemaArgs := d.buildQuery(schema)
		tables, err := stdscan.All(ctx, d.conn, scan.SingleColumnMapper[string], schemaQuery, schemaArgs...)
		if err != nil {
			return nil, err
		}
		colFilter = drivers.ParseColumnFilter(tables, d.config.Only, d.config.Except)
		for _, name := range tables {
			table, err := d.getTable(ctx, schema, name, colFilter)
			if err != nil {
				return nil, err
			}
			allTables = append(allTables, table)
		}
	}

	return allTables, nil
}

func (d driver) getTable(ctx context.Context, schema, name string, colFilter drivers.ColumnFilter) (drivers.Table[any, IndexExtra], error) {
	var err error

	table := drivers.Table[any, IndexExtra]{
		Key:    d.key(schema, name),
		Schema: d.schema(schema),
		Name:   name,
	}

	tinfo, err := d.tableInfo(ctx, schema, name)
	if err != nil {
		return table, err
	}

	table.Columns, err = d.columns(ctx, schema, name, tinfo, colFilter)
	if err != nil {
		return table, err
	}

	// We cannot rely on the indexes to get the primary key
	// because it is not always included in the indexes
	table.Constraints.Primary = d.primaryKey(schema, name, tinfo)
	table.Constraints.Foreign, err = d.foreignKeys(ctx, schema, name)
	if err != nil {
		return table, err
	}

	table.Indexes, err = d.indexes(ctx, schema, name)
	if err != nil {
		return table, err
	}

	// Get Unique constraints from indexes
	// Also check if the primary key is in the indexes
	hasPk := false
	for _, index := range table.Indexes {
		switch index.Type {
		case "pk":
			hasPk = true
		case "u":
			if !index.HasExpressionColumn() {
				table.Constraints.Uniques = append(
					table.Constraints.Uniques,
					drivers.Constraint[any]{
						Name:    index.Name,
						Columns: index.NonExpressionColumns(),
					},
				)
			}
		}
	}

	// Add the primary key to the indexes if it is not already there
	if !hasPk && table.Constraints.Primary != nil {
		pkIndex := drivers.Index[IndexExtra]{
			Type:    "pk",
			Name:    table.Constraints.Primary.Name,
			Columns: make([]drivers.IndexColumn, len(table.Constraints.Primary.Columns)),
			Unique:  true,
		}

		for i, col := range table.Constraints.Primary.Columns {
			pkIndex.Columns[i] = drivers.IndexColumn{
				Name: col,
			}
		}

		// List the primary key first
		table.Indexes = append([]drivers.Index[IndexExtra]{pkIndex}, table.Indexes...)
	}

	return table, nil
}

// Columns takes a table name and attempts to retrieve the table information
// from the database. It retrieves the column names
// and column types and returns those as a []Column after TranslateColumnType()
// converts the SQL types to Go types, for example: "varchar" to "string"
func (d driver) columns(ctx context.Context, schema, tableName string, tinfo []info, colFilter drivers.ColumnFilter) ([]drivers.Column, error) {
	var columns []drivers.Column //nolint:prealloc

	//nolint:gosec
	query := fmt.Sprintf("SELECT 1 FROM '%s'.sqlite_master WHERE type = 'table' AND name = ? AND sql LIKE '%%AUTOINCREMENT%%'", schema)
	result, err := d.conn.QueryContext(ctx, query, tableName)
	if err != nil {
		return nil, fmt.Errorf("autoincr query: %w", err)
	}
	tableHasAutoIncr := result.Next()
	if err := result.Close(); err != nil {
		return nil, err
	}

	nPkeys := 0
	for _, column := range tinfo {
		if column.Pk != 0 {
			nPkeys++
		}
	}

	filter := colFilter[tableName]
	excludedColumns := make(map[string]struct{}, len(filter.Except))
	if len(filter.Except) > 0 {
		for _, w := range filter.Except {
			excludedColumns[w] = struct{}{}
		}
	}

	for _, colInfo := range tinfo {
		if _, ok := excludedColumns[colInfo.Name]; ok {
			continue
		}
		column := drivers.Column{
			Name:     colInfo.Name,
			DBType:   strings.ToUpper(colInfo.Type),
			Nullable: !colInfo.NotNull && colInfo.Pk < 1,
		}

		isPrimaryKeyInteger := colInfo.Pk == 1 && column.DBType == "INTEGER"
		// This is special behavior noted in the sqlite documentation.
		// An integer primary key becomes synonymous with the internal ROWID
		// and acts as an auto incrementing value. Although there's important
		// differences between using the keyword AUTOINCREMENT and this inferred
		// version, they don't matter here so just masquerade as the same thing as
		// above.
		autoIncr := isPrimaryKeyInteger && (tableHasAutoIncr || nPkeys == 1)

		// See: https://github.com/sqlite/sqlite/blob/91f621531dc1cb9ba5f6a47eb51b1de9ed8bdd07/src/pragma.c#L1165
		column.Generated = colInfo.Hidden == 2 || colInfo.Hidden == 3

		if colInfo.DefaultValue.Valid {
			column.Default = colInfo.DefaultValue.String
		} else if autoIncr {
			column.Default = "auto_increment"
		} else if column.Generated {
			column.Default = "auto_generated"
		}

		if column.Nullable && column.Default == "" {
			column.Default = "NULL"
		}

		column.Type = parser.TranslateColumnType(column.DBType, inferDriver(d.config))
		columns = append(columns, column)
	}

	return columns, nil
}

func (s driver) tableInfo(ctx context.Context, schema, tableName string) ([]info, error) {
	var ret []info
	rows, err := s.conn.QueryContext(ctx, fmt.Sprintf("PRAGMA '%s'.table_xinfo('%s')", schema, tableName))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		tinfo := info{}
		if err := rows.Scan(&tinfo.Cid, &tinfo.Name, &tinfo.Type, &tinfo.NotNull, &tinfo.DefaultValue, &tinfo.Pk, &tinfo.Hidden); err != nil {
			return nil, fmt.Errorf("unable to scan for table %s: %w", tableName, err)
		}

		ret = append(ret, tinfo)
	}
	return ret, nil
}

// primaryKey looks up the primary key for a table.
func (s driver) primaryKey(schema, tableName string, tinfo []info) *drivers.Constraint[any] {
	var cols []string
	for _, c := range tinfo {
		if c.Pk > 0 {
			cols = append(cols, c.Name)
		}
	}

	if len(cols) == 0 {
		return nil
	}

	return &drivers.Constraint[any]{
		Name:    fmt.Sprintf("pk_%s_%s", schema, tableName),
		Columns: cols,
	}
}

func (d driver) skipKey(table, column string) bool {
	if len(d.config.Only) > 0 {
		// check if the table is listed at all
		filter, ok := d.config.Only[table]
		if !ok {
			return true
		}

		// check if the column is listed
		if len(filter) == 0 {
			return false
		}

		return !slices.Contains(filter, column)
	}

	if len(d.config.Except) > 0 {
		filter, ok := d.config.Except[table]
		if !ok {
			return false
		}

		if len(filter) == 0 {
			return true
		}

		if slices.Contains(filter, column) {
			return true
		}
	}

	return false
}

// foreignKeys retrieves the foreign keys for a given table name.
func (d driver) foreignKeys(ctx context.Context, schema, tableName string) ([]drivers.ForeignKey[any], error) {
	rows, err := d.conn.QueryContext(ctx, fmt.Sprintf("PRAGMA '%s'.foreign_key_list('%s')", schema, tableName))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	fkeyMap := make(map[int]drivers.ForeignKey[any])
	for rows.Next() {
		var id, seq int
		var ftable, col string
		var fcolNullable sql.Null[string]

		// not used
		var onupdate, ondelete, match string

		err = rows.Scan(&id, &seq, &ftable, &col, &fcolNullable, &onupdate, &ondelete, &match)
		if err != nil {
			return nil, err
		}

		fullFtable := ftable
		if schema != "main" {
			fullFtable = fmt.Sprintf("%s.%s", schema, ftable)
		}

		fcol := fcolNullable.V
		if fcol == "" {
			fcol, err = stdscan.One(
				ctx, d.conn, scan.SingleColumnMapper[string],
				fmt.Sprintf("SELECT name FROM pragma_table_info('%s', '%s') WHERE pk = ?", ftable, schema), seq+1,
			)
			if err != nil {
				return nil, fmt.Errorf("could not find column %q in table %q: %w", col, ftable, err)
			}
		}

		if d.skipKey(fullFtable, fcol) {
			continue
		}

		fkeyMap[id] = drivers.ForeignKey[any]{
			Constraint: drivers.Constraint[any]{
				Name:    fmt.Sprintf("fk_%s_%d", tableName, id),
				Columns: append(fkeyMap[id].Columns, col),
			},
			ForeignTable:   d.key(schema, ftable),
			ForeignColumns: append(fkeyMap[id].ForeignColumns, fcol),
		}
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	fkeys := make([]drivers.ForeignKey[any], 0, len(fkeyMap))

	for _, fkey := range fkeyMap {
		fkeys = append(fkeys, fkey)
	}

	sort.Slice(fkeys, func(i, j int) bool {
		return fkeys[i].Name < fkeys[j].Name
	})

	return fkeys, nil
}

// uniques retrieves the unique keys for a given table name.

type info struct {
	Cid          int
	Name         string
	Type         string
	NotNull      bool
	DefaultValue sql.NullString
	Pk           int
	Hidden       int
}

func (d *driver) key(schema string, table string) string {
	key := table
	if schema != "" && schema != d.config.SharedSchema {
		key = schema + "." + table
	}

	return key
}

func (d *driver) schema(schema string) string {
	if schema == d.config.SharedSchema {
		return ""
	}

	return schema
}

func (d *driver) indexes(ctx context.Context, schema, tableName string) ([]drivers.Index[IndexExtra], error) {
	query := fmt.Sprintf(`
        SELECT name, "unique", origin, partial
        FROM pragma_index_list('%s', '%s') ORDER BY seq ASC
        `, tableName, schema)
	indexNames, err := stdscan.All(ctx, d.conn, scan.StructMapper[struct {
		Name    string
		Unique  bool
		Origin  string
		Partial bool
	}](), query)
	if err != nil {
		return nil, err
	}

	indexes := make([]drivers.Index[IndexExtra], len(indexNames))
	for i, index := range indexNames {
		cols, err := d.getIndexInformation(ctx, schema, tableName, index.Name)
		if err != nil {
			return nil, err
		}
		indexes[i] = drivers.Index[IndexExtra]{
			Type:    index.Origin,
			Name:    index.Name,
			Unique:  index.Unique,
			Columns: cols,
			Extra: IndexExtra{
				Partial: index.Partial,
			},
		}

	}

	return indexes, nil
}

func (d *driver) getIndexInformation(ctx context.Context, schema, tableName, indexName string) ([]drivers.IndexColumn, error) {
	colExpressions, err := d.extractIndexExpressions(ctx, schema, tableName, indexName)
	if err != nil {
		return nil, err
	}

	query := fmt.Sprintf(`
            SELECT seqno, name, desc
            FROM pragma_index_xinfo('%s', '%s')
            WHERE key = 1
            ORDER BY seqno ASC`,
		indexName, schema)

	var columns []drivers.IndexColumn //nolint:prealloc
	for column, err := range stdscan.Each(ctx, d.conn, scan.StructMapper[struct {
		Seqno int
		Name  sql.NullString
		Desc  bool
	}](), query) {
		if err != nil {
			return nil, err
		}

		col := drivers.IndexColumn{
			Name: column.Name.String,
			Desc: column.Desc,
		}

		if !column.Name.Valid {
			col.Name = colExpressions[column.Seqno]
			col.IsExpression = true
		}

		columns = append(columns, col)
	}

	return columns, nil
}

func (d driver) extractIndexExpressions(ctx context.Context, schema, tableName, indexName string) ([]string, error) {
	var nullDDL sql.NullString

	//nolint:gosec
	query := fmt.Sprintf("SELECT sql FROM '%s'.sqlite_master WHERE type = 'index' AND name = ? AND tbl_name = ?", schema)
	result := d.conn.QueryRowContext(ctx, query, indexName, tableName)
	err := result.Scan(&nullDDL)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("failed retrieving index DDL statement: %w", err)
	}

	if !nullDDL.Valid {
		return nil, nil
	}

	ddl := nullDDL.String
	// We're following the parsing logic from the `intckParseCreateIndex` function in the SQLite source code.
	// 1. https://github.com/sqlite/sqlite/blob/1d8cde9d56d153767e98595c4b015221864ef0e7/ext/intck/sqlite3intck.c#L363
	// 2. https://www.sqlite.org/lang_createindex.html

	// skip forward until the first "(" token
	i := strings.Index(ddl, "(")
	if i == -1 {
		return nil, fmt.Errorf("failed locating first column: %w", err)
	}
	ddl = ddl[i+1:]
	// discard the WHERE clause fragment (if one exists)
	i = strings.LastIndex(ddl, ")")
	if i == -1 {
		return nil, fmt.Errorf("failed locating last column: %w", err)
	}
	ddl = ddl[:i]
	// organize column definitions into a list
	colDefs := d.splitColumnDefinitions(ddl)

	expressions := make([]string, len(colDefs))
	for seqNo, expression := range colDefs {
		expressions[seqNo] = strings.TrimSpace(expression)
	}

	return expressions, nil
}

// splitColumnDefinitions performs an intelligent split of the DDL part defining the index columns.
//
// We cannot perform a simple `strings.Split(ddl, ",")` as `ddl` could contain functional expressions, i.e.:
//
//	sql  := CREATE INDEX idx ON test (col1, (col2 + col3), (POW(col3, 2)));
//	ddl  := "col1, (col2 + col3), (POW(col3, 2))"
//	defs := []string{"col1", "(col2 + col3)", "(POW(col3, 2))"}
func (d driver) splitColumnDefinitions(ddl string) []string {
	var defs []string
	var i, pOpen int

	for j := range len(ddl) {
		if ddl[j] == '(' {
			pOpen++
		}
		if ddl[j] == ')' {
			pOpen--
		}
		if pOpen == 0 && ddl[j] == ',' {
			defs = append(defs, ddl[i:j])
			i = j + 1
		}
	}

	if i < len(ddl) {
		defs = append(defs, ddl[i:])
	}

	return defs
}

func registerRegexpFunction() error {
	return sqlite.RegisterScalarFunction("regexp", 2, func(
		ctx *sqlite.FunctionContext,
		args []sqlDriver.Value,
	) (sqlDriver.Value, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("expected 2 arguments, got %d", len(args))
		}

		re, ok := args[0].(string)
		if !ok {
			return nil, fmt.Errorf("expected string, got %T", args[0])
		}

		s, ok := args[1].(string)
		if !ok {
			return nil, fmt.Errorf("expected string, got %T", args[1])
		}

		match, err := regexp.MatchString(re, s)
		if err != nil {
			return nil, err
		}

		return match, nil
	})
}
