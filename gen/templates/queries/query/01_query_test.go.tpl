{{if .QueryFile.Queries}}

{{$.Importer.Import "testing"}}
{{$.Importer.Import "context"}}
{{$.Importer.Import "strings"}}
{{$.Importer.Import "github.com/stephenafamo/bob"}}
{{$.Importer.Import "github.com/google/go-cmp/cmp"}}

{{range $query := $.QueryFile.Queries}}
{{$upperName := title $query.Name}}
{{$lowerName := untitle $query.Name}}
{{$queryType := (lower $query.Type.String | titleCase)}}
{{$args := list }}
{{range $arg := $query.Args -}}
  {{$args = append $args ($arg.RandomExpr $.Importer $.Types) }}
{{end}}


func Test{{$upperName}} (t *testing.T) {
  var sb strings.Builder

	query := {{$upperName}}({{join ", " $args}})

	if _, err := query.WriteQuery(context.Background(), &sb, 1); err != nil {
		t.Fatal(err)
	}

	if diff := cmp.Diff({{$lowerName}}SQL, sb.String()); diff != "" {
		t.Fatalf("unexpected result (-got +want):\n%s", diff)
	}
}

{{$.Importer.Import "fmt"}}
{{$.Importer.Import "testutils" "github.com/stephenafamo/bob/test/utils"}}
{{$.Importer.Import (printf "github.com/stephenafamo/bob/dialect/%s" $.Dialect)}}
func Test{{$upperName}}Mod (t *testing.T) {
  var sb strings.Builder

	query := {{$upperName}}({{join ", " $args}})

	if _, err := {{$.Dialect}}.{{$queryType}}(query).WriteQuery(context.Background(), &sb, 1); err != nil {
		t.Fatal(err)
	}

	queryDiff, err := testutils.QueryDiff({{$lowerName}}SQL, sb.String(), formatQuery)
  if err != nil {
    t.Fatal(err)
  }
  if queryDiff != "" {
    fmt.Println(sb.String())
		t.Fatalf("unexpected result (-got +want):\n%s", queryDiff)
	}
}

{{if not $query.Columns}}
func Test{{$upperName}}Exec (t *testing.T) {
  if testDB == nil {
    t.Skip("skipping test, no DSN provided")
  }

  ctxTx, cancel := context.WithCancel(context.Background())
  defer cancel()

  tx, err := testDB.Begin(ctxTx)
  if err != nil {
    t.Fatalf("Error starting transaction: %v", err)
  }

  defer func() {
    if err := tx.Rollback(ctxTx); err != nil {
      t.Fatalf("Error rolling back transaction: %v", err)
    }
  }()

	query := {{$.Dialect}}.{{$queryType}}({{$upperName}}({{join ", " $args}}))
  if _, err := bob.Exec(ctxTx, tx, query); err != nil {
    t.Fatal(err)
  }
}
{{end}}


{{if $query.Columns}}
{{$.Importer.Import "slices"}}
{{$.Importer.Import "github.com/stephenafamo/scan"}}
func Test{{$upperName}}Map (t *testing.T) {
  {{$queryRowName := $query.Config.RowName}}
  {{if not $query.Config.GenerateRow}}
    {{- $typDef :=  index $.Types $queryRowName -}}
    {{- $queryRowName = or $typDef.AliasOf $queryRowName -}}
    {{- $.Importer.ImportList $typDef.Imports -}}
  {{end}}

  mapCols, err := scan.StructMapperColumns[{{$queryRowName}}]()
  if err != nil {
    t.Fatal(err)
  }

  {{range $index, $col := $query.Columns}}
  if !slices.Contains(mapCols, "{{$col.DBName}}") {
    t.Errorf("Return type %q does not contain column %q", "{{$queryRowName}}", "{{$col.DBName}}")
  }
  {{end}}
}

func Test{{$upperName}}Scan (t *testing.T) {
  if testDB == nil {
    t.Skip("skipping test, no DSN provided")
  }

  ctxTx, cancel := context.WithCancel(context.Background())
  defer cancel()

  tx, err := testDB.Begin(ctxTx)
  if err != nil {
    t.Fatalf("Error starting transaction: %v", err)
  }

  defer func() {
    if err := tx.Rollback(ctxTx); err != nil {
      t.Fatalf("Error rolling back transaction: %v", err)
    }
  }()

	query, args, err := bob.Build(ctxTx, {{$.Dialect}}.{{$queryType}}({{$upperName}}({{join ", " $args}})))
  if err != nil {
    t.Fatal(err)
  }

  rows, err := tx.QueryContext(ctxTx, query, args...)
  if err != nil {
    t.Fatal(err)
  }
  defer rows.Close()

  columns, err := rows.Columns()
  if err != nil {
    t.Fatal(err)
  }

  if len(columns) != {{len $query.Columns}} {
    t.Fatalf("expected %d columns, got %d", {{len $query.Columns}}, len(columns))
  }

  {{range $index, $col := $query.Columns}}
  if columns[{{$index}}] != "{{$col.DBName}}" {
    t.Fatalf("expected column %d to be %s, got %s", {{$index}}, "{{$col.DBName}}", columns[{{$index}}])
  }
  {{end}}
}
{{end}}

{{end}}
{{end}}
