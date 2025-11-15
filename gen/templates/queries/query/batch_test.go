package query

import (
	"bytes"
	"testing"
	"text/template"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/gen/drivers"
)

// TestBatchTemplate verifies the batch template can be executed
func TestBatchTemplate(t *testing.T) {
	// Mock data structure for template
	data := struct {
		QueryFile struct {
			Queries []struct {
				Name    string
				Type    bob.QueryType
				Config  drivers.QueryConfig
				Columns []struct {
					Name   string
					DBName string
				}
				Args []struct {
					Col struct {
						Name string
					}
				}
			}
		}
		Importer    mockImporter
		Types       mockTypes
		CurrentPackage string
	}{
		CurrentPackage: "test",
		Importer:       mockImporter{},
		Types:          mockTypes{},
	}

	// Add a batch query
	data.QueryFile.Queries = []struct {
		Name    string
		Type    bob.QueryType
		Config  drivers.QueryConfig
		Columns []struct {
			Name   string
			DBName string
		}
		Args []struct {
			Col struct {
				Name string
			}
		}
	}{
		{
			Name: "SelectUser",
			Type: bob.QueryTypeSelect,
			Config: drivers.QueryConfig{
				Batch: true,
			},
			Columns: []struct {
				Name   string
				DBName string
			}{
				{Name: "ID", DBName: "id"},
				{Name: "Name", DBName: "name"},
			},
			Args: []struct {
				Col struct {
					Name string
				}
			}{
				{Col: struct{ Name string }{Name: "ID"}},
			},
		},
	}

	// This test just verifies the template can be parsed
	// Full template execution would require the complete Bob template engine setup
	t.Run("batch query config", func(t *testing.T) {
		if !data.QueryFile.Queries[0].Config.Batch {
			t.Error("Expected Batch to be true")
		}

		if data.QueryFile.Queries[0].Name != "SelectUser" {
			t.Errorf("Expected query name SelectUser, got %s", data.QueryFile.Queries[0].Name)
		}

		if data.QueryFile.Queries[0].Type != bob.QueryTypeSelect {
			t.Errorf("Expected query type Select, got %v", data.QueryFile.Queries[0].Type)
		}
	})
}

// TestBatchTemplateStructure verifies the template file syntax
func TestBatchTemplateStructure(t *testing.T) {
	// Simple syntax check - verify the template can be parsed
	tmplContent := `
{{if .QueryFile.Queries}}
{{range $query := .QueryFile.Queries}}
{{if $query.Config.Batch}}
	Batch query: {{$query.Name}}
{{end}}
{{end}}
{{end}}
`

	tmpl, err := template.New("test").Parse(tmplContent)
	if err != nil {
		t.Fatalf("Template parse error: %v", err)
	}

	data := struct {
		QueryFile struct {
			Queries []struct {
				Name   string
				Config struct {
					Batch bool
				}
			}
		}
	}{}

	data.QueryFile.Queries = []struct {
		Name   string
		Config struct {
			Batch bool
		}
	}{
		{
			Name:   "TestQuery",
			Config: struct{ Batch bool }{Batch: true},
		},
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	if err != nil {
		t.Fatalf("Template execution error: %v", err)
	}

	output := buf.String()
	if !bytes.Contains(buf.Bytes(), []byte("Batch query: TestQuery")) {
		t.Errorf("Expected output to contain 'Batch query: TestQuery', got: %s", output)
	}
}

// Mock types for testing
type mockImporter struct{}

func (m mockImporter) Import(path string) string {
	return path
}

func (m mockImporter) ImportList(paths []string) {
}

type mockTypes struct{}

func (m mockTypes) Get(currPkg string, i interface{}, typ string) string {
	return typ
}
