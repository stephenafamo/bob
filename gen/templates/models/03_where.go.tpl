{{$table := .Table}}
{{$tAlias := .Aliases.Table $table.Name -}}

type {{$tAlias.DownSingular}}Where[Q {{$.Dialect}}.Filterable] struct {
	{{range $column := $table.Columns -}}
	{{- $colAlias := $tAlias.Column $column.Name -}}
		{{- if $column.Nullable -}}
			{{$colAlias}} {{$.Dialect}}.WhereNullMod[Q, {{$column.Type}}]
		{{- else -}}
			{{$colAlias}} {{$.Dialect}}.WhereMod[Q, {{$column.Type}}]
		{{- end}}
  {{end -}}
}

func {{$tAlias.UpSingular}}Where[Q {{$.Dialect}}.Filterable]() {{$tAlias.DownSingular}}Where[Q] {
	return {{$tAlias.DownSingular}}Where[Q]{
			{{range $column := $table.Columns -}}
			{{- $colAlias := $tAlias.Column $column.Name -}}
				{{- if $column.Nullable -}}
					{{$colAlias}}: {{$.Dialect}}.WhereNull[Q, {{$column.Type}}]({{$.Dialect}}.Quote("{{$table.Name}}", "{{$column.Name}}")),
				{{- else -}}
					{{$colAlias}}: {{$.Dialect}}.Where[Q, {{$column.Type}}]({{$.Dialect}}.Quote({{quote $table.Name}}, {{quote $column.Name}})),
				{{- end}}
			{{end -}}
	}
}

