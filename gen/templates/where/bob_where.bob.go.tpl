{{block "helpers/where_variables" . -}}
{{$.Importer.Import "github.com/stephenafamo/bob/clause"}}
{{$.Importer.Import (printf "github.com/stephenafamo/bob/dialect/%s/dialect" $.Dialect)}}
{{if $.IsTablePackage -}}
{{$table := index .Tables 0 -}}
{{$tAlias := $.Aliases.Table $table.Key -}}
var (
	SelectWhere = {{$.BuildWhereFunc $table.Key}}[*dialect.SelectQuery]({{$.TableVar $table.Key}}.Columns)
	UpdateWhere = {{$.BuildWhereFunc $table.Key}}[*dialect.UpdateQuery]({{$.TableVar $table.Key}}.Columns)
	DeleteWhere = {{$.BuildWhereFunc $table.Key}}[*dialect.DeleteQuery]({{$.TableVar $table.Key}}.Columns)
	OnConflictWhere = {{$.BuildWhereFunc $table.Key}}[*clause.ConflictClause]({{$.TableVar $table.Key}}.Columns)
)
{{else -}}
var (
	SelectWhere = Where[*dialect.SelectQuery]()
	UpdateWhere = Where[*dialect.UpdateQuery]()
	DeleteWhere = Where[*dialect.DeleteQuery]()
	OnConflictWhere = Where[*clause.ConflictClause]() // Used in ON CONFLICT DO UPDATE
)
{{end -}}
{{- end}}

{{$.Importer.Import (printf "github.com/stephenafamo/bob/dialect/%s" $.Dialect)}}
{{if $.IsTablePackage -}}
{{$table := index .Tables 0 -}}
func Where[Q {{$.Dialect}}.Filterable]() {{$.WhereType $table.Key}}[Q] {
	return {{$.BuildWhereFunc $table.Key}}[Q]({{$.TableVar $table.Key}}.Columns)
}
{{else -}}
func Where[Q {{$.Dialect}}.Filterable]() struct {
	{{range $table := .Tables -}}
	{{$tAlias := $.Aliases.Table $table.Key -}}
	{{$tAlias.UpPlural}} {{$.WhereType $table.Key}}[Q]
	{{end -}}
} {
	return struct {
		{{range $table := .Tables -}}
		{{$tAlias := $.Aliases.Table $table.Key -}}
		{{$tAlias.UpPlural}} {{$.WhereType $table.Key}}[Q]
		{{end -}}
	}{
		{{range $table := .Tables -}}
		{{$tAlias := $.Aliases.Table $table.Key -}}
		{{$tAlias.UpPlural}}: {{$.BuildWhereFunc $table.Key}}[Q]({{$.TableVar $table.Key}}.Columns),
		{{end -}}
	}
}
{{end -}}
