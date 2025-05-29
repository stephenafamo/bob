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
{{ else if eq $.DriverName "github.com/jackc/pgx/v5" }}
	{{$dsnEnvVarName = "PSQL_TEST_DSN"}}
{{ else if eq $.DriverName "github.com/jackc/pgx/v5/stdlib" }}
  {{$.Importer.Import "_" $.DriverName }}
	{{$sqlDriverName = "pgx"}}
	{{$dsnEnvVarName = "PSQL_TEST_DSN"}}
{{ else if eq $.DriverName "modernc.org/sqlite" }}
	{{$.Importer.Import $.DriverName }}
  {{$.Importer.Import "context"}}
  {{$.Importer.Import "strings"}}
	{{$sqlDriverName = "sqlite"}}
	{{$dsnEnvVarName = "SQLITE_TEST_DSN"}}
{{ else if eq $.DriverName "github.com/mattn/go-sqlite3" }}
	{{$.Importer.Import $.DriverName }}
  {{$.Importer.Import "strings"}}
	{{$sqlDriverName = "sqlite3_extended"}}
	{{$dsnEnvVarName = "SQLITE_TEST_DSN"}}
{{ else if eq $.DriverName  "github.com/tursodatabase/libsql-client-go/libsql" }}
	{{$.Importer.Import "_" $.DriverName }}
	{{$sqlDriverName = "libsql"}}
	{{$dsnEnvVarName = "LIBSQL_TEST_DSN"}}
{{ end }}


{{$.Importer.Import "os"}}
{{$.Importer.Import "log"}}
{{$.Importer.Import "testing"}}
func TestMain(m *testing.M) {
  dsn := os.Getenv("{{$dsnEnvVarName}}")
  if dsn == "" {
    log.Fatal(`missing environment variable "{{$dsnEnvVarName}}" `)
  }

  {{if eq $.DriverName "modernc.org/sqlite" }}
    sqlite.RegisterConnectionHook(func(conn sqlite.ExecQuerierContext, dsn string) error {
      queries := os.Getenv("BOB_SQLITE_ATTACH_QUERIES")
      for _, query := range strings.Split(queries, ";") {
        if query == "" {
          continue
        }
        if _, err := conn.ExecContext(context.Background(), query, nil); err != nil {
          return err
        }
      }
      return nil
    })
  {{ else if eq $.DriverName  "github.com/mattn/go-sqlite3" }}
    {{$.Importer.Import "database/sql"}}
  	sql.Register("sqlite3_extended", &sqlite3.SQLiteDriver{
      ConnectHook: func(conn *sqlite3.SQLiteConn) error {
        queries := os.Getenv("BOB_SQLITE_ATTACH_QUERIES")
        for _, query := range strings.Split(queries, ";") {
          if query == "" {
            continue
          }
          if _, err := conn.Exec(query, nil); err != nil {
            return err
          }
        }
        return nil
      },
    })
  {{end}}

  {{if eq $.DriverName "github.com/jackc/pgx/v5"}}
    {{$.Importer.Import "context"}}
    {{$.Importer.Import "bobpgx" "github.com/stephenafamo/bob/drivers/pgx"}}
    var err error
    testDB, err = bobpgx.New(context.Background(), dsn)
    if err != nil {
      log.Fatalf("failed to open database connection: %v", err)
    }
  {{else}}
    {{$.Importer.Import "github.com/stephenafamo/bob"}}
    var err error
    testDB, err = bob.Open("{{$sqlDriverName}}", dsn)
    if err != nil {
      log.Fatalf("failed to open database connection: %v", err)
    }
  {{end}}

	os.Exit(m.Run())
}

