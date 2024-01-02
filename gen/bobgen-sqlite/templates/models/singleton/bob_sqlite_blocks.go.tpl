{{define "helpers/join_variables" -}}
var (
	SelectJoins = getJoins[*dialect.SelectQuery]
	UpdateJoins = getJoins[*dialect.UpdateQuery]
)
{{- end}}

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
{{- end}}

