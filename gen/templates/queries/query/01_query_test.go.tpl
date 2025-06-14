{{if .QueryFile.Queries}}

{{$.Importer.Import "fmt"}}
{{$.Importer.Import "testing"}}
{{$.Importer.Import "context"}}
{{$.Importer.Import "strings"}}
{{$.Importer.Import "github.com/google/go-cmp/cmp"}}
{{$.Importer.Import "github.com/stephenafamo/bob"}}
{{$.Importer.Import "testutils" "github.com/stephenafamo/bob/test/utils"}}
{{$.Importer.Import (printf "github.com/stephenafamo/bob/dialect/%s" $.Dialect)}}

{{range $query := $.QueryFile.Queries}}
{{$upperName := title $query.Name}}
{{$lowerName := untitle $query.Name}}
{{$queryType := (lower $query.Type.String | titleCase)}}
{{$args := list }}
{{range $arg := $query.Args -}}
  {{$args = append $args ($arg.RandomExpr $.CurrentPackage $.Importer $.Types) }}
{{end}}


func Test{{$upperName}} (t *testing.T) {
  t.Run("Base", func(t *testing.T) {
    var sb strings.Builder

    query := {{$upperName}}({{join ", " $args}})

    if _, err := query.WriteQuery(context.Background(), &sb, 1); err != nil {
      t.Fatal(err)
    }

    if diff := cmp.Diff({{$lowerName}}SQL, sb.String()); diff != "" {
      t.Fatalf("unexpected result (-got +want):\n%s", diff)
    }

  })

  t.Run("Mod", func(t *testing.T) {
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
  })

  {{if not $query.Columns}}
  t.Run("Exec", func(t *testing.T) {
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
  })
  {{end}}

  {{if and $query.Columns $query.Config.ResultTransformer}}
  {{$.Importer.Import "slices"}}
  {{$.Importer.Import "github.com/stephenafamo/scan"}}
  t.Run("ScanMapping", func(t *testing.T) {
    {{- $queryResultTypeOne := $.Types.Get $.CurrentPackage $.Importer $query.Config.ResultTypeOne -}}

    mapCols, err := scan.StructMapperColumns[{{$queryResultTypeOne}}]()
    if err != nil {
      t.Fatal(err)
    }

    {{range $index, $col := $query.Columns}}
    if !slices.Contains(mapCols, "{{$col.DBName}}") {
      t.Errorf("Return type %q does not contain column %q", "{{$queryResultTypeOne}}", "{{$col.DBName}}")
    }
    {{end}}
  })
  {{end}}

  {{if $query.Columns}}
  t.Run("Scanning", func(t *testing.T) {
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
  })
  {{end}}
}


{{end}}
{{end}}
