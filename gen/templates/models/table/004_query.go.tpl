{{$table := .Table}}
{{$tAlias := .Aliases.Table .Table.Key -}}

{{if .Table.Constraints.Primary -}}
{{$pkArgs := ""}}
{{range $colName := $table.Constraints.Primary.Columns -}}
{{- $column := $table.GetColumn $colName -}}
{{- $colAlias := $tAlias.Column $colName -}}
{{- $colTyp := $.Types.Get $.CurrentPackage $.Importer $column.Type -}}
{{$pkArgs = printf "%s%sPK %s," $pkArgs $colAlias $colTyp}}
{{end -}}

// Find{{$tAlias.UpSingular}} retrieves a single record by primary key
// If cols is empty Find will return all columns.
func Find{{$tAlias.UpSingular}}(ctx context.Context, exec bob.Executor, {{$pkArgs}} cols ...string) (*{{$tAlias.UpSingular}}, error) {
	return {{$tAlias.UpPlural}}.FindByKey(ctx, exec, {{$tAlias.DownSingular}}Key({{range $table.Constraints.Primary.Columns}}{{$tAlias.Column .}}PK,{{end}}), cols...)
}

// {{$tAlias.UpSingular}}Exists checks the presence of a single record by primary key
func {{$tAlias.UpSingular}}Exists(ctx context.Context, exec bob.Executor, {{$pkArgs}}) (bool, error) {
	return {{$tAlias.UpPlural}}.ExistsByKey(ctx, exec, {{$tAlias.DownSingular}}Key({{range $table.Constraints.Primary.Columns}}{{$tAlias.Column .}}PK,{{end}}))
}

func {{$tAlias.DownSingular}}Key({{$pkArgs}}) bob.Expression {
	return {{$.Dialect}}.{{if gt (len $table.Constraints.Primary.Columns) 1}}ArgGroup{{else}}Arg{{end}}({{range $table.Constraints.Primary.Columns}}{{$tAlias.Column .}}PK,{{end}})
}

{{- end}}
