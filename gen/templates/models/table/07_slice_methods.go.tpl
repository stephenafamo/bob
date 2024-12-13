{{$table := .Table}}
{{$tAlias := .Aliases.Table $table.Key -}}
{{$.Importer.Import "context"}}
{{$.Importer.Import "github.com/stephenafamo/bob"}}


// AfterQueryHook is called after {{$tAlias.UpSingular}}Slice is retrieved from the database
func (o {{$tAlias.UpSingular}}Slice) AfterQueryHook(ctx context.Context, exec bob.Executor, queryType bob.QueryType) error {
  var err error

  switch queryType {
  case bob.QueryTypeSelect:
    ctx, err = {{$tAlias.UpPlural}}.AfterSelectHooks.RunHooks(ctx, exec, o)
  {{if .Table.Constraints.Primary -}}
    case bob.QueryTypeInsert:
      ctx, err = {{$tAlias.UpPlural}}.AfterInsertHooks.RunHooks(ctx, exec, o)
    case bob.QueryTypeUpdate:
      ctx, err = {{$tAlias.UpPlural}}.AfterUpdateHooks.RunHooks(ctx, exec, o)
    case bob.QueryTypeDelete:
      ctx, err = {{$tAlias.UpPlural}}.AfterDeleteHooks.RunHooks(ctx, exec, o)
  {{- end}}
  }

	return err
}

{{if .Table.Constraints.Primary -}}
{{$.Importer.Import (printf "github.com/stephenafamo/bob/dialect/%s/sm" $.Dialect)}}
{{$.Importer.Import (printf "github.com/stephenafamo/bob/dialect/%s/dialect" $.Dialect)}}

{{$pkCols := $table.Constraints.Primary.Columns}}
{{$multiPK := gt (len $pkCols) 1}}
func (o {{$tAlias.UpSingular}}Slice) pkIN() dialect.Expression {
  if len(o) == 0 {
    return {{$.Dialect}}.Raw("NULL")
  }

  return {{if $multiPK}}{{$.Dialect}}.Group({{end}}{{- range $i, $col := $pkCols -}}{{if gt $i 0}}, {{end}}{{$.Dialect}}.Quote("{{$table.Key}}", "{{$col}}"){{end}}{{if $multiPK}}){{end -}}
    .In(bob.ExpressionFunc(func(ctx context.Context, w io.Writer, d bob.Dialect, start int) ([]any, error){
      pkPairs := make([]bob.Expression, len(o))
      for i, row := range o {
        pkPairs[i] = row.PrimaryKeyVals()
      }
      return bob.ExpressSlice(ctx, w, d, start, pkPairs, "", ", ", "")
    }))
}

// copyMatchingRows finds models in the given slice that have the same primary key
// then it first copies the existing relationships from the old model to the new model
// and then replaces the old model in the slice with the new model
func (o {{$tAlias.UpSingular}}Slice) copyMatchingRows(from ...*{{$tAlias.UpSingular}}) {
  for i, old := range o {
    for _, new := range from {
			{{range $column := $table.Constraints.Primary.Columns -}}
				{{- $colAlias := $tAlias.Column $column -}}
				{{- $typInfo :=  index $.Types ($table.GetColumn $column).Type -}}
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
      {{if $.Relationships.Get $table.Key}}new.R = old.R{{end}}
      o[i] = new
      break
    }
  }
}


// UpdateMod modifies an update query with "WHERE primary_key IN (o...)"
func (o {{$tAlias.UpSingular}}Slice) UpdateMod() bob.Mod[*dialect.UpdateQuery] {
  return bob.ModFunc[*dialect.UpdateQuery](func(q *dialect.UpdateQuery) {
    q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
      return {{$tAlias.UpPlural}}.BeforeUpdateHooks.RunHooks(ctx, exec, o)
    })

    q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
      var err error
      switch retrieved := retrieved.(type) {
      case *{{$tAlias.UpSingular}}:
        o.copyMatchingRows(retrieved)
      case []*{{$tAlias.UpSingular}}:
        o.copyMatchingRows(retrieved...)
      case {{$tAlias.UpSingular}}Slice:
        o.copyMatchingRows(retrieved...)
      default:
        // If the retrieved value is not a {{$tAlias.UpSingular}} or a slice of {{$tAlias.UpSingular}}
        // then run the AfterUpdateHooks on the slice
        _, err = {{$tAlias.UpPlural}}.AfterUpdateHooks.RunHooks(ctx, exec, o)
      }

      return err
    }))

    q.AppendWhere(o.pkIN())
  })
}

// DeleteMod modifies an delete query with "WHERE primary_key IN (o...)"
func (o {{$tAlias.UpSingular}}Slice) DeleteMod() bob.Mod[*dialect.DeleteQuery] {
  return bob.ModFunc[*dialect.DeleteQuery](func(q *dialect.DeleteQuery) {
    q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
      return {{$tAlias.UpPlural}}.BeforeDeleteHooks.RunHooks(ctx, exec, o)
    })

    q.AppendLoader(bob.LoaderFunc(func(ctx context.Context, exec bob.Executor, retrieved any) error {
      var err error
      switch retrieved := retrieved.(type) {
      case *{{$tAlias.UpSingular}}:
        o.copyMatchingRows(retrieved)
      case []*{{$tAlias.UpSingular}}:
        o.copyMatchingRows(retrieved...)
      case {{$tAlias.UpSingular}}Slice:
        o.copyMatchingRows(retrieved...)
      default:
        // If the retrieved value is not a {{$tAlias.UpSingular}} or a slice of {{$tAlias.UpSingular}}
        // then run the AfterDeleteHooks on the slice
        _, err = {{$tAlias.UpPlural}}.AfterDeleteHooks.RunHooks(ctx, exec, o)
      }

      return err
    }))

    q.AppendWhere(o.pkIN())
  })
}


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

