{{$table := .Table}}
{{$tAlias := .Aliases.Table $table.Name -}}

{{$.Importer.Import (printf "github.com/stephenafamo/bob/dialect/%s/model" $.Dialect)}}
{{if not $table.PKey -}}
	var {{$tAlias.UpPlural}}View = model.NewView[*{{$tAlias.UpSingular}}, {{$tAlias.UpSingular}}Slice]({{quoteAndJoin .Schema $table.Name}})
	type {{$tAlias.UpPlural}}Query = *model.ViewQuery[*{{$tAlias.UpSingular}}, {{$tAlias.UpSingular}}Slice]
{{- else -}}
var {{$tAlias.UpPlural}}Table = model.NewTable[*{{$tAlias.UpSingular}}, {{$tAlias.UpSingular}}Slice, *Optional{{$tAlias.UpSingular}}]({{quoteAndJoin .Schema $table.Name}})
	type {{$tAlias.UpPlural}}Query = *model.TableQuery[*{{$tAlias.UpSingular}}, {{$tAlias.UpSingular}}Slice, *Optional{{$tAlias.UpSingular}}]
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

