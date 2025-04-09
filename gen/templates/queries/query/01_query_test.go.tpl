{{if .QueryFile.Queries}}

{{$.Importer.Import "testing"}}
{{$.Importer.Import "context"}}
{{$.Importer.Import "strings"}}
{{$.Importer.Import "github.com/google/go-cmp/cmp"}}

{{range $query := $.QueryFile.Queries}}
{{$args := list }}
{{range $arg := $query.Args -}}
  {{ $argName := camelCase $arg.Col.Name }}
  {{ $argType := ($arg.Type $.Importer $.Types) }}
  {{- if $arg.CanBeMultiple -}}
    {{$args = append $args (printf "%s{zero[%s]()}" $argType (substr 2 (len $argType) $argType)) }}
  {{- else -}}
    {{$args = append $args (printf "zero[%s]()" $argType) }}
  {{- end -}}
{{end}}

{{$upperName := title $query.Name}}
{{$lowerName := untitle $query.Name}}
{{$queryType := (lower $query.Type.String | titleCase)}}

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

	queryDiff, err := testutils.QueryDiff({{$lowerName}}SQL, sb.String(), nil)
  if err != nil {
    t.Fatal(err)
  }
  if queryDiff != "" {
    fmt.Println(sb.String())
		t.Fatalf("unexpected result (-got +want):\n%s", queryDiff)
	}
}


{{if $query.Columns}}
{{$.Importer.Import "os"}}
{{$.Importer.Import "slices"}}
{{$.Importer.Import "database/sql"}}
{{$.Importer.Import "github.com/stephenafamo/bob"}}
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

{{ $sqlDriverName := "" }}
{{ $dsnEnvVarName := "" }}
{{ if eq $.DriverName "github.com/go-sql-driver/mysql" }}
	{{$.Importer.Import "_" $.DriverName }}
	{{$sqlDriverName = "mysql"}}
	{{$dsnEnvVarName = "MYSQL_TEST_DSN"}}
{{ else if eq $.DriverName "github.com/lib/pq" }}
	{{$.Importer.Import "_" $.DriverName }}
	{{$sqlDriverName = "postgres"}}
	{{$dsnEnvVarName = "PSQL_TEST_DSN"}}
{{ else if eq $.DriverName "github.com/jackc/pgx" }}
	{{$.Importer.Import "_" (printf "%s/stdlib" $.DriverName) }}
	{{$sqlDriverName = "pgx"}}
	{{$dsnEnvVarName = "PSQL_TEST_DSN"}}
{{ else if eq $.DriverName "github.com/jackc/pgx/v4" }}
	{{$.Importer.Import "_" (printf "%s/stdlib" $.DriverName) }}
	{{$sqlDriverName = "pgx"}}
	{{$dsnEnvVarName = "PSQL_TEST_DSN"}}
{{ else if eq $.DriverName "github.com/jackc/pgx/v5" }}
	{{$.Importer.Import "_" (printf "%s/stdlib" $.DriverName) }}
	{{$sqlDriverName = "pgx"}}
	{{$dsnEnvVarName = "PSQL_TEST_DSN"}}
{{ else if eq $.DriverName "modernc.org/sqlite" }}
	{{$.Importer.Import "_" $.DriverName }}
	{{$sqlDriverName = "sqlite"}}
	{{$dsnEnvVarName = "SQLITE_TEST_DSN"}}
{{ else if eq $.DriverName  "github.com/mattn/go-sqlite3" }}
	{{$.Importer.Import "_" $.DriverName }}
	{{$sqlDriverName = "sqlite3"}}
	{{$dsnEnvVarName = "SQLITE_TEST_DSN"}}
{{ end }}
func Test{{$upperName}}Scan (t *testing.T) {
  dsn := os.Getenv("{{$dsnEnvVarName}}")
  if dsn == "" {
    t.Skip("skipping test, no DSN provided")
  }

	db, err := sql.Open("{{$sqlDriverName}}", dsn)
	if err != nil {
		t.Fatal("Error connecting to database")
	}

	query, args, err := bob.Build(context.Background(), {{$upperName}}({{join ", " $args}}))
  if err != nil {
    t.Fatal(err)
  }

  rows, err := db.Query(query, args...)
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
