{{$.Importer.Import (printf "github.com/stephenafamo/bob/dialect/%s" $.Dialect)}}

{{$table := .Table}}
{{$tAlias := .Aliases.Table $table.Key -}}

type {{$tAlias.DownSingular}}Where[Q {{$.Dialect}}.Filterable] struct {
	cols {{$tAlias.DownSingular}}Columns
	{{range $column := $table.Columns -}}
    {{- $colAlias := $tAlias.Column $column.Name -}}
    {{- $colTyp := $.Types.Get $.CurrentPackage $.Importer $column.Type -}}
		{{- if $column.Nullable -}}
			{{$colAlias}} {{$.Dialect}}.WhereNullMod[Q, {{$colTyp}}]
		{{- else -}}
			{{$colAlias}} {{$.Dialect}}.WhereMod[Q, {{$colTyp}}]
		{{- end}}
  {{end -}}
  {{if $.Relationships.Get $table.Key -}}
	R {{$tAlias.DownSingular}}WhereR[Q]
  {{end -}}
}

func ({{$tAlias.DownSingular}}Where[Q]) AliasedAs(alias string) {{$tAlias.DownSingular}}Where[Q] {
	return build{{$tAlias.UpSingular}}Where[Q](build{{$tAlias.UpSingular}}Columns(alias))
}

func build{{$tAlias.UpSingular}}Where[Q {{$.Dialect}}.Filterable](cols {{$tAlias.DownSingular}}Columns) {{$tAlias.DownSingular}}Where[Q] {
	return {{$tAlias.DownSingular}}Where[Q]{
			cols: cols,
			{{range $column := $table.Columns -}}
      {{- $colAlias := $tAlias.Column $column.Name -}}
      {{- $colTyp := $.Types.Get $.CurrentPackage $.Importer $column.Type -}}
				{{- if $column.Nullable -}}
					{{$colAlias}}: {{$.Dialect}}.WhereNull[Q, {{$colTyp}}](cols.{{$colAlias}}.Expression),
				{{- else -}}
					{{$colAlias}}: {{$.Dialect}}.Where[Q, {{$colTyp}}](cols.{{$colAlias}}.Expression),
				{{- end}}
			{{end -}}
			{{if $.Relationships.Get $table.Key -}}
			R: {{$tAlias.DownSingular}}WhereR[Q]{cols: cols},
			{{end -}}
	}
}

{{if $.Relationships.Get $table.Key -}}
// {{$tAlias.DownSingular}}WhereR holds the relationship-based filters of
// {{$tAlias.DownSingular}}Where under the R namespace — mirroring the model's
// R struct — so they cannot collide with column-based filter fields.
type {{$tAlias.DownSingular}}WhereR[Q {{$.Dialect}}.Filterable] struct {
	cols {{$tAlias.DownSingular}}Columns
}
{{end -}}


{{range $rel := $.Relationships.Get $table.Key -}}
{{$.Importer.Import "github.com/stephenafamo/bob"}}
{{$.Importer.Import "github.com/stephenafamo/bob/mods"}}
{{$.Importer.Import (printf "github.com/stephenafamo/bob/dialect/%s/dialect" $.Dialect)}}
{{$relAlias := $tAlias.Relationship $rel.Name -}}
// Has{{$relAlias}} filters by a correlated EXISTS without multiplying the parent rows.
func (w {{$tAlias.DownSingular}}WhereR[Q]) Has{{$relAlias}}(filters ...bob.Mod[dialect.SelectEditor]) mods.Where[Q] {
    q := correlatedQuery({{$tAlias.DownSingular}}Rel{{$relAlias}}, w.cols.Alias(), {{$.Dialect}}.Raw("1"), filters)
    return mods.Where[Q]{E: {{$.Dialect}}.Exists(q)}
}
{{end -}}
