package gen

import (
	"testing"

	"github.com/aarondl/opt/omit"
	"github.com/stephenafamo/bob/gen/drivers"
)

func TestProcessTypeReplacements(t *testing.T) {
	tables := drivers.Tables[any, any]{
		{
			Columns: []drivers.Column{
				{
					Name:     "id",
					Type:     "int",
					DBType:   "serial",
					Default:  "some db nonsense",
					Nullable: false,
				},
				{
					Name:     "name",
					Type:     "null.String",
					DBType:   "text",
					Default:  "some db nonsense",
					Nullable: true,
				},
				{
					Name:       "user_email",
					Type:       "string",
					DBType:     "text",
					DomainName: "email",
				},
				{
					Name:     "by_named",
					Type:     "int",
					DBType:   "numeric",
					Default:  "some db nonsense",
					Nullable: false,
				},
				{
					Name:     "by_comment",
					Type:     "string",
					DBType:   "text",
					Default:  "some db nonsense",
					Nullable: false,
					Comment:  "xid",
				},
				{
					Name:     "author_id",
					Type:     "int",
					DBType:   "integer",
					Nullable: true,
				},
			},
		},
		{
			Key: "named_table",
			Columns: []drivers.Column{
				{
					Name:     "id",
					Type:     "int",
					DBType:   "serial",
					Default:  "some db nonsense",
					Nullable: false,
				},
				{
					Name:     "by_comment",
					Type:     "string",
					DBType:   "text",
					Default:  "some db nonsense",
					Nullable: false,
					Comment:  "xid",
				},
			},
		},
		{
			Columns: []drivers.Column{
				{
					Name:     "id",
					Type:     "int",
					DBType:   "serial",
					AutoIncr: true,
					Nullable: false,
				},
			},
		},
	}

	types := drivers.Types{}
	types.RegisterAll(map[string]drivers.Type{
		"excellent.Type": {
			Imports: []string{`"rock.com/excellent"`},
		},
		"excellent.NamedType": {
			Imports: []string{`"rock.com/excellent-name"`},
		},
		"int": {
			Imports: []string{`"context"`},
		},
		"contextInt": {
			Imports: []string{`"contextual"`},
		},
		"big.Int": {
			Imports: []string{`"math/big"`},
		},
		"xid.ID": {
			Imports: []string{`"github.com/rs/xid"`},
		},
		"fk.ID": {
			Imports: []string{`"github.com/fk"`},
		},
		"pk.ID": {
			Imports: []string{`"github.com/pk"`},
		},
	})

	replacements := []Replace{
		{
			Match: ColumnFilter{
				DBType: omit.From("SERIAL"),
			},
			Replace: "excellent.Type",
		},
		{
			Tables: []string{"named_table"},
			Match: ColumnFilter{
				Name: omit.From("id"),
			},
			Replace: "excellent.NamedType",
		},
		{
			Match: ColumnFilter{
				Type:     omit.From("null.String"),
				Nullable: omit.From(true),
			},
			Replace: "int",
		},
		{
			Match: ColumnFilter{
				DomainName: omit.From("EMAIL"),
			},
			Replace: "contextInt",
		},
		{
			Match: ColumnFilter{
				Name: omit.From("by_named"),
			},
			Replace: "big.Int",
		},
		{
			Match: ColumnFilter{
				Comment: omit.From("xid"),
			},
			Replace: "xid.ID",
		},
		{
			Match: ColumnFilter{
				Name: omit.From("/_id$/"),
			},
			Replace: "fk.ID",
		},
		{
			Match: ColumnFilter{
				Name:     omit.From("id"),
				AutoIncr: omit.From(true),
			},
			Replace: "pk.ID",
		},
	}

	processTypeReplacements(types, replacements, tables)

	if typ := tables[0].Columns[0].Type; typ != "excellent.Type" {
		t.Error("type was wrong:", typ)
	}

	if typ := tables[0].Columns[1].Type; typ != "int" {
		t.Error("type was wrong:", typ)
	}

	if typ := tables[0].Columns[2].Type; typ != "contextInt" {
		t.Error("type was wrong:", typ)
	}

	if typ := tables[0].Columns[3].Type; typ != "big.Int" {
		t.Error("type was wrong:", typ)
	}

	if typ := tables[0].Columns[4].Type; typ != "xid.ID" {
		t.Error("type was wrong:", typ)
	}

	if typ := tables[0].Columns[5].Type; typ != "fk.ID" {
		t.Error("type was wrong:", typ)
	}

	if typ := tables[1].Columns[0].Type; typ != "excellent.NamedType" {
		t.Error("type was wrong:", typ)
	}

	if typ := tables[1].Columns[1].Type; typ != "xid.ID" {
		t.Error("type was wrong:", typ)
	}

	if typ := tables[2].Columns[0].Type; typ != "pk.ID" {
		t.Error("type was wrong:", typ)
	}
}
