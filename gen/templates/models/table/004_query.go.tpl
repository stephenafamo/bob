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

{{$.Importer.Import (printf "github.com/stephenafamo/bob/dialect/%s/sm" $.Dialect)}}
// Find{{$tAlias.UpSingular}} retrieves a single record by primary key
// If cols is empty Find will return all columns.
func Find{{$tAlias.UpSingular}}(ctx context.Context, exec bob.Executor, {{$pkArgs}} cols ...string) (*{{$tAlias.UpSingular}}, error) {
	q := find{{$tAlias.UpSingular}}({{range $table.Constraints.Primary.Columns}}{{$tAlias.Column .}}PK,{{end}})
	if len(cols) != 0 { q.Apply(sm.Columns({{$tAlias.UpPlural}}.Columns.Only(cols...))) }
	return q.One(ctx, exec)
}

// {{$tAlias.UpSingular}}Exists checks the presence of a single record by primary key
func {{$tAlias.UpSingular}}Exists(ctx context.Context, exec bob.Executor, {{$pkArgs}}) (bool, error) {
	return find{{$tAlias.UpSingular}}({{range $table.Constraints.Primary.Columns}}{{$tAlias.Column .}}PK,{{end}}).Exists(ctx, exec)
}

func find{{$tAlias.UpSingular}}({{$pkArgs}}) {{$tAlias.UpPlural}}Query {
	return {{$tAlias.UpPlural}}.Query(
		{{range $column := $table.Constraints.Primary.Columns -}}
		{{- $colAlias := $tAlias.Column $column -}}
    sm.Where({{$tAlias.UpPlural}}.Columns.{{$colAlias}}.EQ({{$.Dialect}}.Arg({{$colAlias}}PK))),
		{{end -}}
	)
}

{{- end}}
