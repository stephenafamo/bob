package driver

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"sort"

	"github.com/friendsofgo/errors"
	"github.com/lib/pq"
	"github.com/stephenafamo/bob/gen/drivers"
	"github.com/stephenafamo/scan"
	"github.com/stephenafamo/scan/stdscan"
	"github.com/volatiletech/strmangle"
)

//go:embed templates
var templates embed.FS

//nolint:gochecknoglobals
var (
	ModelTemplates, _   = fs.Sub(templates, "templates/models")
	FactoryTemplates, _ = fs.Sub(templates, "templates/factory")
)

type (
	Interface = drivers.Interface[Extra]
	DBInfo    = drivers.DBInfo[Extra]
	Extra     struct {
		Enums []Enum
	}
)

func New(config Config) Interface {
	return &Driver{config: config}
}

// Driver holds the database connection string and a handle
// to the database connection.
type Driver struct {
	config Config

	conn *sql.DB

	enums         []Enum
	uniqueColumns map[columnIdentifier]struct{}
}

type Config struct {
	// The database connection string
	Dsn string
	// The database schemas to generate models for
	Schemas pq.StringArray
	// The name of this schema will not be included in the generated models
	// a context value can then be used ot set the schema at runtime
	// useful for multi-tenant setups
	SharedSchema string
	// List of tables that will be included. Others are ignored
	Only []string
	// List of tables that will be should be ignored. Others are included
	Except   map[string][]string
	Excludes []string
	// How many tables to fetch in parallel
	Concurrency int

	// Used in main.go

	// The name of the folder to output the models package to
	Output string
	// The name you wish to assign to your generated models package
	Pkgname   string
	NoFactory bool `yaml:"no_factory"`
}

type columnIdentifier struct {
	Schema string
	Table  string
	Column string
}

// Assemble all the information we need to provide back to the driver
func (d *Driver) Assemble() (*DBInfo, error) {
	var dbinfo *DBInfo
	var err error

	defer func() {
		if r := recover(); r != nil && err == nil {
			dbinfo = nil
			err = r.(error)
		}
	}()

	d.conn, err = sql.Open("postgres", d.config.Dsn)
	if err != nil {
		return nil, errors.Wrap(err, "failed to connect to database")
	}

	defer func() {
		if e := d.conn.Close(); e != nil {
			dbinfo = nil
			err = e
		}
	}()

	dbinfo = &DBInfo{}

	if err := d.loadUniqueColumns(); err != nil {
		return nil, errors.Wrapf(err, "unable to load unique data")
	}

	// drivers.Tables call translateColumnType which uses Enums
	if err := d.loadEnums(); err != nil {
		return nil, errors.Wrapf(err, "unable to load enums")
	}

	dbinfo.Tables, err = drivers.Tables(d, d.config.Concurrency, d.config.Only, d.config.Excludes)
	if err != nil {
		return nil, err
	}

	dbinfo.ExtraInfo.Enums, err = d.Enums()
	if err != nil {
		return nil, err
	}

	return dbinfo, err
}

const keyClause = "(CASE WHEN table_schema <> $1 THEN table_schema|| '.'  ELSE '' END || table_name)"

// TableNames connects to the postgres database and
// retrieves all table names from the information_schema where the
// table schema is schema. It uses a whitelist and blacklist.
func (d *Driver) TablesInfo(tableFilter drivers.Filter) (drivers.TablesInfo, error) {
	query := fmt.Sprintf(`SELECT
	  %s AS "key" ,
	  table_schema AS "schema",
	  table_name AS "name"
	FROM
	  information_schema.tables
	WHERE
	  table_schema = ANY($2)
	  AND table_type = 'BASE TABLE'`, keyClause)

	args := []any{d.config.SharedSchema, d.config.Schemas}

	include := tableFilter.Include
	exclude := tableFilter.Exclude

	if len(include) > 0 {
		query += fmt.Sprintf(" and %s in (%s)", keyClause, strmangle.Placeholders(true, len(include), 3, 1))
		for _, w := range include {
			args = append(args, w)
		}
	}

	if len(exclude) > 0 {
		query += fmt.Sprintf(" and %s not in (%s)", keyClause, strmangle.Placeholders(true, len(exclude), 3+len(include), 1))
		for _, w := range exclude {
			args = append(args, w)
		}
	}

	query += ` order by table_name;`

	ctx := context.Background()
	return stdscan.All(ctx, d.conn, scan.StructMapper[drivers.TableInfo](), query, args...)
}

