{{$table := .Table}}
{{$tAlias := .Aliases.Table .Table.Key -}}

{{if .Table.Constraints.Primary -}}
{{$pkArgs := ""}}
{{range $colName := $table.Constraints.Primary.Columns -}}
{{- $column := $table.GetColumn $colName -}}
{{- $colAlias := $tAlias.Column $colName -}}
{{$pkArgs = printf "%s%sPK %s," $pkArgs $colAlias $column.Type}}
{{end -}}

{{$.Importer.Import (printf "github.com/stephenafamo/bob/dialect/%s/sm" $.Dialect)}}
// Find{{$tAlias.UpSingular}} retrieves a single record by primary key
// If cols is empty Find will return all columns.
func Find{{$tAlias.UpSingular}}(ctx context.Context, exec bob.Executor, {{$pkArgs}} cols ...string) (*{{$tAlias.UpSingular}}, error) {
	if len(cols) == 0 {
		return {{$tAlias.UpPlural}}.Query(
			ctx, exec,
			{{range $column := $table.Constraints.Primary.Columns -}}
			{{- $colAlias := $tAlias.Column $column -}}
			SelectWhere.{{$tAlias.UpPlural}}.{{$colAlias}}.EQ({{$colAlias}}PK),
			{{end -}}
		).One()
	}

	return {{$tAlias.UpPlural}}.Query(
		ctx, exec,
		{{range $column := $table.Constraints.Primary.Columns -}}
		{{- $colAlias := $tAlias.Column $column -}}
		SelectWhere.{{$tAlias.UpPlural}}.{{$colAlias}}.EQ({{$colAlias}}PK),
		{{end -}}
		sm.Columns({{$tAlias.UpPlural}}.Columns().Only(cols...)),
	).One()
}

// {{$tAlias.UpSingular}}Exists checks the presence of a single record by primary key
func {{$tAlias.UpSingular}}Exists(ctx context.Context, exec bob.Executor, {{$pkArgs}}) (bool, error) {
	return {{$tAlias.UpPlural}}.Query(
		ctx, exec,
		{{range $column := $table.Constraints.Primary.Columns -}}
		{{- $colAlias := $tAlias.Column $column -}}
		SelectWhere.{{$tAlias.UpPlural}}.{{$colAlias}}.EQ({{$colAlias}}PK),
		{{end -}}
	).Exists()
}

{{- end}}

