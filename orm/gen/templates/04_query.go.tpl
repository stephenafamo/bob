{{$table := .Table}}
{{$tAlias := .Aliases.Table .Table.Name -}}
{{$.Importer.Import "github.com/stephenafamo/bob"}}


{{if not .Table.PKey -}}

// {{$tAlias.UpPlural}} begins a query on {{.Table.Name}}
func {{$tAlias.UpPlural}}(mods ...bob.Mod[*{{$.Dialect}}.SelectQuery]) *model.ViewQuery[*{{$tAlias.UpSingular}}, {{$tAlias.UpSingular}}Slice] {
	return {{$tAlias.UpPlural}}View.Query(mods...)
}

{{- else -}}

// {{$tAlias.UpPlural}} begins a query on {{.Table.Name}}
func {{$tAlias.UpPlural}}(mods ...bob.Mod[*{{$.Dialect}}.SelectQuery]) *model.TableQuery[*{{$tAlias.UpSingular}}, {{$tAlias.UpSingular}}Slice, *Optional{{$tAlias.UpSingular}}] {
	return {{$tAlias.UpPlural}}Table.Query(mods...)
}

{{$pkArgs := ""}}
{{range $colName := $table.PKey.Columns -}}
{{- $column := $table.GetColumn $colName -}}
{{- $colAlias := $tAlias.Column $colName -}}
{{$pkArgs = printf "%s%sPK %s," $pkArgs $colAlias $column.Type}}
{{end -}}

{{$.Importer.Import (printf "github.com/stephenafamo/bob/dialect/%s/select/qm" $.Dialect)}}
{{$.Importer.Import "github.com/stephenafamo/bob/orm"}}
// Find{{$tAlias.UpSingular}} retrieves a single record by primary key
// If cols is empty Find will return all columns.
func Find{{$tAlias.UpSingular}}(ctx context.Context, exec bob.Executor, {{$pkArgs}} cols ...string) (*{{$tAlias.UpSingular}}, error) {
	if len(cols) == 0 {
		return {{$tAlias.UpPlural}}Table.Query(
			{{range $column := $table.PKey.Columns -}}
			{{- $colAlias := $tAlias.Column $column -}}
			SelectWhere.{{$tAlias.UpPlural}}.{{$colAlias}}.EQ({{$colAlias}}PK),
			{{end -}}
		).One(ctx, exec)
	}

	return {{$tAlias.UpPlural}}Table.Query(
		{{range $column := $table.PKey.Columns -}}
		{{- $colAlias := $tAlias.Column $column -}}
		SelectWhere.{{$tAlias.UpPlural}}.{{$colAlias}}.EQ({{$colAlias}}PK),
		{{end -}}
		qm.Columns(orm.NewColumns(cols...)),
	).One(ctx, exec)
}

// {{$tAlias.UpSingular}}Exists checks the presence of a single record by primary key
func {{$tAlias.UpSingular}}Exists(ctx context.Context, exec bob.Executor, {{$pkArgs}}) (bool, error) {
	return {{$tAlias.UpPlural}}Table.Query(
		{{range $column := $table.PKey.Columns -}}
		{{- $colAlias := $tAlias.Column $column -}}
		SelectWhere.{{$tAlias.UpPlural}}.{{$colAlias}}.EQ({{$colAlias}}PK),
		{{end -}}
	).Exists(ctx, exec)
}

{{- end}}

