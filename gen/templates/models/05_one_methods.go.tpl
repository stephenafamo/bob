{{if .Table.PKey -}}
{{$.Importer.Import "context"}}
{{$table := .Table}}
{{$tAlias := .Aliases.Table $table.Name -}}

// Update uses an executor to update the {{$tAlias.UpSingular}}
func (o *{{$tAlias.UpSingular}}) Update(ctx context.Context, exec bob.Executor, cols []string) (int64, error) {
	rowsAff, err := {{$tAlias.UpPlural}}Table.Update(ctx, exec, cols, o)
	if err != nil {
		return rowsAff, err
	}

	return rowsAff, nil
}

// Delete deletes a single {{$tAlias.UpSingular}} record with an executor
func (o *{{$tAlias.UpSingular}}) Delete(ctx context.Context, exec bob.Executor) (int64, error) {
	return {{$tAlias.UpPlural}}Table.Delete(ctx, exec, o)
}

// Reload refreshes the {{$tAlias.UpSingular}} using the executor
func (o *{{$tAlias.UpSingular}}) Reload(ctx context.Context, exec bob.Executor) error {
	o2, err := {{$tAlias.UpPlural}}Table.Query(
		{{range $column := $table.PKey.Columns -}}
		{{- $colAlias := $tAlias.Column $column -}}
		SelectWhere.{{$tAlias.UpPlural}}.{{$colAlias}}.EQ(o.{{$colAlias}}),
		{{end -}}
	).One(ctx, exec)
	if err != nil {
		return err
	}
	{{if $table.Relationships}}o2.R = o.R{{end}}
	*o = *o2

	return nil
}

{{- end}}
