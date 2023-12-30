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
{{$.Importer.Import (printf "github.com/stephenafamo/bob/dialect/%s/dialect" $.Dialect)}}
var (
	SelectWhere = Where[*dialect.SelectQuery]()
	InsertWhere = Where[*dialect.InsertQuery]()
	UpdateWhere = Where[*dialect.UpdateQuery]()
	DeleteWhere = Where[*dialect.DeleteQuery]()
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
		{{$tAlias.UpPlural}}: {{$tAlias.UpSingular}}Where[Q](),
		{{end -}}
	}
}

{{block "helpers/join_variables" . -}}
var (
	SelectJoins = getJoins[*dialect.SelectQuery]
	UpdateJoins = getJoins[*dialect.UpdateQuery]
	DeleteJoins = getJoins[*dialect.DeleteQuery]
)
{{- end}}

type joinSet[Q any] struct {
    InnerJoin Q
    LeftJoin Q
    RightJoin Q
}

type joins[Q dialect.Joinable] struct {
		{{range $table := .Tables -}}{{if $.Relationships.Get $table.Key -}}
		{{$tAlias := $.Aliases.Table $table.Key -}}
		{{$tAlias.UpPlural}} joinSet[{{$tAlias.DownSingular}}RelationshipJoins[Q]]
		{{end}}{{end}}
}

{{$.Importer.Import "context"}}
func getJoins[Q dialect.Joinable](ctx context.Context) joins[Q] {
	return joins[Q]{
		{{range $table := .Tables -}}{{if $.Relationships.Get $table.Key -}}
		{{$tAlias := $.Aliases.Table $table.Key -}}
		{{$tAlias.UpPlural}}: {{$tAlias.DownPlural}}Join[Q](ctx),
		{{end}}{{end}}
	}
}
