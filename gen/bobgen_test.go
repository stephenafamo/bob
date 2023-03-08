package gen

import (
	"testing"

	"github.com/stephenafamo/bob/gen/drivers"
)

func TestProcessTypeReplacements(t *testing.T) {
	s := new(State[any])
	s.Config = Config{}
	s.tables = []drivers.Table{
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
			},
		},
	}

	s.Config.Replacements = []Replace{
		{
			Match: drivers.Column{
				DBType: "serial",
			},
			Replace: drivers.Column{
				Type:    "excellent.Type",
				Imports: []string{`"rock.com/excellent"`},
			},
		},
		{
			Tables: []string{"named_table"},
			Match: drivers.Column{
				DBType: "serial",
			},
			Replace: drivers.Column{
				Type:    "excellent.NamedType",
				Imports: []string{`"rock.com/excellent-name"`},
			},
		},
		{
			Match: drivers.Column{
				Type:     "null.String",
				Nullable: true,
			},
			Replace: drivers.Column{
				Type:    "int",
				Imports: []string{`"context"`},
			},
		},
		{
			Match: drivers.Column{
				DomainName: "domain name",
			},
			Replace: drivers.Column{
				Type:    "contextInt",
				Imports: []string{`"contextual"`},
			},
		},
		{
			Match: drivers.Column{
				Name: "by_named",
			},
			Replace: drivers.Column{
				Type:    "big.Int",
				Imports: []string{`"math/big"`},
			},
		},
	}

	s.processTypeReplacements()

	if typ := s.tables[0].Columns[0].Type; typ != "excellent.Type" {
		t.Error("type was wrong:", typ)
	}
	if i := s.tables[0].Columns[0].Imports[0]; i != `"rock.com/excellent"` {
		t.Error("imports were not adjusted")
	}

	if typ := s.tables[0].Columns[1].Type; typ != "int" {
		t.Error("type was wrong:", typ)
	}
	if i := s.tables[0].Columns[1].Imports[0]; i != `"context"` {
		t.Error("imports were not adjusted")
	}

	if typ := s.tables[0].Columns[2].Type; typ != "contextInt" {
		t.Error("type was wrong:", typ)
	}
	if i := s.tables[0].Columns[2].Imports[0]; i != `"contextual"` {
		t.Error("imports were not adjusted")
	}

	if typ := s.tables[0].Columns[3].Type; typ != "big.Int" {
		t.Error("type was wrong:", typ)
	}
	if i := s.tables[0].Columns[3].Imports[0]; i != `"math/big"` {
		t.Error("imports were not adjusted")
	}

	if typ := s.tables[1].Columns[0].Type; typ != "excellent.NamedType" {
		t.Error("type was wrong:", typ)
	}
	if i := s.tables[1].Columns[0].Imports[0]; i != `"rock.com/excellent-name"` {
		t.Error("imports were not adjusted")
	}
}
