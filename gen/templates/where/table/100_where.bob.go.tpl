{{$.Importer.Import (printf "github.com/stephenafamo/bob/dialect/%s" $.Dialect)}}

{{$table := .Table}}
{{$tAlias := .Aliases.Table $table.Key -}}

type {{$tAlias.DownSingular}}Where[Q {{$.Dialect}}.Filterable] struct {
	{{range $column := $table.Columns -}}
    {{- $colAlias := $tAlias.Column $column.Name -}}
    {{- $colTyp := $.Types.Get $.CurrentPackage $.Importer $column.Type -}}
		{{- if $column.Nullable -}}
			{{$colAlias}} {{$.Dialect}}.WhereNullMod[Q, {{$colTyp}}]
		{{- else -}}
			{{$colAlias}} {{$.Dialect}}.WhereMod[Q, {{$colTyp}}]
		{{- end}}
  {{end -}}
}

func ({{$tAlias.DownSingular}}Where[Q]) AliasedAs(alias string) {{$tAlias.DownSingular}}Where[Q] {
	return build{{$tAlias.UpSingular}}Where[Q](build{{$tAlias.UpSingular}}Columns(alias))
}

func build{{$tAlias.UpSingular}}Where[Q {{$.Dialect}}.Filterable](cols {{$tAlias.DownSingular}}Columns) {{$tAlias.DownSingular}}Where[Q] {
	return {{$tAlias.DownSingular}}Where[Q]{
			{{range $column := $table.Columns -}}
      {{- $colAlias := $tAlias.Column $column.Name -}}
      {{- $colTyp := $.Types.Get $.CurrentPackage $.Importer $column.Type -}}
				{{- if $column.Nullable -}}
					{{$colAlias}}: {{$.Dialect}}.WhereNull[Q, {{$colTyp}}](cols.{{$colAlias}}),
				{{- else -}}
					{{$colAlias}}: {{$.Dialect}}.Where[Q, {{$colTyp}}](cols.{{$colAlias}}),
				{{- end}}
			{{end -}}
	}
}
