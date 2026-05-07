{{$.Importer.Import (printf "github.com/stephenafamo/bob/dialect/%s" $.Dialect)}}

{{$table := .Table}}
{{$tAlias := .Aliases.Table $table.Key -}}

type {{$tAlias.UpSingular}}Where[Q {{$.Dialect}}.Filterable] struct {
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

func ({{$tAlias.UpSingular}}Where[Q]) AliasedAs(alias string) {{$tAlias.UpSingular}}Where[Q] {
	return Build{{$tAlias.UpSingular}}Where[Q](Build{{$tAlias.UpSingular}}Columns(alias))
}

func Build{{$tAlias.UpSingular}}Where[Q {{$.Dialect}}.Filterable](cols {{$tAlias.UpSingular}}Columns) {{$tAlias.UpSingular}}Where[Q] {
	return {{$tAlias.UpSingular}}Where[Q]{
			{{range $column := $table.Columns -}}
      {{- $colAlias := $tAlias.Column $column.Name -}}
      {{- $colTyp := $.Types.Get $.CurrentPackage $.Importer $column.Type -}}
				{{- if $column.Nullable -}}
					{{$colAlias}}: {{$.Dialect}}.WhereNull[Q, {{$colTyp}}](cols.{{$colAlias}}.Expression),
				{{- else -}}
					{{$colAlias}}: {{$.Dialect}}.Where[Q, {{$colTyp}}](cols.{{$colAlias}}.Expression),
				{{- end}}
			{{end -}}
	}
}
