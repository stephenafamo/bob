package driver

import (
	"context"
	"database/sql"

	"github.com/lib/pq"
	"github.com/stephenafamo/bob/gen/drivers"
	"github.com/stephenafamo/scan"
	"github.com/stephenafamo/scan/stdscan"
)

func (d *Driver) Constraints(ctx context.Context, _ drivers.ColumnFilter) (drivers.DBConstraints, error) {
	ret := drivers.DBConstraints{
		PKs:     map[string]*drivers.Constraint{},
		FKs:     map[string][]drivers.ForeignKey{},
		Uniques: map[string][]drivers.Constraint{},
	}

	query := `SELECT 
		nsp.nspname as schema
		, rel.relname as table
		, con.conname as name
		, con.contype as type
		, max(fnsp.nspname) as foreign_schema
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
		ON nsp.oid = rel.relnamespace
		
	LEFT JOIN pg_catalog.pg_namespace fnsp
		ON fnsp.oid = out.relnamespace
		
	LEFT JOIN information_schema.columns local_cols
		ON local_cols.table_schema = nsp.nspname 
		AND local_cols.table_name = rel.relname 
		AND local_cols.ordinal_position = ANY(con.conkey)
		
	LEFT JOIN information_schema.columns foreign_cols
		ON foreign_cols.table_schema = fnsp.nspname 
		AND foreign_cols.table_name = out.relname 
		AND foreign_cols.ordinal_position = ANY(con.confkey)
		
	WHERE nsp.nspname = ANY($1)
	AND con.contype IN ('p', 'f', 'u')
	GROUP BY nsp.nspname, rel.relname, name, con.contype
	ORDER BY nsp.nspname, rel.relname, name, con.contype`

	constraints, err := stdscan.All(ctx, d.conn, scan.StructMapper[struct {
		Schema         string
		Table          string
		Name           string
		Type           string
		Columns        pq.StringArray
		ForeignSchema  sql.NullString
		ForeignTable   sql.NullString
		ForeignColumns pq.StringArray
	}](), query, d.config.Schemas)
	if err != nil {
		return ret, err
	}

	for _, c := range constraints {
		key := c.Table
		if c.Schema != "" && c.Schema != d.config.SharedSchema {
			key = c.Schema + "." + c.Table
		}

		switch c.Type {
		case "p":
			ret.PKs[key] = &drivers.Constraint{
				Name:    c.Name,
				Columns: c.Columns,
			}
		case "u":
			ret.Uniques[key] = append(ret.Uniques[c.Table], drivers.Constraint{
				Name:    c.Name,
				Columns: c.Columns,
			})
		case "f":
			fkey := c.ForeignTable.String
			if c.ForeignSchema.Valid && c.ForeignSchema.String != d.config.SharedSchema {
				fkey = c.ForeignSchema.String + "." + c.ForeignTable.String
			}
			ret.FKs[key] = append(ret.FKs[key], drivers.ForeignKey{
				Constraint: drivers.Constraint{
					Name:    c.Name,
					Columns: c.Columns,
				},
				ForeignTable:   fkey,
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
func (d *Driver) loadUniqueColumns() error {
	if d.uniqueColumns != nil {
		return nil
	}

	d.uniqueColumns = map[columnIdentifier]struct{}{}

	query := `with
method_a as (
    select
        tc.table_schema as schema,
        ccu.table_name as table,
        ccu.column_name as column
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
        pgix.schemaname as schema,
        pgix.tablename as table,
        pga.attname as column
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
select * from results where schema = ANY($1);
`
	ctx := context.Background()
	colIds, err := stdscan.All(
		ctx, d.conn,
		scan.StructMapper[columnIdentifier](),
		query, d.config.Schemas,
	)
	if err != nil {
		return err
	}

	for _, c := range colIds {
		d.uniqueColumns[c] = struct{}{}
	}
	return nil
}
