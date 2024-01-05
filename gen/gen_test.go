package gen

import (
	"testing"

	"github.com/stephenafamo/bob/gen/drivers"
)

func TestProcessTypeReplacements(t *testing.T) {
	tables := []drivers.Table{
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
					DBType:   "serial",
					Default:  "some db nonsense",
					Nullable: true,
				},
				{
					Name:       "domain",
					Type:       "string",
					DBType:     "text",
					DomainName: "domain name",
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
	}

	types := map[string]Type{
		"excellent.Type": {
			Name:    "excellent.Type",
			Imports: []string{`"rock.com/excellent"`},
		},
		"excellent.NamedType": {
			Name:    "excellent.NamedType",
			Imports: []string{`"rock.com/excellent-name"`},
		},
		"int": {
			Name:    "int",
			Imports: []string{`"context"`},
		},
		"contextInt": {
			Name:    "contextInt",
			Imports: []string{`"contextual"`},
		},
		"big.Int": {
			Name:    "big.Int",
			Imports: []string{`"math/big"`},
		},
		"xid.ID": {
			Name:    "xid.ID",
			Imports: []string{`"github.com/rs/xid"`},
		},
	}

	replacements := []Replace{
		{
			Match: drivers.Column{
				DBType: "serial",
			},
			Replace: "excellent.Type",
		},
		{
			Tables: []string{"named_table"},
			Match: drivers.Column{
				DBType: "serial",
			},
			Replace: "excellent.NamedType",
		},
		{
			Match: drivers.Column{
				Type:     "null.String",
				Nullable: true,
			},
			Replace: "int",
		},
		{
			Match: drivers.Column{
				DomainName: "domain name",
			},
			Replace: "contextInt",
		},
		{
			Match: drivers.Column{
				Name: "by_named",
			},
			Replace: "big.Int",
		},
		{
			Match: drivers.Column{
				Comment: "xid",
			},
			Replace: "xid.ID",
		},
	}

	processTypeReplacements(types, replacements, tables)

	if typ := tables[0].Columns[0].Type; typ != "excellent.Type" {
		t.Error("type was wrong:", typ)
	}
	if i := tables[0].Columns[0].Imports[0]; i != `"rock.com/excellent"` {
		t.Error("imports were not adjusted")
	}

	if typ := tables[0].Columns[1].Type; typ != "int" {
		t.Error("type was wrong:", typ)
	}
	if i := tables[0].Columns[1].Imports[0]; i != `"context"` {
		t.Error("imports were not adjusted")
	}

	if typ := tables[0].Columns[2].Type; typ != "contextInt" {
		t.Error("type was wrong:", typ)
	}
	if i := tables[0].Columns[2].Imports[0]; i != `"contextual"` {
		t.Error("imports were not adjusted")
	}

	if typ := tables[0].Columns[3].Type; typ != "big.Int" {
		t.Error("type was wrong:", typ)
	}
	if i := tables[0].Columns[3].Imports[0]; i != `"math/big"` {
		t.Error("imports were not adjusted")
	}
	if typ := tables[0].Columns[4].Type; typ != "xid.ID" {
		t.Error("type was wrong:", typ)
	}
	if i := tables[0].Columns[4].Imports[0]; i != `"github.com/rs/xid"` {
		t.Error("imports were not adjusted")
	}

	if typ := tables[1].Columns[0].Type; typ != "excellent.NamedType" {
		t.Error("type was wrong:", typ)
	}
	if i := tables[1].Columns[0].Imports[0]; i != `"rock.com/excellent-name"` {
		t.Error("imports were not adjusted")
	}
	if typ := tables[1].Columns[1].Type; typ != "xid.ID" {
		t.Error("type was wrong:", typ)
	}
	if i := tables[1].Columns[1].Imports[0]; i != `"github.com/rs/xid"` {
		t.Error("imports were not adjusted")
	}
}
