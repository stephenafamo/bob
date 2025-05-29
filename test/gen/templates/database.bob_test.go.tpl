{{ $sqlDriver := "" }}
{{ $dsnEnvVarName := "" }}
{{ if eq $.Driver "github.com/go-sql-driver/mysql" }}
	{{$.Importer.Import "_" $.Driver }}
	{{$sqlDriver = "mysql"}}
	{{$dsnEnvVarName = "MYSQL_TEST_DSN"}}
{{ else if eq $.Driver "github.com/lib/pq" }}
	{{$.Importer.Import "_" $.Driver }}
	{{$sqlDriver = "postgres"}}
	{{$dsnEnvVarName = "PSQL_TEST_DSN"}}
{{ else if eq $.Driver "github.com/jackc/pgx/v5" }}
	{{$dsnEnvVarName = "PSQL_TEST_DSN"}}
{{ else if eq $.Driver "github.com/jackc/pgx/v5/stdlib" }}
  {{$.Importer.Import "_" $.Driver }}
	{{$sqlDriver = "pgx"}}
	{{$dsnEnvVarName = "PSQL_TEST_DSN"}}
{{ else if eq $.Driver "modernc.org/sqlite" }}
	{{$.Importer.Import $.Driver }}
  {{$.Importer.Import "context"}}
  {{$.Importer.Import "strings"}}
	{{$sqlDriver = "sqlite"}}
	{{$dsnEnvVarName = "SQLITE_TEST_DSN"}}
{{ else if eq $.Driver "github.com/mattn/go-sqlite3" }}
	{{$.Importer.Import $.Driver }}
  {{$.Importer.Import "strings"}}
	{{$sqlDriver = "sqlite3_extended"}}
	{{$dsnEnvVarName = "SQLITE_TEST_DSN"}}
{{ else if eq $.Driver "github.com/ncruces/go-sqlite3" }}
	{{$.Importer.Import $.Driver }}
  {{$.Importer.Import "_" "github.com/ncruces/go-sqlite3/driver" }}
  {{$.Importer.Import "_" "github.com/ncruces/go-sqlite3/embed" }}
	{{$sqlDriver = "sqlite3"}}
	{{$dsnEnvVarName = "SQLITE_TEST_DSN"}}
{{ else if eq $.Driver  "github.com/tursodatabase/libsql-client-go/libsql" }}
	{{$.Importer.Import "_" $.Driver }}
	{{$sqlDriver = "libsql"}}
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

  {{if eq $.Driver "modernc.org/sqlite" }}
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
  {{ else if eq $.Driver  "github.com/mattn/go-sqlite3" }}
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
  {{ else if eq $.Driver  "github.com/ncruces/go-sqlite3" }}
    sqlite3.AutoExtension(func(c *sqlite3.Conn) error {
      queries := os.Getenv("BOB_SQLITE_ATTACH_QUERIES")
      return c.Exec(queries)
    })
  {{end}}

  {{if eq $.Driver "github.com/jackc/pgx/v5"}}
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
    testDB, err = bob.Open("{{$sqlDriver}}", dsn)
    if err != nil {
      log.Fatalf("failed to open database connection: %v", err)
    }
  {{end}}

	os.Exit(m.Run())
}

