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
	ModelTemplates, _ = fs.Sub(templates, "templates/models")
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

	conn    *sql.DB
	version int

	enums         map[string]Enum
	uniqueColumns map[columnIdentifier]struct{}
}

type Config struct {
	Dsn         string
	Schema      string
	Includes    []string
	Excludes    []string
	Concurrency int
}

type columnIdentifier struct {
	Schema string
	Table  string
	Column string
}

// Assemble all the information we need to provide back to the driver
func (p *Driver) Assemble() (*DBInfo, error) {
	var dbinfo *DBInfo
	var err error

	defer func() {
		if r := recover(); r != nil && err == nil {
			dbinfo = nil
			err = r.(error)
		}
	}()

	p.conn, err = sql.Open("postgres", p.config.Dsn)
	if err != nil {
		return nil, errors.Wrap(err, "failed to connect to database")
	}

	defer func() {
		if e := p.conn.Close(); e != nil {
			dbinfo = nil
			err = e
		}
	}()

	p.version, err = p.getVersion()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get database version")
	}

	dbinfo = &DBInfo{Schema: p.config.Schema}

	if err := p.loadUniqueColumns(); err != nil {
		return nil, errors.Wrapf(err, "unable to load unique data")
	}

	// drivers.Tables call translateColumnType which uses Enums
	if err := p.loadEnums(); err != nil {
		return nil, errors.Wrapf(err, "unable to load enums")
	}

	dbinfo.Tables, err = drivers.Tables(p, p.config.Concurrency, p.config.Includes, p.config.Excludes)
	if err != nil {
		return nil, err
	}

	dbinfo.ExtraInfo.Enums, err = p.Enums(p.config.Schema)
	if err != nil {
		return nil, err
	}

	return dbinfo, err
}

// TableNames connects to the postgres database and
// retrieves all table names from the information_schema where the
// table schema is schema. It uses a whitelist and blacklist.
func (p *Driver) TableNames(tableFilter drivers.Filter) ([]string, error) {
	query := `select table_name from information_schema.tables where table_schema = $1 and table_type = 'BASE TABLE'`
	args := []interface{}{p.config.Schema}

	include := tableFilter.Include
	exclude := tableFilter.Exclude

	if len(include) > 0 {
		query += fmt.Sprintf(" and table_name in (%s)", strmangle.Placeholders(true, len(include), 2, 1))
		for _, w := range include {
			args = append(args, w)
		}
	}

	if len(exclude) > 0 {
		query += fmt.Sprintf(" and table_name not in (%s)", strmangle.Placeholders(true, len(exclude), 2+len(include), 1))
		for _, w := range exclude {
			args = append(args, w)
		}
	}

	query += ` order by table_name;`

	ctx := context.Background()
	return stdscan.All(ctx, p.conn, scan.SingleColumnMapper[string], query, args...)
}

// ViewNames connects to the postgres database and
// retrieves all view names from the information_schema where the
// view schema is schema. It uses a whitelist and blacklist.
func (p *Driver) ViewNames(tableFilter drivers.Filter) ([]string, error) {
	query := `select 
		table_name 
	from (
			select 
				table_name, 
				table_schema 
			from information_schema.views
			UNION
			select 
				matviewname as table_name, 
				schemaname as table_schema 
			from pg_matviews 
	) as v where v.table_schema= $1`
	args := []interface{}{p.config.Schema}

	include := tableFilter.Include
	exclude := tableFilter.Exclude

	if len(include) > 0 {
		query += fmt.Sprintf(" and table_name in (%s)", strmangle.Placeholders(true, len(include), 2, 1))
		for _, w := range include {
			args = append(args, w)
		}
	}

	if len(exclude) > 0 {
		query += fmt.Sprintf(" and table_name not in (%s)", strmangle.Placeholders(true, len(exclude), 2+len(include), 1))
		for _, w := range exclude {
			args = append(args, w)
		}
	}

	query += ` order by table_name;`

	ctx := context.Background()
	return stdscan.All(ctx, p.conn, scan.SingleColumnMapper[string], query, args...)
}

func (p *Driver) loadEnums() error {
	if p.enums != nil {
		return nil
	}
	p.enums = map[string]Enum{}

	query := `SELECT pg_type.typname AS type, array_agg(pg_enum.enumlabel order by pg_enum.enumsortorder) AS values
		FROM pg_type
		JOIN pg_enum ON pg_enum.enumtypid = pg_type.oid
		JOIN pg_namespace ON pg_namespace.oid = pg_type.typnamespace
		WHERE pg_namespace.nspname = $1
		GROUP BY type`

	rows, err := p.conn.Query(query, p.config.Schema)
	if err != nil {
		return err
	}

	defer rows.Close()
	for rows.Next() {
		var name string
		var values pq.StringArray
		if err := rows.Scan(&name, &values); err != nil {
			return err
		}

		p.enums[name] = Enum{
			Name:   name,
			Type:   strmangle.TitleCase(name),
			Values: values,
		}
	}

	return nil
}

type Enum struct {
	Name   string
	Type   string
	Values []string
}

func (p *Driver) Enums(schema string) ([]Enum, error) {
	enums := make([]Enum, 0, len(p.enums))
	for _, e := range p.enums {
		enums = append(enums, e)
	}

	sort.Slice(enums, func(i, j int) bool {
		return enums[i].Name < enums[j].Name
	})

	return enums, nil
}

