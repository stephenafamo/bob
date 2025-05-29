{{- define "helpers/join_variables"}}
var (
	SelectJoins = getJoins[*dialect.SelectQuery]
	UpdateJoins = getJoins[*dialect.UpdateQuery]
)
{{end -}}

{{define "unique_constraint_error_detection_method" -}}
{{$.Importer.Import "strings"}}
{{$.Importer.Import "fmt"}}
func (e *UniqueConstraintError) Is(target error) bool {
	{{if eq $.Driver "github.com/tursodatabase/libsql-client-go/libsql"}}
    fullCols := make([]string, len(e.columns))
    for i, col := range e.columns {
      fullCols[i] = fmt.Sprintf("%s.%s", e.table, col)
    }
    return strings.Contains(
      target.Error(),
      fmt.Sprintf("SQLite error: UNIQUE constraint failed: %s", strings.Join(fullCols, ", ")),
    )
	{{else if eq $.Driver "modernc.org/sqlite" "github.com/mattn/go-sqlite3" "github.com/ncruces/go-sqlite3"}}
		{{$errType := ""}}
		{{$codeGetter := ""}}
		{{$.Importer.Import "strings"}}
		{{if eq $.Driver "modernc.org/sqlite"}}
			{{$.Importer.Import "sqliteDriver" $.Driver}}
			{{$errType = "*sqliteDriver.Error"}}
			{{$codeGetter = "Code()"}}
		{{else if eq $.Driver "github.com/mattn/go-sqlite3"}}
			{{$.Importer.Import $.Driver}}
			{{$errType = "sqlite3.Error"}}
			{{$codeGetter = "ExtendedCode"}}
		{{else if eq $.Driver "github.com/ncruces/go-sqlite3"}}
			{{$.Importer.Import $.Driver}}
			{{$errType = "*sqlite3.Error"}}
			{{$codeGetter = "ExtendedCode()"}}
		{{else}}
			panic("Unsupported driver {{$.Driver}} for UniqueConstraintError detection")
		{{end}}
    var err {{$errType}}
    if !errors.As(target, &err) {
      return false
    }

    // 1555 is for Primary Key Constraint
    // 2067 is for Unique Constraint
    if err.{{$codeGetter}} != 1555 && err.{{$codeGetter}} != 2067  {
      return false
    }

    for _, col := range e.columns {
      if !strings.Contains(err.Error(), fmt.Sprintf("%s.%s", e.table, col)) {
        return false
      }
    }

    return true
    {{else}}
    return false
	{{end}}
}
{{end -}}
