{{define "model_and_query" -}}
{{$table := .Table}}
{{$tAlias := .Aliases.Table $table.Key -}}
{{if not $table.PKey -}}
	// {{$tAlias.UpPlural}}View contains methods to work with the {{$table.Name}} view
	var {{$tAlias.UpPlural}}View = {{$.Dialect}}.NewViewx[*{{$tAlias.UpSingular}}, {{$tAlias.UpSingular}}Slice]("{{$table.Name}}")
	// {{$tAlias.UpPlural}}Query is a query on the {{$table.Name}} view
	type {{$tAlias.UpPlural}}Query = *{{$.Dialect}}.ViewQuery[*{{$tAlias.UpSingular}}, {{$tAlias.UpSingular}}Slice]
{{- else -}}
	// {{$tAlias.UpPlural}}Table contains methods to work with the {{$table.Name}} table
	var {{$tAlias.UpPlural}}Table = {{$.Dialect}}.NewTablex[*{{$tAlias.UpSingular}}, {{$tAlias.UpSingular}}Slice, *{{$tAlias.UpSingular}}Setter]("{{$table.Name}}", {{uniqueColPairs $table}})
	// {{$tAlias.UpPlural}}Query is a query on the {{$table.Name}} table
	type {{$tAlias.UpPlural}}Query = *{{$.Dialect}}.TableQuery[*{{$tAlias.UpSingular}}, {{$tAlias.UpSingular}}Slice, *{{$tAlias.UpSingular}}Setter]
{{- end}}
{{- end}}
