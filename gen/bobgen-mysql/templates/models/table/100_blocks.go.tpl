{{define "model_and_query" -}}
{{$table := .Table}}
{{$tAlias := .Aliases.Table $table.Key -}}
{{if not $table.Constraints.Primary -}}
	// {{$tAlias.UpPlural}} contains methods to work with the {{$table.Name}} view
	var {{$tAlias.UpPlural}} = {{$.Dialect}}.NewViewx[*{{$tAlias.UpSingular}}, {{$tAlias.UpSingular}}Slice]("{{$table.Name}}")
	// {{$tAlias.UpPlural}}Query is a query on the {{$table.Name}} view
	type {{$tAlias.UpPlural}}Query = *{{$.Dialect}}.ViewQuery[*{{$tAlias.UpSingular}}, {{$tAlias.UpSingular}}Slice]
{{- else -}}
	// {{$tAlias.UpPlural}} contains methods to work with the {{$table.Name}} table
	var {{$tAlias.UpPlural}} = {{$.Dialect}}.NewTablex[*{{$tAlias.UpSingular}}, {{$tAlias.UpSingular}}Slice, *{{$tAlias.UpSingular}}Setter]("{{$table.Name}}", {{$table.UniqueColPairs}})
	// {{$tAlias.UpPlural}}Query is a query on the {{$table.Name}} table
	type {{$tAlias.UpPlural}}Query = *{{$.Dialect}}.ViewQuery[*{{$tAlias.UpSingular}}, {{$tAlias.UpSingular}}Slice]
{{- end}}
{{- end}}


{{define "setter_insert_mod" -}}
{{$.Importer.Import "io"}}
{{$.Importer.Import "github.com/stephenafamo/bob"}}
{{$table := .Table}}
{{$tAlias := .Aliases.Table $table.Key -}}
func (s *{{$tAlias.UpSingular}}Setter) Apply(q *dialect.InsertQuery) {
  {{if $table.Constraints.Primary -}}
    q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
      return {{$tAlias.UpPlural}}.BeforeInsertHooks.RunHooks(ctx, exec, s)
    })
  {{end}}

	q.AppendValues(
  {{range $index, $column := $table.NonGeneratedColumns -}}
    bob.ExpressionFunc(func(ctx context.Context, w io.Writer, d bob.Dialect, start int) ([]any, error){
        {{$colAlias := $tAlias.Column $column.Name -}}
        {{$colGetter := $.Types.FromOptional $.CurrentPackage $.Importer $column.Type (cat "s." $colAlias) $column.Nullable $column.Nullable -}}
        if !{{$.Types.IsOptionalValid $.CurrentPackage $column.Type $column.Nullable (cat "s." $colAlias)}} {
          return {{$.Dialect}}.Raw("DEFAULT").WriteSQL(ctx, w, d, start)
        }
        return {{$.Dialect}}.Arg({{$colGetter}}).WriteSQL(ctx, w, d, start)
    }),
  {{- end}})
}
{{- end}}


{{define "setter_update_mod" -}}
{{$table := .Table}}
{{$tAlias := .Aliases.Table $table.Key -}}
func (s {{$tAlias.UpSingular}}Setter) UpdateMod() bob.Mod[*dialect.UpdateQuery] {
  return um.Set(s.Expressions("{{$table.Name}}")...)
}
{{- end}}

{{define "one_update" -}}
{{$table := .Table}}
{{$tAlias := .Aliases.Table $table.Key -}}
{{$.Importer.Import (printf "github.com/stephenafamo/bob/dialect/%s/um" $.Dialect)}}
// Update uses an executor to update the {{$tAlias.UpSingular}}
func (o *{{$tAlias.UpSingular}}) Update(ctx context.Context, exec bob.Executor, s *{{$tAlias.UpSingular}}Setter) error {
	_, err := {{$tAlias.UpPlural}}.Update(s.UpdateMod(), um.Where(o.pkEQ())).Exec(ctx, exec)
  if err != nil {
    return err
  }

  s.Overwrite(o)

  return nil
}
{{- end}}

{{define "slice_update" -}}
{{$table := .Table}}
{{$tAlias := .Aliases.Table $table.Key -}}
func (o {{$tAlias.UpSingular}}Slice) UpdateAll(ctx context.Context, exec bob.Executor, vals {{$tAlias.UpSingular}}Setter) error {
	_, err := {{$tAlias.UpPlural}}.Update(vals.UpdateMod(), o.UpdateMod()).Exec(ctx, exec)

  for i := range o {
    vals.Overwrite(o[i]) 
  }

  return err
}
{{- end}}
