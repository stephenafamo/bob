{{- define "helpers/join_variables"}}
var (
	SelectJoins = getJoins[*dialect.SelectQuery]
	UpdateJoins = getJoins[*dialect.UpdateQuery]
)
{{end -}}

{{define "unique_constraint_error_detection_method" -}}
func (e *UniqueConstraintError) Is(target error) bool {
	{{if not (eq $.DriverName "modernc.org/sqlite" "github.com/mattn/go-sqlite3")}}
	return false
	{{else}}
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
	{{end}}
}
{{end -}}
