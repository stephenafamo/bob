{{$table := .Table}}
{{$tAlias := .Aliases.Table $table.Name -}}

type {{$tAlias.DownSingular}}Where[Q model.Filterable] struct {
	{{range $column := $table.Columns -}}
	{{- $colAlias := $tAlias.Column $column.Name -}}
		{{- if $column.Nullable -}}
			{{$colAlias}} model.WhereNullMod[Q, {{$column.Type}}]
		{{- else -}}
			{{$colAlias}} model.WhereMod[Q, {{$column.Type}}]
		{{- end}}
  {{end -}}
}

func {{$tAlias.UpSingular}}Where[Q model.Filterable]() {{$tAlias.DownSingular}}Where[Q] {
	return {{$tAlias.DownSingular}}Where[Q]{
			{{range $column := $table.Columns -}}
			{{- $colAlias := $tAlias.Column $column.Name -}}
				{{- if $column.Nullable -}}
					{{$colAlias}}: model.WhereNull[Q, {{$column.Type}}]({{$.Dialect}}.Quote("{{$table.Name}}", "{{$column.Name}}")),
				{{- else -}}
					{{$colAlias}}: model.Where[Q, {{$column.Type}}]({{$.Dialect}}.Quote({{quote $table.Name}}, {{quote $column.Name}})),
				{{- end}}
			{{end -}}
	}
}

