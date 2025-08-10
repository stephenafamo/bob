var TableNames = struct {
	{{range $table := .Tables -}}
	{{$tAlias := $.Aliases.Table $table.Key -}}
	{{$tAlias.UpPlural}} string
	{{end -}}
}{
	{{range $table := .Tables -}}
	{{$tAlias := $.Aliases.Table $table.Key -}}
	{{$tAlias.UpPlural}}: {{quote $table.Name}},
	{{end -}}
}

var ColumnNames = struct {
	{{range $table := .Tables -}}
	{{$tAlias := $.Aliases.Table $table.Key -}}
	{{$tAlias.UpPlural}} {{$tAlias.DownSingular}}ColumnNames
	{{end -}}
}{
	{{range $table := .Tables -}}
	{{$tAlias := $.Aliases.Table $table.Key -}}
	{{$tAlias.UpPlural}}: {{$tAlias.DownSingular}}ColumnNames{
		{{range $column := $table.Columns -}}
		{{- $colAlias := $tAlias.Column $column.Name -}}
		{{$colAlias}}: {{quote $column.Name}},
		{{end -}}
	},
	{{end -}}
}

{{block "helpers/where_variables" . -}}
{{$.Importer.Import "github.com/stephenafamo/bob/clause"}}
{{$.Importer.Import (printf "github.com/stephenafamo/bob/dialect/%s/dialect" $.Dialect)}}
var (
	SelectWhere = Where[*dialect.SelectQuery]()
	UpdateWhere = Where[*dialect.UpdateQuery]()
	DeleteWhere = Where[*dialect.DeleteQuery]()
	OnConflictWhere = Where[*clause.ConflictClause]() // Used in ON CONFLICT DO UPDATE
)
{{- end}}

{{$.Importer.Import (printf "github.com/stephenafamo/bob/dialect/%s" $.Dialect)}}
func Where[Q {{$.Dialect}}.Filterable]() struct {
	{{range $table := .Tables -}}
	{{$tAlias := $.Aliases.Table $table.Key -}}
	{{$tAlias.UpPlural}} {{$tAlias.DownSingular}}Where[Q]
	{{end -}}
} {
	return struct {
		{{range $table := .Tables -}}
		{{$tAlias := $.Aliases.Table $table.Key -}}
		{{$tAlias.UpPlural}} {{$tAlias.DownSingular}}Where[Q]
		{{end -}}
	}{
		{{range $table := .Tables -}}
		{{$tAlias := $.Aliases.Table $table.Key -}}
		{{$tAlias.UpPlural}}: build{{$tAlias.UpSingular}}Where[Q]({{$tAlias.UpSingular}}Columns),
		{{end -}}
	}
}

