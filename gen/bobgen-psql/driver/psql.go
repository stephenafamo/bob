package driver

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/lib/pq"
	"github.com/stephenafamo/scan"
	"github.com/stephenafamo/scan/stdscan"
	helpers "github.com/twitter-payments/bob/gen/bobgen-helpers"
	"github.com/twitter-payments/bob/gen/drivers"
	"github.com/twitter-payments/bob/gen/importers"
	"github.com/volatiletech/strmangle"
)

var rgxValidColumnName = regexp.MustCompile(`(?i)^[a-z_][a-z0-9_]*$`)

type (
	Interface  = drivers.Interface[any, any, IndexExtra]
	DBInfo     = drivers.DBInfo[any, any, IndexExtra]
	IndexExtra = struct {
		NullsFirst    []bool   `json:"nulls_first"` // same length as Columns
		NullsDistinct bool     `json:"nulls_not_distinct"`
		Where         string   `json:"where_clause"`
		Include       []string `json:"include"`
	}
)

type Enum struct {
	Schema string
	Name   string
	Type   string
	Values pq.StringArray
}

type Config struct {
	// The database connection string
	Dsn string
	// The database schemas to generate models for
	Schemas pq.StringArray
	// The name of this schema will not be included in the generated models
	// a context value can then be used to set the schema at runtime
	// useful for multi-tenant setups
	SharedSchema string `yaml:"shared_schema"`
	// List of tables that will be included. Others are ignored
	Only map[string][]string
	// List of tables that will be should be ignored. Others are included
	Except map[string][]string
	// How many tables to fetch in parallel
	Concurrency int
	// Which UUID package to use (gofrs or google)
	UUIDPkg string `yaml:"uuid_pkg"`
	// Which `database/sql` driver to use (the full module name)
	DriverName string `yaml:"driver_name"`

	//-------

	// The name of the folder to output the models package to
	Output string
	// The name you wish to assign to your generated models package
	Pkgname   string
	NoFactory bool `yaml:"no_factory"`
}

func New(config Config) Interface {
	// Set defaults
	if config.Schemas == nil {
		config.Schemas = pq.StringArray{"public"}
	}

	if config.SharedSchema == "" {
		config.SharedSchema = config.Schemas[0]
	}

	if config.UUIDPkg == "" {
		config.UUIDPkg = "gofrs"
	}

	if config.DriverName == "" {
		config.DriverName = "github.com/lib/pq"
	}

	if config.Concurrency < 1 {
		config.Concurrency = 10
	}

	types := helpers.Types()

	switch config.UUIDPkg {
	case "google":
		types["uuid.UUID"] = drivers.Type{
			Imports:    importers.List{`"github.com/google/uuid"`},
			RandomExpr: `return uuid.New()`,
		}
	case "x":
		types["uuid.UUID"] = drivers.Type{
			Imports:    importers.List{`"github.com/twitter-payments/orchestrator/internal/uuid"`},
			RandomExpr: `return uuid.New()`,
		}
	default:
		types["uuid.UUID"] = drivers.Type{
			Imports:    importers.List{`"github.com/gofrs/uuid/v5"`},
			RandomExpr: `return uuid.Must(uuid.NewV4())`,
		}
	}
	//

	return &driver{
		config: config,
		types:  types,
	}
}

// driver holds the database connection string and a handle
// to the database connection.
type driver struct {
	config Config
	conn   *sql.DB
	enums  []Enum
	types  drivers.Types
	mu     sync.Mutex
}

func (d *driver) Dialect() string {
	return "psql"
}

func (d *driver) Types() drivers.Types {
	return d.types
}