// ViewNames connects to the postgres database and
// retrieves all view names from the information_schema where the
// view schema is schema. It uses a whitelist and blacklist.
func (d *Driver) ViewsInfo(tableFilter drivers.Filter) (drivers.TablesInfo, error) {
	query := fmt.Sprintf(`SELECT
	  %s AS "key" ,
	  table_schema AS "schema",
	  table_name AS "name"
	FROM (
	  SELECT
		table_name,
		table_schema
	  FROM
		information_schema.views
	  UNION
	  SELECT
		matviewname AS table_name,
		schemaname AS table_schema
	  FROM
		pg_matviews) AS v
	WHERE
	  v.table_schema = ANY ($2)`, keyClause)
	args := []any{d.config.SharedSchema, d.config.Schemas}

	include := tableFilter.Include
	exclude := tableFilter.Exclude

	if len(include) > 0 {
		query += fmt.Sprintf(" and %s in (%s)", keyClause, strmangle.Placeholders(true, len(include), 3, 1))
		for _, w := range include {
			args = append(args, w)
		}
	}

	if len(exclude) > 0 {
		query += fmt.Sprintf(" and %s not in (%s)", keyClause, strmangle.Placeholders(true, len(exclude), 3+len(include), 1))
		for _, w := range exclude {
			args = append(args, w)
		}
	}

	query += ` order by table_name;`

	ctx := context.Background()
	return stdscan.All(ctx, d.conn, scan.StructMapper[drivers.TableInfo](), query, args...)
}

func (d *Driver) loadEnums() error {
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
		context.Background(), d.conn,
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

type Enum struct {
	Schema string
	Name   string
	Type   string
	Values pq.StringArray
}

func (d *Driver) Enums() ([]Enum, error) {
	enums := make([]Enum, len(d.enums))
	copy(enums, d.enums)

	sort.Slice(enums, func(i, j int) bool {
		return enums[i].Name < enums[j].Name
	})

	return enums, nil
}

func (d *Driver) ViewColumns(info drivers.TableInfo, filter drivers.ColumnFilter) (string, string, []drivers.Column, error) {
	return d.TableColumns(info, filter)
}

// TableColumns takes a table name and attempts to retrieve the table information
// from the database information_schema.columns. It retrieves the column names
// and column types and returns those as a []Column after translateColumnType()
// converts the SQL types to Go types, for example: "varchar" to "string"
func (d *Driver) TableColumns(info drivers.TableInfo, colFilter drivers.ColumnFilter) (string, string, []drivers.Column, error) {
	var columns []drivers.Column
	args := []interface{}{info.Schema, info.Name}

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
	WHERE c.table_name = $2 and c.table_schema = $1`

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

	allfilter := colFilter["*"]
	filter := colFilter[info.Key]
	include := append(allfilter.Include, filter.Include...)
	exclude := append(allfilter.Exclude, filter.Exclude...)

	if len(include) > 0 || len(exclude) > 0 {
		query += " where "
	}

	if len(include) > 0 {
		query += fmt.Sprintf("c.column_name in (%s)", strmangle.Placeholders(true, len(include), 3, 1))
		for _, w := range include {
			args = append(args, w)
		}
	}

	if len(include) > 0 && len(exclude) > 0 {
		query += " and "
	}

	if len(exclude) > 0 {
		query += fmt.Sprintf("c.column_name not in (%s)", strmangle.Placeholders(true, len(exclude), 3, 1))
		for _, w := range exclude {
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
			return "", "", nil, errors.Wrapf(err, "unable to scan for table %s", info.Key)
		}

		_, unique := d.uniqueColumns[columnIdentifier{info.Schema, info.Name, colName}]
		column := drivers.Column{
			Name:      colName,
			DBType:    colType,
			UDTSchema: udtSchema,
			UDTName:   udtName,
			Comment:   comment,
			Nullable:  nullable,
			Generated: generated,
			Unique:    unique,
		}

		if arrayType != nil {
			column.ArrType = *arrayType
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

		columns = append(columns, d.translateColumnType(column))
	}

	schema := info.Schema
	if schema == d.config.SharedSchema {
		schema = ""
	}

	return schema, info.Name, columns, nil
}
