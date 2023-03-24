{{$table := .Table}}
{{$tAlias := .Aliases.Table $table.Key -}}

// {{$tAlias.UpSingular}} is an object representing the database table.
type {{$tAlias.UpSingular}} struct {
	{{- range $column := .Table.Columns -}}
	{{- $colAlias := $tAlias.Column $column.Name -}}
	{{- $colTyp := $column.Type -}}
	{{- $.Importer.ImportList $column.Imports -}}
	{{- $orig_col_name := $column.Name -}}
	{{- if $column.Nullable -}}
		{{- $colTyp = printf "null.Val[%s]" $column.Type -}}
		{{ $.Importer.Import "github.com/aarondl/opt/null"}}
	{{- end -}}
	{{- if trim $column.Comment}}{{range $column.Comment | splitList "\n"}}
		// {{ . }}
	{{- end}}{{end -}}
	{{- if ignore $table.Key $orig_col_name $.TagIgnore}}
	{{$colAlias}} {{$colTyp}} `db:"{{dbTag $table $column}}" {{generateIgnoreTags $.Tags | trim}}`
	{{- else}}{{$tagName := columnTagName $.StructTagCasing $column.Name $colAlias}}
		{{$colAlias}} {{$colTyp}} `db:"{{dbTag $table $column}}" {{generateTags $.Tags $tagName | trim}}`
	{{- end -}}
	{{- end -}}
	{{- if .Table.Relationships}}

	R {{$tAlias.DownSingular}}R `db:"-" {{generateTags $.Tags $.RelationTag | trim}}`
	{{end -}}
}

// {{$tAlias.UpSingular}}Slice is an alias for a slice of pointers to {{$tAlias.UpSingular}}.
// This should almost always be used instead of []{{$tAlias.UpSingular}}.
type {{$tAlias.UpSingular}}Slice []*{{$tAlias.UpSingular}}

{{block "model_and_query" . -}}
{{$table := .Table}}
{{$tAlias := .Aliases.Table $table.Key -}}
{{if not $table.PKey -}}
	// {{$tAlias.UpPlural}}View contains methods to work with the {{$table.Name}} view
	var {{$tAlias.UpPlural}}View = {{$.Dialect}}.NewViewx[*{{$tAlias.UpSingular}}, {{$tAlias.UpSingular}}Slice]("{{$table.Schema}}","{{$table.Name}}")
	// {{$tAlias.UpPlural}}Query is a query on the {{$table.Name}} view
	type {{$tAlias.UpPlural}}Query = *{{$.Dialect}}.ViewQuery[*{{$tAlias.UpSingular}}, {{$tAlias.UpSingular}}Slice]
{{- else -}}
	// {{$tAlias.UpPlural}}Table contains methods to work with the {{$table.Name}} table
	var {{$tAlias.UpPlural}}Table = {{$.Dialect}}.NewTablex[*{{$tAlias.UpSingular}}, {{$tAlias.UpSingular}}Slice, *{{$tAlias.UpSingular}}Setter]("{{$table.Schema}}","{{$table.Name}}")
	// {{$tAlias.UpPlural}}Query is a query on the {{$table.Name}} table
	type {{$tAlias.UpPlural}}Query = *{{$.Dialect}}.TableQuery[*{{$tAlias.UpSingular}}, {{$tAlias.UpSingular}}Slice, *{{$tAlias.UpSingular}}Setter]
{{- end}}
{{- end}}

// {{$tAlias.UpPlural}}Stmt is a prepared statment on {{$table.Name}}
type {{$tAlias.UpPlural}}Stmt = bob.QueryStmt[*{{$tAlias.UpSingular}}, {{$tAlias.UpSingular}}Slice]

