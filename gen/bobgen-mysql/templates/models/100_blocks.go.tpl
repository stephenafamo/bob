{{define "model_and_query" -}}
{{$table := .Table}}
{{$tAlias := .Aliases.Table $table.Key -}}
{{if not $table.PKey -}}
	// {{$tAlias.UpPlural}} contains methods to work with the {{$table.Name}} view
	var {{$tAlias.UpPlural}} = {{$.Dialect}}.NewViewx[*{{$tAlias.UpSingular}}, {{$tAlias.UpSingular}}Slice]("{{$table.Name}}")
	// {{$tAlias.UpPlural}}Query is a query on the {{$table.Name}} view
	type {{$tAlias.UpPlural}}Query = *{{$.Dialect}}.ViewQuery[*{{$tAlias.UpSingular}}, {{$tAlias.UpSingular}}Slice]
{{- else -}}
	// {{$tAlias.UpPlural}} contains methods to work with the {{$table.Name}} table
	var {{$tAlias.UpPlural}} = {{$.Dialect}}.NewTablex[*{{$tAlias.UpSingular}}, {{$tAlias.UpSingular}}Slice, *{{$tAlias.UpSingular}}Setter]("{{$table.Name}}", {{uniqueColPairs $table}})
	// {{$tAlias.UpPlural}}Query is a query on the {{$table.Name}} table
	type {{$tAlias.UpPlural}}Query = *{{$.Dialect}}.ViewQuery[*{{$tAlias.UpSingular}}, {{$tAlias.UpSingular}}Slice]
{{- end}}
{{- end}}


{{define "setter_update_mod" -}}
{{$table := .Table}}
{{$tAlias := .Aliases.Table $table.Key -}}
func (s {{$tAlias.UpSingular}}Setter) Apply(q *dialect.UpdateQuery) {
	{{$.Importer.Import (printf "github.com/stephenafamo/bob/dialect/%s/um" $.Dialect)}}
	{{- range $column := .Table.Columns -}}
	{{if $column.Generated}}{{continue}}{{end -}}
	{{$colAlias := $tAlias.Column $column.Name -}}
		if !s.{{$colAlias}}.IsUnset() {
			um.Set("{{$table.Name}}", "{{$column.Name}}").ToArg(s.{{$colAlias}}).Apply(q)
		}
	{{end -}}
}
{{- end}}
