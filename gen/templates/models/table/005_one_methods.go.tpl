{{$table := .Table}}
{{$tAlias := .Aliases.Table $table.Key -}}
{{$.Importer.Import "context"}}
{{$.Importer.Import "github.com/stephenafamo/bob"}}


// AfterQueryHook is called after {{$tAlias.UpSingular}} is retrieved from the database
func (o *{{$tAlias.UpSingular}}) AfterQueryHook(ctx context.Context, exec bob.Executor, queryType bob.QueryType) error {
  return {{$tAlias.UpSingular}}Slice{o}.AfterQueryHook(ctx, exec, queryType)
}

{{if .Table.Constraints.Primary -}}
{{$.Importer.Import "github.com/stephenafamo/bob/expr"}}
{{$.Importer.Import "github.com/stephenafamo/bob/orm"}}
{{$pkCols := $table.Constraints.Primary.Columns}}
var {{$tAlias.DownSingular}}RowMapping = orm.RowMapping[*{{$tAlias.UpSingular}}, {{$tAlias.UpSingular}}Slice]{
	KeyColumns: expr.NewColumnsExpr({{range $pkCols}}{{quote .}},{{end}}).WithParent({{quote (tableColumnAlias $table.Schema $table.Name $table.Key)}}),
	KeyValues: func(o *{{$tAlias.UpSingular}}) bob.Expression {
		return {{$tAlias.DownSingular}}Key({{range $pkCols}}o.{{$tAlias.Column .}},{{end}})
	},
	CopyColumns: (*{{$tAlias.UpSingular}}).copyColumns,
	CopyMatching: {{$tAlias.UpSingular}}Slice.copyMatchingRows,
}


{{block "one_update" . -}}
{{$table := .Table}}
{{$tAlias := .Aliases.Table $table.Key -}}
// Update uses an executor to update the {{$tAlias.UpSingular}}
func (o *{{$tAlias.UpSingular}}) Update(ctx context.Context, exec bob.Executor, s *{{$tAlias.UpSingular}}Setter) error {
	return {{$tAlias.UpPlural}}.UpdateRow(ctx, exec, o, s)
}
{{- end}}

// Delete deletes a single {{$tAlias.UpSingular}} record with an executor
func (o *{{$tAlias.UpSingular}}) Delete(ctx context.Context, exec bob.Executor) error {
	return {{$tAlias.UpSingular}}Slice{o}.DeleteAll(ctx, exec)
}

// Reload refreshes the {{$tAlias.UpSingular}} using the executor
func (o *{{$tAlias.UpSingular}}) Reload(ctx context.Context, exec bob.Executor) error {
	return {{$tAlias.UpPlural}}.ReloadRow(ctx, exec, o)
}

// Preserve relationships, counts, and plugin fields on refresh.
func (o *{{$tAlias.UpSingular}}) copyColumns(o2 *{{$tAlias.UpSingular}}) {
	{{range $column := $table.Columns -}}
	{{- $colAlias := $tAlias.Column $column.Name}}
	o.{{$colAlias}} = o2.{{$colAlias}}
	{{end}}
}

{{- end}}
