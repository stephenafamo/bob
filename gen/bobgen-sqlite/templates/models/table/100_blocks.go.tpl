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

  vals := make([]bob.Expression, 0, len(q.TableRef.Columns))
  for _, col := range q.TableRef.Columns {
    switch col {
    {{range $column := $table.NonGeneratedColumns -}}
    case {{printf "%q" $column.Name}}:
      vals = append(vals, bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
        {{$colAlias := $tAlias.Column $column.Name -}}
        {{$colGetter := $.Types.FromOptional $.CurrentPackage $.Importer $column.Type (cat "s." $colAlias) $column.Nullable $column.Nullable -}}
        if {{$.Types.IsOptionalInvalid $.CurrentPackage $column.Type $column.Nullable (cat "s." $colAlias)}} {
          return {{$.Dialect}}.Arg(nil).WriteSQL(ctx, w, d, start)
        }
        return {{$.Dialect}}.Arg({{$colGetter}}).WriteSQL(ctx, w, d, start)
      }))
    {{end -}}
    }
  }

  q.AppendValues(vals...)
}
{{- end}}