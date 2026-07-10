{{- define "helpers/join_variables"}}
{{if $.IsTablePackage -}}
{{$table := index .Tables 0 -}}
{{$tAlias := $.Aliases.Table $table.Key -}}
{{if $.Relationships.Get $table.Key -}}
var (
	SelectJoins = BuildJoinSet[{{$tAlias.UpSingular}}Joins[*dialect.SelectQuery]]({{$.TableVar $table.Key}}.Columns, Build{{$tAlias.UpSingular}}Joins[*dialect.SelectQuery])
	UpdateJoins = BuildJoinSet[{{$tAlias.UpSingular}}Joins[*dialect.UpdateQuery]]({{$.TableVar $table.Key}}.Columns, Build{{$tAlias.UpSingular}}Joins[*dialect.UpdateQuery])
)
{{end -}}
{{else -}}
var (
	SelectJoins = getJoins[*dialect.SelectQuery]
	UpdateJoins = getJoins[*dialect.UpdateQuery]
)
{{end -}}
{{end -}}
