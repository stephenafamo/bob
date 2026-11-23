{{$table := .Table}}
{{$tAlias := .Aliases.Table $table.Key -}}
{{$.Importer.Import "context"}}
{{$.Importer.Import "github.com/stephenafamo/bob"}}


// AfterQueryHook is called after {{$tAlias.UpSingular}}Slice is retrieved from the database
func (o {{$tAlias.UpSingular}}Slice) AfterQueryHook(ctx context.Context, exec bob.Executor, queryType bob.QueryType) error {
	return {{$tAlias.UpPlural}}.RunAfterQueryHooks(ctx, exec, queryType, o)
}

{{if .Table.Constraints.Primary -}}
{{$.Importer.Import "github.com/stephenafamo/bob/orm/genruntime"}}
{{$.Importer.Import (printf "github.com/stephenafamo/bob/dialect/%s/dialect" $.Dialect)}}

{{$pkCols := $table.Constraints.Primary.Columns}}
{{$multiPK := gt (len $pkCols) 1}}

{{/* A single ==-comparable primary key column is matched in O(N+M) via a map;
     composite keys and custom compare_expr types fall back to the nested loop. */ -}}
{{- $pkCol := index $pkCols 0 -}}
{{- $useMap := and (not $multiPK) ($.Types.CanCompareWithEquals $.CurrentPackage ($table.GetColumn $pkCol).Type)}}
// copyMatchingRows finds models in the given slice that have the same primary key
// as one of the given models, then copies only the column fields over onto the
// model already in the slice. The slice keeps its original pointers, so every
// non-column field (the relationship cache .R, plugin fields such as the counts
// .C) survives, and callers holding a pointer to a member of the slice see the
// new column values.
func (o {{$tAlias.UpSingular}}Slice) copyMatchingRows(from ...*{{$tAlias.UpSingular}}) {
  {{if $useMap -}}
  {{$colAlias := $tAlias.Column $pkCol -}}
  genruntime.CopyByKey(o, from, func(row *{{$tAlias.UpSingular}}) {{$.Types.Get $.CurrentPackage $.Importer ($table.GetColumn $pkCol).Type}} { return row.{{$colAlias}} }, (*{{$tAlias.UpSingular}}).copyColumns)
  {{- else -}}
  genruntime.CopyByMatch(o, from, func(a, b *{{$tAlias.UpSingular}}) bool {
    return {{range $i, $name := $pkCols}}{{if $i}} && {{end}}
      {{- $col := $table.GetColumn $name -}}
      {{- $field := $tAlias.Column $name -}}
      {{- $cmp := $.Types.GetCompareExpr $.CurrentPackage $.Importer $col.Type $col.Nullable $col.Nullable -}}
      ({{replace "AAA" (printf "a.%s" $field) $cmp | replace "BBB" (printf "b.%s" $field)}})
      {{- end}}
  }, (*{{$tAlias.UpSingular}}).copyColumns)
  {{- end}}
}


{{range $op := list "Update" "Delete" "Merge" -}}
{{if and (eq $op "Merge") (ne $.Dialect "psql")}}{{continue}}{{end -}}
// {{$op}}Mod installs the model hooks{{if ne $op "Merge"}} and filters by primary key{{end}}.
func (o {{$tAlias.UpSingular}}Slice) {{$op}}Mod() bob.Mod[*dialect.{{$op}}Query] {
  return bob.ModFunc[*dialect.{{$op}}Query](func(q *dialect.{{$op}}Query) {
    genruntime.ApplyModelHooks(q, o, o.copyMatchingRows, &{{$tAlias.UpPlural}}.Before{{$op}}Hooks, &{{$tAlias.UpPlural}}.After{{$op}}Hooks)
    {{if ne $op "Merge" -}}
    q.AppendWhere({{$tAlias.DownSingular}}RowMapping.In(o))
    {{end -}}
  })
}
{{end}}

{{block "slice_update" . -}}
{{$table := .Table}}
{{$tAlias := .Aliases.Table $table.Key -}}
func (o {{$tAlias.UpSingular}}Slice) UpdateAll(ctx context.Context, exec bob.Executor, vals {{$tAlias.UpSingular}}Setter) error {
  if len(o) == 0 {
    return nil
  }

	_, err := {{$tAlias.UpPlural}}.Update(vals.UpdateMod(), o.UpdateMod()).All(ctx, exec)
  return err
}
{{- end}}

func (o {{$tAlias.UpSingular}}Slice) DeleteAll(ctx context.Context, exec bob.Executor) error {
  if len(o) == 0 {
    return nil
  }

	_, err := {{$tAlias.UpPlural}}.Delete(o.DeleteMod()).Exec(ctx, exec)
  return err
}


func (o {{$tAlias.UpSingular}}Slice) ReloadAll(ctx context.Context, exec bob.Executor) error {
	return {{$tAlias.UpPlural}}.ReloadRows(ctx, exec, o)
}

{{- end}}