{{if .Table.Relationships -}}
// {{$tAlias.DownSingular}}R is where relationships are stored.
type {{$tAlias.DownSingular}}R struct {
	{{range .Table.Relationships -}}
	{{- $ftable := $.Aliases.Table .Foreign -}}
	{{- $relAlias := $tAlias.Relationship .Name -}}
	{{if .IsToMany -}}
	{{$relAlias}} {{$ftable.UpSingular}}Slice `{{generateTags $.Tags $relAlias}}db:"{{$relAlias}}" json:"{{$relAlias}}" toml:"{{$relAlias}}" yaml:"{{$relAlias}}"`
	{{else -}}
	{{$relAlias}} *{{$ftable.UpSingular}} `{{generateTags $.Tags $relAlias}}db:"{{$relAlias}}" json:"{{$relAlias}}" toml:"{{$relAlias}}" yaml:"{{$relAlias}}"`
	{{end}}{{end -}}
}
{{- end}}

{{if .Table.PKey -}}
// {{$tAlias.UpSingular}}Setter is used for insert/upsert/update operations
// All values are optional, and do not have to be set
// Generated columns are not included
type {{$tAlias.UpSingular}}Setter struct {
	{{- range $column := .Table.Columns -}}
	{{- if $column.Generated}}{{continue}}{{end -}}
	{{- $colAlias := $tAlias.Column $column.Name -}}
	{{- $colTyp := "" -}}
		{{- if $column.Nullable -}}
			{{- $.Importer.Import "github.com/aarondl/opt/omitnull" -}}
			{{- $colTyp = printf "omitnull.Val[%s]" $column.Type -}}
		{{- else -}}
			{{- $.Importer.Import "github.com/aarondl/opt/omit" -}}
			{{- $colTyp = printf "omit.Val[%s]" $column.Type -}}
		{{- end -}}
		{{$colAlias}} {{$colTyp}} `db:"{{dbTag $table $column}}"`
	{{end -}}
}

{{- end}}

type {{$tAlias.DownSingular}}ColumnNames struct {
	{{range $column := $table.Columns -}}
	{{- $colAlias := $tAlias.Column $column.Name -}}
	{{$colAlias}} string
  {{end -}}
}

{{if $table.Relationships -}}
type {{$tAlias.DownSingular}}RelationshipJoins[Q dialect.Joinable] struct {
	{{range $table.Relationships -}}
	{{- $relAlias := $tAlias.Relationship .Name -}}
	{{$relAlias}} bob.Mod[Q]
  {{end -}}
}

func build{{$tAlias.DownSingular}}RelationshipJoins[Q dialect.Joinable](ctx context.Context, typ string) {{$tAlias.DownSingular}}RelationshipJoins[Q] {
  return {{$tAlias.DownSingular}}RelationshipJoins[Q]{
		{{range $table.Relationships -}}
			{{$ftable := $.Aliases.Table .Foreign -}}
			{{$relAlias := $tAlias.Relationship .Name -}}
			{{$relAlias}}: {{$tAlias.DownPlural}}Join{{$relAlias}}[Q](ctx, typ),
		{{end}}
	}
}

{{$.Importer.Import "github.com/stephenafamo/bob/clause"}}
func {{$tAlias.DownPlural}}Join[Q dialect.Joinable](ctx context.Context) joinSet[{{$tAlias.DownSingular}}RelationshipJoins[Q]] {
	return joinSet[{{$tAlias.DownSingular}}RelationshipJoins[Q]] {
	  InnerJoin: build{{$tAlias.DownSingular}}RelationshipJoins[Q](ctx, clause.InnerJoin),
	  LeftJoin: build{{$tAlias.DownSingular}}RelationshipJoins[Q](ctx, clause.LeftJoin),
	  RightJoin: build{{$tAlias.DownSingular}}RelationshipJoins[Q](ctx, clause.RightJoin),
	}
}
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
	{{$colAlias}}: {{$.Dialect}}.Quote({{quote $table.Key}}, {{quote $column.Name}}),
	{{end -}}
}

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
					{{$colAlias}}: {{$.Dialect}}.WhereNull[Q, {{$column.Type}}]({{$tAlias.UpSingular}}Columns.{{$colAlias}}),
				{{- else -}}
					{{$colAlias}}: {{$.Dialect}}.Where[Q, {{$column.Type}}]({{$tAlias.UpSingular}}Columns.{{$colAlias}}),
				{{- end}}
			{{end -}}
	}
}