// Assemble all the information we need to provide back to the driver
func (d *driver) Assemble(ctx context.Context) (*DBInfo, error) {
	var dbinfo *DBInfo
	var err error

	if d.config.Dsn == "" {
		return nil, fmt.Errorf("database dsn is not set")
	}

	d.conn, err = sql.Open("postgres", d.config.Dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	defer d.conn.Close()

	dbinfo = &DBInfo{DriverName: d.config.DriverName}

	// drivers.Tables call translateColumnType which uses Enums
	if err := d.loadEnums(ctx); err != nil {
		return nil, fmt.Errorf("unable to load enums: %w", err)
	}

	dbinfo.Tables, err = drivers.BuildDBInfo[any](ctx, d, d.config.Concurrency, d.config.Only, d.config.Except)
	if err != nil {
		return nil, err
	}

	dbinfo.Enums = make([]drivers.Enum, len(d.enums))
	for i, e := range d.enums {
		dbinfo.Enums[i] = drivers.Enum{
			Type:   e.Type,
			Values: e.Values,
		}
	}

	sort.Slice(dbinfo.Enums, func(i, j int) bool {
		return dbinfo.Enums[i].Type < dbinfo.Enums[j].Type
	})

	return dbinfo, err
}

const keyClause = "(CASE WHEN table_schema <> $1 THEN table_schema|| '.'  ELSE '' END || table_name)"

// TableNames connects to the postgres database and
// retrieves all table names from the information_schema where the
// table schema is schema. It uses a whitelist and blacklist.
func (d *driver) TablesInfo(ctx context.Context, tableFilter drivers.Filter) (drivers.TablesInfo, error) {
	query := fmt.Sprintf(`SELECT
	  %s AS "key" ,
	  table_schema AS "schema",
	  table_name AS "name"
	FROM (
	  SELECT
		table_name,
		table_schema
	  FROM
		information_schema.tables
	  UNION
	  SELECT
		matviewname AS table_name,
		schemaname AS table_schema
	  FROM
		pg_matviews) AS v
	WHERE
	  v.table_schema = ANY ($2)`, keyClause)
	args := []any{d.config.SharedSchema, d.config.Schemas}

	include := tableFilter.Only
	exclude := tableFilter.Except

	if len(include) > 0 {
		var subqueries []string
		stringPatterns, regexPatterns := tableFilter.ClassifyPatterns(include)
		if len(stringPatterns) > 0 {
			subqueries = append(subqueries, fmt.Sprintf("%s in (%s)", keyClause, strmangle.Placeholders(true, len(stringPatterns), len(args)+1, 1)))
			for _, w := range stringPatterns {
				args = append(args, w)
			}
		}
		if len(regexPatterns) > 0 {
			subqueries = append(subqueries, fmt.Sprintf("%s ~ (%s)", keyClause, strmangle.Placeholders(true, 1, len(args)+1, 1)))
			args = append(args, strings.Join(regexPatterns, "|"))
		}
		query += fmt.Sprintf(" and (%s)", strings.Join(subqueries, " or "))
	}

	if len(exclude) > 0 {
		var subqueries []string
		stringPatterns, regexPatterns := tableFilter.ClassifyPatterns(exclude)
		if len(stringPatterns) > 0 {
			subqueries = append(subqueries, fmt.Sprintf("%s not in (%s)", keyClause, strmangle.Placeholders(true, len(stringPatterns), len(args)+1, 1)))
			for _, w := range stringPatterns {
				args = append(args, w)
			}
		}
		if len(regexPatterns) > 0 {
			subqueries = append(subqueries, fmt.Sprintf("%s !~ (%s)", keyClause, strmangle.Placeholders(true, 1, len(args)+1, 1)))
			args = append(args, strings.Join(regexPatterns, "|"))
		}
		query += fmt.Sprintf(" and (%s)", strings.Join(subqueries, " and "))
	}

	query += ` order by table_name;`

	return stdscan.All(ctx, d.conn, scan.StructMapper[drivers.TableInfo](), query, args...)
}

// Load details about a single table
func (d *driver) TableDetails(ctx context.Context, info drivers.TableInfo, colFilter drivers.ColumnFilter) (string, string, []drivers.Column, error) {
	var columns []drivers.Column
	args := []any{info.Schema, info.Name}

	tableQuery := `
	SELECT
		c.ordinal_position,
		c.column_name,
		ct.column_type,
		c.udt_schema,
		c.udt_name,
		(
			SELECT
				data_type
			FROM
				information_schema.element_types e
			WHERE
				c.table_catalog = e.object_catalog
				AND c.table_schema = e.object_schema
				AND c.table_name = e.object_name
				AND 'TABLE' = e.object_type
				AND c.dtd_identifier = e.collection_type_identifier
		) AS array_type,
	c.domain_name,
	c.column_default,
	coalesce(col_description(('"' || c.table_schema || '"."' || c.table_name || '"')::regclass::oid, ordinal_position), '') AS column_comment,
	c.is_nullable = 'YES' AS is_nullable,
	(
		CASE WHEN c.is_generated = 'ALWAYS'
			OR c.identity_generation = 'ALWAYS' THEN
			TRUE
		ELSE
			FALSE
		END) AS is_generated,
	(
		CASE WHEN (
			SELECT
				CASE WHEN column_name = 'is_identity' THEN
				(
					SELECT
						c.is_identity = 'YES' AS is_identity)
				ELSE
					FALSE
				END AS is_identity
			FROM
				information_schema.columns
			WHERE
				table_schema = 'information_schema'
				AND table_name = 'columns'
				AND column_name = 'is_identity') IS NULL THEN
			'NO'
		ELSE
			is_identity
		END) = 'YES' AS is_identity
	FROM
		information_schema.columns AS c
		INNER JOIN pg_namespace AS pgn ON pgn.nspname = c.udt_schema
		LEFT JOIN pg_type pgt ON c.data_type = 'USER-DEFINED'
			AND pgn.oid = pgt.typnamespace
			AND c.udt_name = pgt.typname,
			LATERAL (
				SELECT
					(
						CASE WHEN pgt.typtype = 'e' THEN
							'ENUM'
						ELSE
							c.data_type
						END) AS column_type) ct
	WHERE c.table_name = $2 and c.table_schema = $1
	ORDER BY c.ordinal_position`

	//nolint:gosec
	query := fmt.Sprintf(`SELECT 
		column_name,
		column_type,
		udt_schema,
		udt_name,
		array_type,
		domain_name,
		column_default,
		column_comment,
		is_nullable,
		is_generated,
		is_identity
	FROM (
		%s
	) AS c`, tableQuery) // matviewQuery, tableQuery)

	filter := colFilter[info.Key]
	only := filter.Only
	except := filter.Except

	if len(only) > 0 || len(except) > 0 {
		query += " where "
	}

	if len(only) > 0 {
		query += fmt.Sprintf("c.column_name in (%s)", strmangle.Placeholders(true, len(only), 3, 1))
		for _, w := range only {
			args = append(args, w)
		}
	} else if len(except) > 0 {
		query += fmt.Sprintf("c.column_name not in (%s)", strmangle.Placeholders(true, len(except), 3, 1))
		for _, w := range except {
			args = append(args, w)
		}
	}

	query += ` order by c.ordinal_position;`

	rows, err := d.conn.Query(query, args...)
	if err != nil {
		return "", "", nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var colName, colType, udtSchema, udtName, comment string
		var defaultValue, arrayType, domainName *string
		var nullable, generated, identity bool
		if err := rows.Scan(&colName, &colType, &udtSchema, &udtName, &arrayType, &domainName, &defaultValue, &comment, &nullable, &generated, &identity); err != nil {
			return "", "", nil, fmt.Errorf("unable to scan for table %s: %w", info.Key, err)
		}

		column := drivers.Column{
			Name:      colName,
			DBType:    colType,
			Comment:   comment,
			Nullable:  nullable,
			Generated: generated,
		}
		info := colInfo{
			UDTSchema: udtSchema,
			UDTName:   udtName,
		}

		if arrayType != nil {
			info.ArrType = *arrayType
		}

		if domainName != nil {
			column.DomainName = *domainName
		}

		if defaultValue != nil {
			column.Default = *defaultValue
		}

		if identity {
			column.Default = "IDENTITY"
		}

		// A generated column technically has a default value
		if generated && column.Default == "" {
			column.Default = "GENERATED"
		}

		// A nullable column can always default to NULL
		if nullable && column.Default == "" {
			column.Default = "NULL"
		}

		columns = append(columns, d.translateColumnType(column, info))
	}

	schema := info.Schema
	if schema == d.config.SharedSchema {
		schema = ""
	}

	return schema, info.Name, columns, nil
}

func (d *driver) loadEnums(ctx context.Context) error {
	if d.enums != nil {
		return nil
	}

	query := `SELECT pg_namespace.nspname AS schema, pg_type.typname AS name, array_agg(pg_enum.enumlabel order by pg_enum.enumsortorder) AS values
		FROM pg_type
		JOIN pg_enum ON pg_enum.enumtypid = pg_type.oid
		JOIN pg_namespace ON pg_namespace.oid = pg_type.typnamespace
		WHERE pg_namespace.nspname = ANY($1)
		GROUP BY schema, name`

	var err error
	d.enums, err = stdscan.All(
		ctx, d.conn,
		func(_ context.Context, _ []string) (scan.BeforeFunc, func(any) (Enum, error)) {
			return func(r *scan.Row) (any, error) {
					var e Enum
					r.ScheduleScan("schema", &e.Schema)
					r.ScheduleScan("name", &e.Name)
					r.ScheduleScan("values", &e.Values)
					return &e, nil
				}, func(a any) (Enum, error) {
					e := a.(*Enum)
					if e.Schema != "" && e.Schema != d.config.SharedSchema {
						e.Type = strmangle.TitleCase(e.Schema + "_" + e.Name)
					} else {
						e.Type = strmangle.TitleCase(e.Name)
					}

					return *e, nil
				}
		},
		query, d.config.Schemas,
	)
	if err != nil {
		return err
	}

	return nil
}

func (d *driver) Indexes(ctx context.Context) (drivers.DBIndexes[IndexExtra], error) {
	ret := drivers.DBIndexes[IndexExtra]{}

	query := `SELECT	
          n.nspname AS schema_name,
          t.relname AS table_name,
          i.relname AS index_name,
          a.amname AS type,
          cols.cols[:x.indnkeyatts] AS index_cols,
          ARRAY(SELECT unnest(x.indoption) & 1 = 1 ) AS descending,
          ARRAY(SELECT unnest(x.indoption) & 2 = 2 ) AS nulls_first,
          x.indisunique as unique,
          x.indnullsnotdistinct as nulls_not_distinct,
          pg_get_expr(x.indpred, x.indrelid) AS where_clause,
          cols.cols[x.indnkeyatts+1:] AS included_cols,
          obj_description(x.indexrelid, 'pg_class') AS comment
	  FROM pg_index x
	  JOIN pg_class t ON t.oid = x.indrelid
	  JOIN pg_class i ON i.oid = x.indexrelid
      JOIN pg_am a on i.relam = a.oid
	  JOIN pg_namespace n ON n.oid = t.relnamespace
	  JOIN (
	    SELECT x.indexrelid, array_agg(cols.cols) cols
      FROM pg_index x
        LEFT JOIN (SELECT a.attrelid, pg_get_indexdef(a.attrelid, a.attnum, TRUE) AS cols
          FROM pg_attribute a) cols ON cols.attrelid = x.indexrelid
      WHERE cols IS NOT NULL
      GROUP BY x.indexrelid
    ) cols ON cols.indexrelid = x.indexrelid
	WHERE n.nspname = ANY($1)
	    AND x.indisvalid AND x.indislive AND x.indisvalid
	ORDER BY n.nspname, t.relname, x.indisprimary DESC, i.relname;`

	type indexColumns struct {
		SchemaName       string
		TableName        string
		IndexName        string
		Type             string
		IndexCols        pq.StringArray // a list of column names and/or expressions
		Descending       pq.BoolArray
		NullsFirst       pq.BoolArray
		Unique           bool
		NullsNotDistinct bool
		WhereClause      sql.NullString
		IncludedCols     pq.StringArray
		Comment          sql.NullString
	}
	res, err := stdscan.All(ctx, d.conn, scan.StructMapper[indexColumns](), query, d.config.Schemas)
	if err != nil {
		return nil, err
	}
	for _, r := range res {
		key := r.TableName
		if r.SchemaName != "" && r.SchemaName != d.config.SharedSchema {
			key = r.SchemaName + "." + r.TableName
		}
		index := drivers.Index[IndexExtra]{
			Type:    r.Type,
			Name:    r.IndexName,
			Unique:  r.Unique,
			Comment: r.Comment.String,
			Extra: IndexExtra{
				NullsFirst:    r.NullsFirst,
				NullsDistinct: r.NullsNotDistinct,
				Where:         r.WhereClause.String,
				Include:       r.IncludedCols,
			},
		}
		for i, colName := range r.IndexCols {
			isExpression := !rgxValidColumnName.MatchString(colName)
			index.Columns = append(index.Columns, drivers.IndexColumn{
				Name:         colName,
				Desc:         r.Descending[i],
				IsExpression: isExpression,
			})
		}
		ret[key] = append(ret[key], index)
	}

	return ret, nil
}

func (d *driver) Comments(ctx context.Context) (map[string]string, error) {
	query := fmt.Sprintf(`SELECT
	  %s AS "key",
      obj_description((table_schema||'.'||table_name)::regclass::oid, 'pg_class') AS comment
	FROM (
	  SELECT
		table_name,
		table_schema
	  FROM
		information_schema.tables
	  UNION
	  SELECT
		matviewname AS table_name,
		schemaname AS table_schema
	  FROM
		pg_matviews) AS v
	WHERE
	  v.table_schema = ANY ($2)`, keyClause)
	args := []any{d.config.SharedSchema, d.config.Schemas}

	comments := make(map[string]string)

	for row, err := range stdscan.Each(ctx, d.conn, scan.StructMapper[struct {
		Key     string
		Comment sql.NullString
	}](), query, args...) {
		if err != nil {
			return nil, err
		}
		comments[row.Key] = row.Comment.String
	}

	return comments, nil
}
