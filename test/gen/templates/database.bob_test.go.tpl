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
{{ else if eq $.DriverName  "github.com/tursodatabase/libsql-client-go/libsql" }}
	{{$.Importer.Import "_" $.DriverName }}
	{{$sqlDriverName = "libsql"}}
	{{$dsnEnvVarName = "LIBSQL_TEST_DSN"}}
{{ end }}


{{$.Importer.Import "os"}}
{{$.Importer.Import "log"}}
{{$.Importer.Import "strings"}}
{{$.Importer.Import "context"}}
{{$.Importer.Import "testing"}}
{{$.Importer.Import "database/sql"}}
{{$.Importer.Import "github.com/stephenafamo/bob"}}
func TestMain(m *testing.M) {
  dsn := os.Getenv("{{$dsnEnvVarName}}")
  if dsn == "" {
    log.Fatal(`missing environment variable "{{$dsnEnvVarName}}" `)
  }

	db, err := sql.Open("{{$sqlDriverName}}", dsn)
	if err != nil {
    log.Fatalf("failed to open database connection: %v", err)
	}

  testDB = bob.NewDB(db)

  queries := os.Getenv("BOB_DB_SETUP_QUERIES")
  for _, query := range strings.Split(queries, ";") {
    if query == "" {
      continue
    }
    _, err := testDB.ExecContext(context.Background(), query)
    if err != nil {
      log.Fatalf("failed to execute query %q: %v", query, err)
    }
  }

	os.Exit(m.Run())
}

