{{$table := .Table}}
{{$tAlias := .Aliases.Table $table.Key -}}

// {{$tAlias.UpSingular}} is an object representing the database table.
type {{$tAlias.UpSingular}} struct {
	{{- range $column := $table.Columns -}}
	{{- $colAlias := $tAlias.Column $column.Name -}}
	{{- $orig_col_name := $column.Name -}}
  {{- $colTyp := $.Types.GetNullable $.CurrentPackage $.Importer $column.Type $column.Nullable -}}
	{{- if trim $column.Comment}}{{range $column.Comment | splitList "\n"}}
		// {{ . }}
	{{- end}}{{end -}}
	{{- if ignore $table.Key $orig_col_name $.TagIgnore}}
	{{$colAlias}} {{$colTyp}} `db:"{{$table.DBTag $column}}" {{generateIgnoreTags $.Tags | trim}}`
	{{- else}}{{$tagName := columnTagName $.StructTagCasing $column.Name $colAlias}}
		{{$colAlias}} {{$colTyp}} `db:"{{$table.DBTag $column}}" {{generateTags $.Tags $tagName | trim}}`
	{{- end -}}
	{{- end -}}
	{{block "model/fields/additional" $}}{{end}}
	{{- if $.Relationships.Get $table.Key}}

	R {{$tAlias.DownSingular}}R `db:"-" {{generateTags $.Tags $.RelationTag | trim}}`
	{{end -}}
}

// {{$tAlias.UpSingular}}Slice is an alias for a slice of pointers to {{$tAlias.UpSingular}}.
// This should almost always be used instead of []*{{$tAlias.UpSingular}}.
type {{$tAlias.UpSingular}}Slice []*{{$tAlias.UpSingular}}

{{block "model_and_query" . -}}
{{$table := .Table}}
{{$tAlias := .Aliases.Table $table.Key -}}
{{if not $table.Constraints.Primary -}}
	// {{$tAlias.UpPlural}} contains methods to work with the {{$table.Name}} view
	var {{$tAlias.UpPlural}} = {{$.Dialect}}.NewViewx[*{{$tAlias.UpSingular}}, {{$tAlias.UpSingular}}Slice]("{{$table.Schema}}","{{$table.Name}}")
	// {{$tAlias.UpPlural}}Query is a query on the {{$table.Name}} view
	type {{$tAlias.UpPlural}}Query = *{{$.Dialect}}.ViewQuery[*{{$tAlias.UpSingular}}, {{$tAlias.UpSingular}}Slice]
{{- else -}}
	// {{$tAlias.UpPlural}} contains methods to work with the {{$table.Name}} table
	var {{$tAlias.UpPlural}} = {{$.Dialect}}.NewTablex[*{{$tAlias.UpSingular}}, {{$tAlias.UpSingular}}Slice, *{{$tAlias.UpSingular}}Setter]("{{$table.Schema}}","{{$table.Name}}")
	// {{$tAlias.UpPlural}}Query is a query on the {{$table.Name}} table
	type {{$tAlias.UpPlural}}Query = *{{$.Dialect}}.ViewQuery[*{{$tAlias.UpSingular}}, {{$tAlias.UpSingular}}Slice]
{{- end}}
{{- end}}

{{if $.Relationships.Get $table.Key -}}
{{$.Importer.Import (printf "github.com/stephenafamo/bob/dialect/%s/dialect" $.Dialect)}}
// {{$tAlias.DownSingular}}R is where relationships are stored.
type {{$tAlias.DownSingular}}R struct {
	{{range $.Relationships.Get $table.Key -}}
	{{- $ftable := $.Aliases.Table .Foreign -}}
	{{- $relAlias := $tAlias.Relationship .Name -}}
	{{if .IsToMany -}}
		{{$relAlias}} {{$ftable.UpSingular}}Slice {{if $.Tags}}`{{generateTags $.Tags $relAlias | trim}}`{{end}} // {{.Name}}
	{{else -}}
		{{$relAlias}} *{{$ftable.UpSingular}} {{if $.Tags}}`{{generateTags $.Tags $relAlias | trim}}`{{end}} // {{.Name}}
	{{end}}{{end -}}
}
{{- end}}

type {{$tAlias.DownSingular}}ColumnNames struct {
	{{range $column := $table.Columns -}}
	{{- $colAlias := $tAlias.Column $column.Name -}}
	{{$colAlias}} string
  {{end -}}
}

{{$.Importer.Import (printf "github.com/stephenafamo/bob/dialect/%s" $.Dialect)}}
var {{$tAlias.UpSingular}}Columns = build{{$tAlias.UpSingular}}Columns({{quote $table.Key}})

type {{$tAlias.DownSingular}}Columns struct {
  tableAlias string
	{{range $column := $table.Columns -}}
	{{- $colAlias := $tAlias.Column $column.Name -}}
	{{$colAlias}} {{$.Dialect}}.Expression
	{{end -}}
}

func (c {{$tAlias.DownSingular}}Columns) Alias() string {
  return c.tableAlias
}

func ({{$tAlias.DownSingular}}Columns) AliasedAs(alias string) {{$tAlias.DownSingular}}Columns {
  return build{{$tAlias.UpSingular}}Columns(alias)
}

func build{{$tAlias.UpSingular}}Columns(alias string) {{$tAlias.DownSingular}}Columns {
  return {{$tAlias.DownSingular}}Columns{
    tableAlias: alias,
    {{range $column := $table.Columns -}}
    {{- $colAlias := $tAlias.Column $column.Name -}}
    {{$colAlias}}: {{$.Dialect}}.Quote(alias, {{quote $column.Name}}),
    {{end -}}
  }
}


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

{{ $hasUniqueIndex := false }}
{{range $index := $table.Indexes}}
{{ if $index.Unique }} 
	{{ $hasUniqueIndex = true }}
	{{ break }}
{{ end }} 
{{end -}}

{{ if or $hasUniqueIndex $table.Constraints.Primary }}
var {{$tAlias.UpSingular}}Errors = &{{$tAlias.DownSingular}}Errors{
  {{if $table.Constraints.Primary}}
  {{$pk := $table.Constraints.Primary}}
	ErrUnique{{$pk.Name | camelcase}}: &UniqueConstraintError{
    schema: {{printf "%q" $table.Schema}},
    table: {{printf "%q" $table.Name}},
    columns: {{printf "%#v" $pk.Columns}},
    s: {{printf "%q" $pk.Name}},
  },
  {{end}}
	{{range $index := $table.Constraints.Uniques}}
	ErrUnique{{$index.Name | camelcase}}: &UniqueConstraintError{
    schema: {{printf "%q" $table.Schema}},
    table: {{printf "%q" $table.Name}},
    columns: {{printf "%#v" $index.Columns}},
    s: "{{$index.Name}}",
  },
	{{end}}
}

type {{$tAlias.DownSingular}}Errors struct {
  {{if $table.Constraints.Primary}}
  {{$pk := $table.Constraints.Primary}}
	ErrUnique{{$pk.Name | camelcase}} *UniqueConstraintError
  {{end}}
	{{range $index := $table.Constraints.Uniques}}
	ErrUnique{{$index.Name | camelcase}} *UniqueConstraintError
	{{end}}
}
{{ end }}
