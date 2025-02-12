package driver

import (
	"context"
	"database/sql"

	"github.com/lib/pq"
	"github.com/stephenafamo/scan"
	"github.com/stephenafamo/scan/stdscan"
	"github.com/twitter-payments/bob/gen/drivers"
)

func (d *driver) Constraints(ctx context.Context, _ drivers.ColumnFilter) (drivers.DBConstraints[any], error) {
	ret := drivers.DBConstraints[any]{
		PKs:     map[string]*drivers.Constraint[any]{},
		FKs:     map[string][]drivers.ForeignKey[any]{},
		Uniques: map[string][]drivers.Constraint[any]{},
		Checks:  map[string][]drivers.Check[any]{},
	}

	query := `SELECT 
		nsp.nspname as schema
		, rel.relname as table
		, con.conname as name
		, con.contype as type
		, max(fnsp.nspname) as foreign_schema
		, max(out.relname) as foreign_table
		, max(local_cols.columns) as columns
		, max(foreign_cols.columns) as foreign_columns
		, max(pg_get_expr(con.conbin, rel.oid)) AS check_expr
		, max(obj_description(con.oid, 'pg_constraint')) AS comment
	FROM pg_catalog.pg_constraint con
	
	INNER JOIN pg_catalog.pg_class rel
		ON rel.oid = con.conrelid
		
	LEFT JOIN pg_catalog.pg_class out
		ON out.oid = con.confrelid
		
	INNER JOIN pg_catalog.pg_namespace nsp
		ON nsp.oid = rel.relnamespace
		
	LEFT JOIN pg_catalog.pg_namespace fnsp
		ON fnsp.oid = out.relnamespace
	
	LEFT JOIN LATERAL (
		SELECT table_schema, table_name, array_agg(column_name ORDER BY pos.ordinality) AS columns
		FROM unnest(con.conkey) with ordinality pos
		LEFT JOIN information_schema.columns
		ON ordinal_position = pos.pos
		GROUP BY table_schema, table_name
	) AS local_cols
	ON local_cols.table_schema = nsp.nspname
	AND local_cols.table_name = rel.relname

	LEFT JOIN LATERAL (
		SELECT table_schema, table_name, array_agg(column_name ORDER BY pos.ordinality) AS columns
		FROM unnest(con.confkey) with ordinality pos
		LEFT JOIN information_schema.columns
		ON ordinal_position = pos.pos
		GROUP BY table_schema, table_name
	) AS foreign_cols
	ON foreign_cols.table_schema = fnsp.nspname
	AND foreign_cols.table_name = out.relname
		
	WHERE nsp.nspname = ANY($1)
	AND con.contype IN ('p', 'f', 'u', 'c')
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
		CheckExpr      sql.NullString
		Comment        sql.NullString
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
			ret.PKs[key] = &drivers.Constraint[any]{
				Name:    c.Name,
				Columns: c.Columns,
				Comment: c.Comment.String,
			}

		case "u":
			ret.Uniques[key] = append(ret.Uniques[c.Table], drivers.Constraint[any]{
				Name:    c.Name,
				Columns: c.Columns,
				Comment: c.Comment.String,
			})

		case "f":
			fkey := c.ForeignTable.String
			if c.ForeignSchema.Valid && c.ForeignSchema.String != d.config.SharedSchema {
				fkey = c.ForeignSchema.String + "." + c.ForeignTable.String
			}
			ret.FKs[key] = append(ret.FKs[key], drivers.ForeignKey[any]{
				Constraint: drivers.Constraint[any]{
					Name:    key + "." + c.Name,
					Columns: c.Columns,
					Comment: c.Comment.String,
				},
				ForeignTable:   fkey,
				ForeignColumns: c.ForeignColumns,
			})

		case "c":
			ret.Checks[key] = append(ret.Checks[key], drivers.Check[any]{
				Constraint: drivers.Constraint[any]{
					Name:    c.Name,
					Columns: c.Columns,
					Comment: c.Comment.String,
				},
				Expression: c.CheckExpr.String,
			})
		}
	}

	return ret, nil
}
