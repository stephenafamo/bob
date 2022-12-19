package driver

import (
	"context"
	"database/sql"

	"github.com/friendsofgo/errors"
	"github.com/lib/pq"
	"github.com/stephenafamo/bob/gen/drivers"
	"github.com/stephenafamo/scan"
	"github.com/stephenafamo/scan/stdscan"
)

func (p *Driver) Constraints(_ drivers.ColumnFilter) (drivers.DBConstraints, error) {
	ret := drivers.DBConstraints{
		PKs:     map[string]*drivers.Constraint{},
		FKs:     map[string][]drivers.ForeignKey{},
		Uniques: map[string][]drivers.Constraint{},
	}

	query := `SELECT 
		rel.relname as table
		, con.conname as name
		, con.contype as type
		, max(out.relname) as foreign_table
		, array_agg(local_cols.column_name) as columns
		, (
			case when con.contype = 'f'
			then array_agg(foreign_cols.column_name)
			else array[]::text[] end
		) as foreign_columns
	FROM pg_catalog.pg_constraint con
	INNER JOIN pg_catalog.pg_class rel
		ON rel.oid = con.conrelid
	LEFT JOIN pg_catalog.pg_class out
		ON out.oid = con.confrelid
	INNER JOIN pg_catalog.pg_namespace nsp
		ON nsp.oid = connamespace
	LEFT JOIN information_schema.columns local_cols
		ON local_cols.table_schema = nsp.nspname 
		AND local_cols.table_name = rel.relname 
		AND local_cols.ordinal_position = ANY(con.conkey)
	LEFT JOIN information_schema.columns foreign_cols
		ON foreign_cols.table_schema = nsp.nspname 
		AND foreign_cols.table_name = out.relname 
		AND foreign_cols.ordinal_position = ANY(con.confkey)
	WHERE nsp.nspname = $1
	AND con.contype IN ('p', 'f', 'u')
	GROUP BY rel.relname, name, con.contype`

	ctx := context.Background()
	constraints, err := stdscan.All(ctx, p.conn, scan.StructMapper[struct {
		Table          string
		Name           string
		Type           string
		Columns        pq.StringArray
		ForeignTable   sql.NullString
		ForeignColumns pq.StringArray
	}](), query, p.config.Schema)
	if err != nil {
		return ret, err
	}

	for _, c := range constraints {
		switch c.Type {
		case "p":
			ret.PKs[c.Table] = &drivers.Constraint{
				Name:    c.Name,
				Columns: c.Columns,
			}
		case "u":
			ret.Uniques[c.Table] = append(ret.Uniques[c.Table], drivers.Constraint{
				Name:    c.Name,
				Columns: c.Columns,
			})
		case "f":
			ret.FKs[c.Table] = append(ret.FKs[c.Table], drivers.ForeignKey{
				Constraint: drivers.Constraint{
					Name:    c.Name,
					Columns: c.Columns,
				},
				ForeignTable:   c.ForeignTable.String,
				ForeignColumns: c.ForeignColumns,
			})
		}
	}

	return ret, nil
}

// loadUniqueColumns is responsible for populating p.uniqueColumns with an entry
// for every table or view column that is made unique by an index or constraint.
// This information is queried once, rather than for each table, for performance
// reasons.
func (p *Driver) loadUniqueColumns() error {
	if p.uniqueColumns != nil {
		return nil
	}
	p.uniqueColumns = map[columnIdentifier]struct{}{}
	query := `with
method_a as (
    select
        tc.table_schema as schema_name,
        ccu.table_name as table_name,
        ccu.column_name as column_name
    from information_schema.table_constraints tc
    inner join information_schema.constraint_column_usage as ccu
        on tc.constraint_name = ccu.constraint_name
    where
        tc.constraint_type = 'UNIQUE' and (
            (select count(*)
            from information_schema.constraint_column_usage
            where constraint_schema = tc.table_schema and constraint_name = tc.constraint_name
            ) = 1
        )
),
method_b as (
    select
        pgix.schemaname as schema_name,
        pgix.tablename as table_name,
        pga.attname as column_name
    from pg_indexes pgix
    inner join pg_class pgc on pgix.indexname = pgc.relname and pgc.relkind = 'i' and pgc.relnatts = 1
    inner join pg_index pgi on pgi.indexrelid = pgc.oid
    inner join pg_attribute pga on pga.attrelid = pgi.indrelid and pga.attnum = ANY(pgi.indkey)
    where pgi.indisunique = true
),
results as (
    select * from method_a
    union
    select * from method_b
)
select * from results;
`
	rows, err := p.conn.Query(query)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var c columnIdentifier
		if err := rows.Scan(&c.Schema, &c.Table, &c.Column); err != nil {
			return errors.Wrapf(err, "unable to scan unique entry row")
		}
		p.uniqueColumns[c] = struct{}{}
	}
	return nil
}
