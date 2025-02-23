{{- if and .Table.Constraints.Uniques (not $.NoFactory)}}
{{$table := .Table}}
{{$tAlias := .Aliases.Table $table.Key}}
{{$factoryPackage := printf "%s/factory" $.ModelsPackage }}
{{$.Importer.Import "factory" $factoryPackage }}
{{$.Importer.Import "models" $.ModelsPackage}}
{{$.Importer.Import "context"}}
{{$.Importer.Import "database/sql"}}
{{$.Importer.Import "errors"}}
{{$.Importer.Import "os"}}
{{$.Importer.Import "testing"}}
{{$.Importer.Import "github.com/stephenafamo/bob"}}

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
{{ else if eq $.DriverName "github.com/tursodatabase/libsql-client-go/libsql" }}
	{{$.Importer.Import "_" $.DriverName }}
	{{$sqlDriverName = "libsql"}}
	{{$dsnEnvVarName = "SQLITE_TEST_DSN"}}
{{ end }}

func Test{{$tAlias.UpSingular}}UniqueConstraintErrors(t *testing.T) {
	db, err := sql.Open("{{$sqlDriverName}}", os.Getenv("{{$dsnEnvVarName}}"))
	if err != nil {
		t.Fatal("Error connecting to database")
	}
	tests := []struct{
		name        string
		expectedErr *models.UniqueConstraintError
		applyFn     func(tpl *factory.{{$tAlias.UpSingular}}Template, obj *models.{{$tAlias.UpSingular}})
	}{
	{{range $index := $table.Indexes}}
		{{ if $index.Unique }}
		{{- $errName := printf "ErrUnique%s" ($index.Name | camelcase) -}}
		{
			name: "{{$errName}}",
			expectedErr: models.{{$tAlias.UpSingular}}Errors.{{$errName}},
			applyFn: func(tpl *factory.{{$tAlias.UpSingular}}Template, obj *models.{{$tAlias.UpSingular}}) {
				tpl.Apply(
					factory.{{$tAlias.UpSingular}}Mods.RandomizeAllColumns(nil),
					{{range $indexColumn := $index.Columns}}
					{{- $colAlias := $tAlias.Column $indexColumn.Name -}}
					factory.{{$tAlias.UpSingular}}Mods.{{$colAlias}}(obj.{{$colAlias}}),
					{{end}}
				)
			},
		},
		{{ end }}
	{{end}}
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			tx, err := db.Begin()
			if err != nil {
				t.Fatal("Couldn't start database transaction")
			}
			exec := bob.New(tx)
			f := factory.New()
			tpl := f.New{{$tAlias.UpSingular}}(factory.{{$tAlias.UpSingular}}Mods.RandomizeAllColumns(nil))
			obj, err := tpl.Create(ctx, exec)
			if err != nil {
				t.Fatal(err)
			}
			tt.applyFn(tpl, obj)
			_, err = models.{{$tAlias.UpPlural}}.Insert(tpl.BuildSetter()).One(ctx, exec)
			if !errors.Is(models.ErrUniqueConstraint, err) {
				t.Fatalf("Expected: %s, Got: %v", tt.name, err)
			}
			if !errors.Is(tt.expectedErr, err) {
				t.Fatalf("Expected: %s, Got: %v", tt.name, err)
			}
			if !models.ErrUniqueConstraint.Is(err) {
				t.Fatalf("Expected: %s, Got: %v", tt.name, err)
			}
			if !tt.expectedErr.Is(err) {
				t.Fatalf("Expected: %s, Got: %v", tt.name, err)
			}
			if err = tx.Rollback(); err != nil {
				t.Fatal("Couldn't rollback database transaction")
			}
		})
	}
}
{{end -}}
