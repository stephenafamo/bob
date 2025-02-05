{{- define "helpers/join_variables"}}
var (
	SelectJoins = getJoins[*dialect.SelectQuery]
	UpdateJoins = getJoins[*dialect.UpdateQuery]
)
{{end -}}

{{define "unique_constraint_error_detection_method" -}}
func (e *UniqueConstraintError) Is(target error) bool {
	{{if eq $.DriverName "github.com/tursodatabase/libsql-client-go/libsql"}}
		{{$.Importer.Import "strings"}}
		{{$.Importer.Import "fmt"}}
	return strings.Contains(target.Error(), fmt.Sprintf("SQLite error: UNIQUE constraint failed: %s", e.s))
	{{else if eq $.DriverName "modernc.org/sqlite" "github.com/mattn/go-sqlite3"}}
		{{$errType := ""}}
		{{$codeGetter := ""}}
		{{$.Importer.Import "strings"}}
		{{if eq $.DriverName "modernc.org/sqlite"}}
			{{$.Importer.Import "sqliteDriver" $.DriverName}}
			{{$errType = "*sqliteDriver.Error"}}
			{{$codeGetter = "Code()"}}
		{{else}}
			{{$.Importer.Import $.DriverName}}
			{{$errType = "sqlite3.Error"}}
			{{$codeGetter = "ExtendedCode"}}
		{{end}}
	err, ok := target.({{$errType}})
	if !ok {
		return false
	}
	return err.{{$codeGetter}} == 2067 && strings.Contains(err.Error(), e.s)
	{{else}}
	return false
	{{end}}
}
{{end -}}
