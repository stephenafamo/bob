{{- define "helpers/join_variables"}}
var (
	SelectJoins = getJoins[*dialect.SelectQuery]
	UpdateJoins = getJoins[*dialect.UpdateQuery]
)
{{end -}}

{{define "setter_insert_mod" -}}
{{$.Importer.Import (printf "github.com/stephenafamo/bob/dialect/%s/im" $.Dialect)}}
{{$table := .Table}}
{{$tAlias := .Aliases.Table $table.Key -}}
func (s {{$tAlias.UpSingular}}Setter) InsertMod() bob.Mod[*dialect.InsertQuery] {
  vals := make([]bob.Expression, 0, {{len $table.NonGeneratedColumns}})
	{{range $column := $table.NonGeneratedColumns -}}
		{{$colAlias := $tAlias.Column $column.Name -}}
		if !s.{{$colAlias}}.IsUnset() {
			vals = append(vals, {{$.Dialect}}.Arg(s.{{$colAlias}}))
		}

	{{end}}

	return im.Values(vals...)
}
{{end -}}

{{define "unique_constraint_error_detection_method" -}}
func (e *errUniqueConstraint) Is(target error) bool {
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
