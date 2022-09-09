{{$table := .Table}}
{{$tAlias := .Aliases.Table $table.Name -}}

// {{$tAlias.UpSingular}}Slice is an alias for a slice of pointers to {{$tAlias.UpSingular}}.
// This should almost always be used instead of []{{$tAlias.UpSingular}}.
type {{$tAlias.UpSingular}}Slice []*{{$tAlias.UpSingular}}

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
	{{- if ignore $table.Name $orig_col_name $.TagIgnore}}
	{{$colAlias}} {{$colTyp}} `{{generateIgnoreTags $.Tags}}db:"{{dbTag $table $column}}" json:"-" toml:"-" yaml:"-"`
	{{- else}}{{$tagName := columnTagName $.StructTagCasing $column.Name $colAlias}}
		{{$colAlias}} {{$colTyp}} `{{generateTags $.Tags $tagName}}db:"{{dbTag $table $column}}" json:"{{$tagName}}{{if $column.Nullable}},omitempty{{end}}" toml:"{{$tagName}}" yaml:"{{$tagName}}{{if $column.Nullable}},omitempty{{end}}"`
	{{- end -}}
	{{- end -}}
	{{- if .Table.Relationships}}

	R {{$tAlias.DownSingular}}R `{{generateTags $.Tags $.RelationTag}}db:"{{$.RelationTag}}" json:"{{$.RelationTag}}" toml:"{{$.RelationTag}}" yaml:"{{$.RelationTag}}"`
	{{end -}}
}

{{if .Table.PKey -}}
// Optional{{$tAlias.UpSingular}} is used for insert/upsert operations
// Fields that have default values are optional, and do not have to be set
// Fields without default values must be set or the zero value will be used
// Generated columns are not included
type Optional{{$tAlias.UpSingular}} struct {
	{{- range $column := .Table.Columns -}}
	{{- if $column.Generated}}{{continue}}{{end -}}
	{{- $colAlias := $tAlias.Column $column.Name -}}
	{{- $colTyp := $column.Type -}}
		{{- if and $column.Default $column.Nullable -}}
			{{- $.Importer.Import "github.com/aarondl/opt/omitnull" -}}
			{{- $colTyp = printf "omitnull.Val[%s]" $column.Type -}}
		{{- else if $column.Default -}}
			{{- $.Importer.Import "github.com/aarondl/opt/omit" -}}
			{{- $colTyp = printf "omit.Val[%s]" $column.Type -}}
		{{- else if $column.Nullable -}}
			{{- $.Importer.Import "github.com/aarondl/opt/null" -}}
			{{- $colTyp = printf "null.Val[%s]" $column.Type -}}
		{{- end -}}
		{{$colAlias}} {{$colTyp}} `db:"{{dbTag $table $column}}"`
	{{end -}}
}

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

type {{$tAlias.DownSingular}}ColumnNames struct {
	{{range $column := $table.Columns -}}
	{{- $colAlias := $tAlias.Column $column.Name -}}
	{{$colAlias}} string
  {{end -}}
}