func (p *Driver) ViewColumns(tableName string, filter drivers.ColumnFilter) ([]drivers.Column, error) {
	return p.TableColumns(tableName, filter)
}

// TableColumns takes a table name and attempts to retrieve the table information
// from the database information_schema.columns. It retrieves the column names
// and column types and returns those as a []Column after translateColumnType()
// converts the SQL types to Go types, for example: "varchar" to "string"
func (p *Driver) TableColumns(tableName string, colFilter drivers.ColumnFilter) ([]drivers.Column, error) {
	var columns []drivers.Column
	args := []interface{}{p.config.Schema, tableName}

	matviewQuery := `WITH cte_pg_attribute AS (
		SELECT
			pg_catalog.format_type(a.atttypid, NULL) LIKE '%[]' = TRUE as is_array,
			pg_catalog.format_type(a.atttypid, a.atttypmod) as column_full_type,
			a.*
		FROM pg_attribute a
	), cte_pg_namespace AS (
		SELECT
			n.nspname NOT IN ('pg_catalog', 'information_schema') = TRUE as is_user_defined,
			n.oid
		FROM pg_namespace n
	), cte_information_schema_domains AS (
		SELECT
			domain_name IS NOT NULL = TRUE as is_domain,
			data_type LIKE '%[]' = TRUE as is_array,
			domain_name,
			udt_name,
			data_type
		FROM information_schema.domains
	)
	SELECT 
		a.attnum as ordinal_position,
		a.attname as column_name,
		(
			case 
			when t.typtype = 'e'
			then 'enum.' || t.typname
			when a.is_array OR d.is_array
			then 'ARRAY'
			when d.is_domain
			then d.data_type
			when tn.is_user_defined
			then 'USER-DEFINED'
			else pg_catalog.format_type(a.atttypid, NULL)
			end
		) as column_type,
		(
			case 
			when d.is_domain
			then d.udt_name		
			when a.column_full_type LIKE '%(%)%' AND t.typcategory IN ('S', 'V')
			then a.column_full_type
			else t.typname
			end
		) as column_full_type,
		(
			case 
			when d.is_domain
			then d.udt_name		
			else t.typname
			end
		) as udt_name,
		(
			case when a.is_array
			then
				case when tn.is_user_defined
				then 'USER-DEFINED'
				else RTRIM(pg_catalog.format_type(a.atttypid, NULL), '[]')
				end
			else NULL
			end
		) as array_type,
		d.domain_name,
		NULL as column_default,
		'' as column_comment,
		a.attnotnull = FALSE as is_nullable,
		FALSE as is_generated,
		a.attidentity <> '' as is_identity
	FROM cte_pg_attribute a
		JOIN pg_class c on a.attrelid = c.oid
		JOIN pg_namespace cn on c.relnamespace = cn.oid
		JOIN pg_type t ON t.oid = a.atttypid
		LEFT JOIN cte_pg_namespace tn ON t.typnamespace = tn.oid
		LEFT JOIN cte_information_schema_domains d ON d.domain_name = pg_catalog.format_type(a.atttypid, NULL)
		WHERE a.attnum > 0 
		AND c.relkind = 'm'
		AND NOT a.attisdropped
		AND c.relname = $2
		AND cn.nspname = $1`

	tableQuery := `
	SELECT
		c.ordinal_position,
		c.column_name,
		ct.column_type,
		(
			CASE WHEN c.character_maximum_length != 0 THEN
				(ct.column_type || '(' || c.character_maximum_length || ')')
			ELSE
				c.udt_name
			END) AS column_full_type,
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
							'enum.' || pgt.typname
						ELSE
							c.data_type
						END) AS column_type) ct
	WHERE c.table_name = $2 and c.table_schema = $1`

	//nolint:gosec
	query := fmt.Sprintf(`SELECT 
		column_name,
		column_type,
		column_full_type,
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
		UNION
		%s
	) AS c`, matviewQuery, tableQuery)

	allfilter := colFilter["*"]
	filter := colFilter[tableName]
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

	rows, err := p.conn.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var colName, colType, colFullType, udtName, comment string
		var defaultValue, arrayType, domainName *string
		var nullable, generated, identity bool
		if err := rows.Scan(&colName, &colType, &colFullType, &udtName, &arrayType, &domainName, &defaultValue, &comment, &nullable, &generated, &identity); err != nil {
			return nil, errors.Wrapf(err, "unable to scan for table %s", tableName)
		}

		_, unique := p.uniqueColumns[columnIdentifier{p.config.Schema, tableName, colName}]
		column := drivers.Column{
			Name:       colName,
			DBType:     colType,
			FullDBType: colFullType,
			ArrType:    arrayType,
			DomainName: domainName,
			UDTName:    udtName,
			Comment:    comment,
			Nullable:   nullable,
			Generated:  generated,
			Unique:     unique,
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

		columns = append(columns, p.translateColumnType(column))
	}

	return columns, nil
}

// getVersion gets the version of underlying database
func (p *Driver) getVersion() (int, error) {
	type versionInfoType struct {
		ServerVersionNum int `json:"server_version_num"`
	}
	versionInfo := &versionInfoType{}

	row := p.conn.QueryRow("SHOW server_version_num")
	if err := row.Scan(&versionInfo.ServerVersionNum); err != nil {
		return 0, err
	}

	return versionInfo.ServerVersionNum, nil
}
