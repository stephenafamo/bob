package gen

import (
	"testing"

	"github.com/stephenafamo/bob/gen/drivers"
	"github.com/stephenafamo/bob/internal"
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
				DBType: internal.Pointer("SERIAL"),
			},
			Replace: "excellent.Type",
		},
		{
			Tables: []string{"named_table"},
			Match: ColumnFilter{
				Name: internal.Pointer("id"),
			},
			Replace: "excellent.NamedType",
		},
		{
			Match: ColumnFilter{
				Type:     internal.Pointer("null.String"),
				Nullable: internal.Pointer(true),
			},
			Replace: "int",
		},
		{
			Match: ColumnFilter{
				DomainName: internal.Pointer("EMAIL"),
			},
			Replace: "contextInt",
		},
		{
			Match: ColumnFilter{
				Name: internal.Pointer("by_named"),
			},
			Replace: "big.Int",
		},
		{
			Match: ColumnFilter{
				Comment: internal.Pointer("xid"),
			},
			Replace: "xid.ID",
		},
		{
			Match: ColumnFilter{
				Name: internal.Pointer("/_id$/"),
			},
			Replace: "fk.ID",
		},
		{
			Match: ColumnFilter{
				Name:     internal.Pointer("id"),
				AutoIncr: internal.Pointer(true),
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
