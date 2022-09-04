{{$table := .Table}}
{{$tAlias := .Aliases.Table $table.Name -}}

{{$.Importer.Import (printf "github.com/stephenafamo/bob/dialect/%s/model" $.Dialect)}}
{{if not $table.PKey -}}
	var {{$tAlias.UpPlural}}View = model.NewView[*{{$tAlias.UpSingular}}, {{$tAlias.UpSingular}}Slice]({{quoteAndJoin .Schema $table.Name}})
{{- else -}}
var {{$tAlias.UpPlural}}Table = model.NewTable[*{{$tAlias.UpSingular}}, {{$tAlias.UpSingular}}Slice, *Optional{{$tAlias.UpSingular}}]({{quoteAndJoin .Schema $table.Name}})
{{- end}}

{{$.Importer.Import (printf "github.com/stephenafamo/bob/dialect/%s" $.Dialect)}}
var {{$tAlias.UpSingular}}Columns = struct {
	{{range $column := $table.Columns -}}
	{{- $colAlias := $tAlias.Column $column.Name -}}
	{{$colAlias}} {{$.Dialect}}.Expression
	{{end -}}
}{
	{{range $column := $table.Columns -}}
	{{- $colAlias := $tAlias.Column $column.Name -}}
	{{$colAlias}}: {{$.Dialect}}.Quote("{{$table.Name}}", "{{$column.Name}}"),
	{{end -}}
}

