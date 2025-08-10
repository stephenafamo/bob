{{$table := .Table}}
{{$tAlias := .Aliases.Table $table.Key -}}
{{$.Importer.Import "context"}}
{{$.Importer.Import "github.com/stephenafamo/bob"}}


// AfterQueryHook is called after {{$tAlias.UpSingular}} is retrieved from the database
func (o *{{$tAlias.UpSingular}}) AfterQueryHook(ctx context.Context, exec bob.Executor, queryType bob.QueryType) error {
  var err error

  switch queryType {
  case bob.QueryTypeSelect:
    ctx, err = {{$tAlias.UpPlural}}.AfterSelectHooks.RunHooks(ctx, exec, {{$tAlias.UpSingular}}Slice{o})
  {{if .Table.Constraints.Primary -}}
    case bob.QueryTypeInsert:
      ctx, err = {{$tAlias.UpPlural}}.AfterInsertHooks.RunHooks(ctx, exec, {{$tAlias.UpSingular}}Slice{o})
    case bob.QueryTypeUpdate:
      ctx, err = {{$tAlias.UpPlural}}.AfterUpdateHooks.RunHooks(ctx, exec, {{$tAlias.UpSingular}}Slice{o})
    case bob.QueryTypeDelete:
      ctx, err = {{$tAlias.UpPlural}}.AfterDeleteHooks.RunHooks(ctx, exec, {{$tAlias.UpSingular}}Slice{o})
  {{- end}}
  }

	return err
}

{{if .Table.Constraints.Primary -}}
{{$.Importer.Import (printf "github.com/stephenafamo/bob/dialect/%s/dialect" $.Dialect)}}

// primaryKeyVals returns the primary key values of the {{$tAlias.UpSingular}} 
func (o *{{$tAlias.UpSingular}}) primaryKeyVals() bob.Expression {
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

{{$pkCols := $table.Constraints.Primary.Columns}}
{{$multiPK := gt (len $pkCols) 1}}
func (o *{{$tAlias.UpSingular}}) pkEQ() dialect.Expression {
   return {{if $multiPK}}{{$.Dialect}}.Group({{end}}{{- range $i, $col := $pkCols -}}{{if gt $i 0}}, {{end}}{{$.Dialect}}.Quote("{{$table.Key}}", "{{$col}}"){{end}}{{if $multiPK}}){{end -}}
    .EQ(bob.ExpressionFunc(func(ctx context.Context, w io.Writer, d bob.Dialect, start int) ([]any, error){
      return o.primaryKeyVals().WriteSQL(ctx, w, d, start)
    }))
}


{{block "one_update" . -}}
{{$table := .Table}}
{{$tAlias := .Aliases.Table $table.Key -}}
{{$.Importer.Import (printf "github.com/stephenafamo/bob/dialect/%s/um" $.Dialect)}}
// Update uses an executor to update the {{$tAlias.UpSingular}}
func (o *{{$tAlias.UpSingular}}) Update(ctx context.Context, exec bob.Executor, s *{{$tAlias.UpSingular}}Setter) error {
	v, err := {{$tAlias.UpPlural}}.Update(s.UpdateMod(), um.Where(o.pkEQ())).One(ctx, exec)
  if err != nil {
    return err
  }

	{{if $.Relationships.Get $table.Key}}o.R = v.R{{end}}
  *o = *v

  return nil
}
{{- end}}

{{$.Importer.Import (printf "github.com/stephenafamo/bob/dialect/%s/dm" $.Dialect)}}
// Delete deletes a single {{$tAlias.UpSingular}} record with an executor
func (o *{{$tAlias.UpSingular}}) Delete(ctx context.Context, exec bob.Executor) error {
	_, err := {{$tAlias.UpPlural}}.Delete(dm.Where(o.pkEQ())).Exec(ctx, exec)
  return err
}

// Reload refreshes the {{$tAlias.UpSingular}} using the executor
func (o *{{$tAlias.UpSingular}}) Reload(ctx context.Context, exec bob.Executor) error {
	o2, err := {{$tAlias.UpPlural}}.Query(
		{{range $column := $table.Constraints.Primary.Columns -}}
		{{- $colAlias := $tAlias.Column $column -}}
		SelectWhere.{{$tAlias.UpPlural}}.{{$colAlias}}.EQ(o.{{$colAlias}}),
		{{end -}}
	).One(ctx, exec)
	if err != nil {
		return err
	}
	{{if $.Relationships.Get $table.Key}}o2.R = o.R{{end}}
	*o = *o2

	return nil
}

{{- end}}
