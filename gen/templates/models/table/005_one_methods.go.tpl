{{$table := .Table}}
{{$tAlias := .Aliases.Table $table.Key -}}
{{$.Importer.Import "context"}}
{{$.Importer.Import "github.com/stephenafamo/bob"}}


// AfterQueryHook is called after {{$tAlias.UpSingular}} is retrieved from the database
func (o *{{$tAlias.UpSingular}}) AfterQueryHook(ctx context.Context, exec bob.Executor, queryType bob.QueryType) error {
  return {{$tAlias.UpSingular}}Slice{o}.AfterQueryHook(ctx, exec, queryType)
}

{{if .Table.Constraints.Primary -}}
{{$.Importer.Import "io"}}
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
   return {{if $multiPK}}{{$.Dialect}}.Group({{end}}{{- range $i, $col := $pkCols -}}{{if gt $i 0}}, {{end}}{{$.Dialect}}.Quote("{{tableColumnAlias $table.Schema $table.Name $table.Key}}", "{{$col}}"){{end}}{{if $multiPK}}){{end -}}
    .EQ(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error){
      return o.primaryKeyVals().WriteSQL(ctx, w, d, start)
    }))
}


{{block "one_update" . -}}
{{$table := .Table}}
{{$tAlias := .Aliases.Table $table.Key -}}
{{$.Importer.Import (printf "github.com/stephenafamo/bob/dialect/%s/um" $.Dialect)}}
// Update uses an executor to update the {{$tAlias.UpSingular}}
func (o *{{$tAlias.UpSingular}}) Update(ctx context.Context, exec bob.Executor, s *{{$tAlias.UpSingular}}Setter) error {
	q := {{$tAlias.UpPlural}}.Update(s.UpdateMod(), um.Where(o.pkEQ()))
	q.Expression.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
		return {{$tAlias.UpPlural}}.BeforeUpdateHooks.RunHooks(ctx, exec, {{$tAlias.UpSingular}}Slice{o})
	})
	v, err := q.One(ctx, exec)
  if err != nil {
    return err
  }

  o.copyColumns(v)
  return nil
}
{{- end}}

// Delete deletes a single {{$tAlias.UpSingular}} record with an executor
func (o *{{$tAlias.UpSingular}}) Delete(ctx context.Context, exec bob.Executor) error {
	return {{$tAlias.UpSingular}}Slice{o}.DeleteAll(ctx, exec)
}

// Reload refreshes the {{$tAlias.UpSingular}} using the executor
func (o *{{$tAlias.UpSingular}}) Reload(ctx context.Context, exec bob.Executor) error {
	o2, err := {{$tAlias.UpPlural}}.Query(
		{{range $column := $table.Constraints.Primary.Columns -}}
		{{- $colAlias := $tAlias.Column $column -}}
    sm.Where({{$tAlias.UpPlural}}.Columns.{{$colAlias}}.EQ({{$.Dialect}}.Arg(o.{{$colAlias}}))),
		{{end -}}
	).One(ctx, exec)
	if err != nil {
		return err
	}
	o.copyColumns(o2)
	return nil
}

// Preserve relationships, counts, and plugin fields on refresh.
func (o *{{$tAlias.UpSingular}}) copyColumns(o2 *{{$tAlias.UpSingular}}) {
	{{range $column := $table.Columns -}}
	{{- $colAlias := $tAlias.Column $column.Name}}
	o.{{$colAlias}} = o2.{{$colAlias}}
	{{end}}
}

{{- end}}
