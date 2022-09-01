var TableNames = struct {
	{{range $table := .Tables}}{{if not $table.IsView -}}
	{{$tAlias := $.Aliases.Table $table.Name -}}
	{{$tAlias.UpPlural}} string
	{{end}}{{end -}}
}{
	{{range $table := .Tables}}{{if not $table.IsView -}}
	{{$tAlias := $.Aliases.Table $table.Name -}}
	{{$tAlias.UpPlural}}: {{quote $table.Name}},
	{{end}}{{end -}}
}

var ViewNames = struct {
	{{range $table := .Tables}}{{if $table.IsView -}}
	{{$tAlias := $.Aliases.Table $table.Name -}}
	{{$tAlias.UpPlural}} string
	{{end}}{{end -}}
}{
	{{range $table := .Tables}}{{if $table.IsView -}}
	{{$tAlias := $.Aliases.Table $table.Name -}}
	{{$tAlias.UpPlural}}: {{quote $table.Name}},
	{{end}}{{end -}}
}

var ColumnNames = struct {
	{{range $table := .Tables -}}
	{{$tAlias := $.Aliases.Table $table.Name -}}
	{{$tAlias.UpPlural}} {{$tAlias.DownSingular}}ColumnNames
	{{end -}}
}{
	{{range $table := .Tables -}}
	{{$tAlias := $.Aliases.Table $table.Name -}}
	{{$tAlias.UpPlural}}: {{$tAlias.DownSingular}}ColumnNames{
		{{range $column := $table.Columns -}}
		{{- $colAlias := $tAlias.Column $column.Name -}}
		{{$colAlias}}: {{quote $column.Name}},
		{{end -}}
	},
	{{end -}}
}

{{$.Importer.Import (printf "github.com/stephenafamo/bob/dialect/%s" $.Dialect)}}
var (
	SelectWhere = Where[*{{$.Dialect}}.SelectQuery]()
	InsertWhere = Where[*{{$.Dialect}}.InsertQuery]()
	UpdateWhere = Where[*{{$.Dialect}}.UpdateQuery]()
	DeleteWhere = Where[*{{$.Dialect}}.DeleteQuery]()
)

{{$.Importer.Import (printf "github.com/stephenafamo/bob/dialect/%s/model" $.Dialect)}}
func Where[Q model.Filterable]() struct {
	{{range $table := .Tables -}}
	{{$tAlias := $.Aliases.Table $table.Name -}}
	{{$tAlias.UpPlural}} {{$tAlias.DownSingular}}Where[Q]
	{{end -}}
} {
	return struct {
		{{range $table := .Tables -}}
		{{$tAlias := $.Aliases.Table $table.Name -}}
		{{$tAlias.UpPlural}} {{$tAlias.DownSingular}}Where[Q]
		{{end -}}
	}{
		{{range $table := .Tables -}}
		{{$tAlias := $.Aliases.Table $table.Name -}}
		{{$tAlias.UpPlural}}: {{$tAlias.UpSingular}}Where[Q](),
		{{end -}}
	}
}
