{{if .Table.Constraints.Primary -}}
{{$.Importer.Import "context"}}
{{$.Importer.Import (printf "github.com/stephenafamo/bob/dialect/%s/dialect" $.Dialect)}}
{{$table := .Table}}
{{$tAlias := .Aliases.Table $table.Key -}}

// PrimaryKeyVals returns the primary key values of the {{$tAlias.UpSingular}} 
func (o *{{$tAlias.UpSingular}}) PrimaryKeyVals() bob.Expression {
	{{if gt (len $table.Constraints.Primary.Columns) 1 -}}
		return {{$.Dialect}}.ArgGroup(
			{{range $column := $table.Constraints.Primary.Columns -}}
				o.{{$tAlias.Column $column}},
			{{end}}
		)
	{{- else -}}
		return {{$.Dialect}}.Arg(o.{{$tAlias.Column (index $table.Constraints.Primary.Columns 0)}})
	{{- end}}
}

// Update uses an executor to update the {{$tAlias.UpSingular}}
func (o *{{$tAlias.UpSingular}}) Update(ctx context.Context, exec bob.Executor, s *{{$tAlias.UpSingular}}Setter) error {
	return {{$tAlias.UpPlural}}.Update(ctx, exec, s, o)
}

// Delete deletes a single {{$tAlias.UpSingular}} record with an executor
func (o *{{$tAlias.UpSingular}}) Delete(ctx context.Context, exec bob.Executor) error {
	return {{$tAlias.UpPlural}}.Delete(ctx, exec, o)
}

// Reload refreshes the {{$tAlias.UpSingular}} using the executor
func (o *{{$tAlias.UpSingular}}) Reload(ctx context.Context, exec bob.Executor) error {
	o2, err := {{$tAlias.UpPlural}}.Query(
		ctx, exec,
		{{range $column := $table.Constraints.Primary.Columns -}}
		{{- $colAlias := $tAlias.Column $column -}}
		SelectWhere.{{$tAlias.UpPlural}}.{{$colAlias}}.EQ(o.{{$colAlias}}),
		{{end -}}
	).One()
	if err != nil {
		return err
	}
	{{if $.Relationships.Get $table.Key}}o2.R = o.R{{end}}
	*o = *o2

	return nil
}

{{- end}}
