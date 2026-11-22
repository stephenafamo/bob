{{$table := .Table}}
{{$tAlias := .Aliases.Table $table.Key -}}
{{$.Importer.Import "context"}}
{{$.Importer.Import "github.com/stephenafamo/bob"}}


// AfterQueryHook is called after {{$tAlias.UpSingular}}Slice is retrieved from the database
func (o {{$tAlias.UpSingular}}Slice) AfterQueryHook(ctx context.Context, exec bob.Executor, queryType bob.QueryType) error {
  var err error

  switch queryType {
  case bob.QueryTypeSelect:
    _, err = {{$tAlias.UpPlural}}.AfterSelectHooks.RunHooks(ctx, exec, o)
  {{if .Table.Constraints.Primary -}}
    case bob.QueryTypeInsert:
      _, err = {{$tAlias.UpPlural}}.AfterInsertHooks.RunHooks(ctx, exec, o)
    case bob.QueryTypeUpdate:
      _, err = {{$tAlias.UpPlural}}.AfterUpdateHooks.RunHooks(ctx, exec, o)
    case bob.QueryTypeDelete:
      _, err = {{$tAlias.UpPlural}}.AfterDeleteHooks.RunHooks(ctx, exec, o)
    {{if eq $.Dialect "psql" -}}
    case bob.QueryTypeMerge:
      _, err = {{$tAlias.UpPlural}}.AfterMergeHooks.RunHooks(ctx, exec, o)
    {{- end}}
  {{- end}}
  }

	return err
}

{{if .Table.Constraints.Primary -}}
{{$.Importer.Import (printf "github.com/stephenafamo/bob/dialect/%s/sm" $.Dialect)}}
{{$.Importer.Import "github.com/stephenafamo/bob/orm"}}
{{$.Importer.Import (printf "github.com/stephenafamo/bob/dialect/%s/dialect" $.Dialect)}}

{{$pkCols := $table.Constraints.Primary.Columns}}
{{$multiPK := gt (len $pkCols) 1}}
func (o {{$tAlias.UpSingular}}Slice) pkIN() dialect.Expression {
  if len(o) == 0 {
    return {{$.Dialect}}.Raw("NULL")
  }

  return {{if $multiPK}}{{$.Dialect}}.Group({{end}}{{- range $i, $col := $pkCols -}}{{if gt $i 0}}, {{end}}{{$.Dialect}}.Quote("{{tableColumnAlias $table.Schema $table.Name $table.Key}}", "{{$col}}"){{end}}{{if $multiPK}}){{end -}}
    .In(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error){
      pkPairs := make([]bob.Expression, len(o))
      for i, row := range o {
        pkPairs[i] = row.primaryKeyVals()
      }
      return bob.ExpressSlice(ctx, w, d, start, pkPairs, "", ", ", "")
    }))
}

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
  orm.CopyByKey(o, from, func(row *{{$tAlias.UpSingular}}) {{$.Types.Get $.CurrentPackage $.Importer ($table.GetColumn $pkCol).Type}} { return row.{{$colAlias}} }, (*{{$tAlias.UpSingular}}).copyColumns)
  {{- else -}}
  for _, old := range o {
    for _, new := range from {
			{{range $column := $table.Constraints.Primary.Columns -}}
				{{- $colAlias := $tAlias.Column $column -}}
				{{- $typInfo :=  $.Types.Index ($table.GetColumn $column).Type -}}
        {{- with $typInfo.CompareExpr -}}
          {{$.Importer.ImportList $typInfo.CompareExprImports -}}
          if {{replace "AAA" (cat "new." $colAlias) . | replace "BBB" (cat "old." $colAlias)}}
        {{- else -}}
          if new.{{$colAlias}} != old.{{$colAlias}}
        {{- end -}}
        {
          continue
        }
      {{end -}}
      old.copyColumns(new)
      break
    }
  }
  {{- end}}
}


{{range $op := list "Update" "Delete" "Merge" -}}
{{if and (eq $op "Merge") (ne $.Dialect "psql")}}{{continue}}{{end -}}
// {{$op}}Mod installs the model hooks{{if ne $op "Merge"}} and filters by primary key{{end}}.
func (o {{$tAlias.UpSingular}}Slice) {{$op}}Mod() bob.Mod[*dialect.{{$op}}Query] {
  return bob.ModFunc[*dialect.{{$op}}Query](func(q *dialect.{{$op}}Query) {
    orm.ApplyModelHooks(q, o, o.copyMatchingRows, &{{$tAlias.UpPlural}}.Before{{$op}}Hooks, &{{$tAlias.UpPlural}}.After{{$op}}Hooks)
    {{if ne $op "Merge" -}}
    q.AppendWhere(o.pkIN())
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
  if len(o) == 0 {
    return nil
  }

	o2, err := {{$tAlias.UpPlural}}.Query(sm.Where(o.pkIN())).All(ctx, exec)
	if err != nil {
		return err
	}

  o.copyMatchingRows(o2...)

	return nil
}

{{- end}}
