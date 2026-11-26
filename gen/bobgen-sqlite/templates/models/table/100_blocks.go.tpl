{{define "setter_insert_mod" -}}
{{$.Importer.Import "github.com/stephenafamo/bob/orm/genruntime"}}
{{$.Importer.Import "github.com/stephenafamo/bob"}}
{{$table := .Table}}
{{$tAlias := .Aliases.Table $table.Key -}}
func (s *{{$tAlias.UpSingular}}Setter) Edit(q dialect.InsertEditor) {
  {{if $table.Constraints.Primary -}}
    q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
      return {{$tAlias.UpPlural}}.BeforeInsertHooks.RunHooks(ctx, exec, s)
    })
  {{end}}

  columns := q.ColumnNames()
  if len(columns) == 0 {
    columns = s.SetColumns()
    {{if $table.Constraints.Primary -}}
    if len(columns) == 0 {
      columns = {{printf "%#v" $table.Constraints.Primary.Columns}}
    }
    {{end}}
    q.SetTableColumns(columns...)
  }

  vals := make([]bob.Expression, 0, len(columns))
  for _, col := range columns {
    switch col {
    {{range $column := $table.NonGeneratedColumns -}}
    case {{printf "%q" $column.Name}}:
      vals = append(vals, genruntime.ValueOrNull(func() (any, bool) {
        {{$colAlias := $tAlias.Column $column.Name -}}
        {{$colGetter := $.Types.FromOptional $.CurrentPackage $.Importer $column.Type (cat "s." $colAlias) $column.Nullable $column.Nullable -}}
        if {{$.Types.IsOptionalInvalid $.CurrentPackage $column.Type $column.Nullable (cat "s." $colAlias)}} {
          return nil, false
        }
        return {{$colGetter}}, true
      }))
    {{end -}}
    }
  }

  q.AppendValues(vals...)
}
{{- end}}
