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

  if len(q.TableRef.Columns) == 0 {
    q.TableRef.Columns = s.SetColumns()
    {{if $table.Constraints.Primary -}}
    if len(q.TableRef.Columns) == 0 {
      q.TableRef.Columns = {{printf "%#v" $table.Constraints.Primary.Columns}}
    }
    {{end}}
  }

	q.AppendValues(bob.ExpressionFunc(func(ctx context.Context, w io.Writer, d bob.Dialect, start int) ([]any, error){
    vals := make([]bob.Expression, 0, {{len $table.NonGeneratedColumns}})
    {{range $index, $column := $table.NonGeneratedColumns -}}
      {{$colAlias := $tAlias.Column $column.Name -}}
      {{$colGetter := $.Types.FromOptional $.CurrentPackage $.Importer $column.Type (cat "s." $colAlias) $column.Nullable $column.Nullable -}}
      if {{$.Types.IsOptionalValid $.CurrentPackage $column.Type $column.Nullable (cat "s." $colAlias)}} {
        vals = append(vals, {{$.Dialect}}.Arg({{$colGetter}}))
      }

    {{end -}}

    {{if $table.Constraints.Primary -}}
    if len(vals) == 0 {
      vals = append(vals{{range $table.Constraints.Primary.Columns}}, {{$.Dialect}}.Arg(nil){{end}})
    }
    {{end}}

    return bob.ExpressSlice(ctx, w, d, start, vals, "", ", ", "")
  }))
}
{{- end}}
