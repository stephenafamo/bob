{{$table := .Table}}
{{$tAlias := .Aliases.Table $table.Name -}}

{{range $rel := $table.Relationships -}}
{{- $ftable := $.Aliases.Table $rel.Foreign -}}
{{- $relAlias := $tAlias.Relationship $rel.Name -}}
{{- if not $rel.IsToMany -}}
  func (o *{{$tAlias.UpSingular}}) Set{{$relAlias}}(ctx context.Context, exec bob.Executor,{{relDependencies $.Aliases $rel}} related *{{$ftable.UpSingular}}) {
  }

  {{if or $rel.ByJoinTable $rel.IsRemovable -}}
  func (o *{{$tAlias.UpSingular}}) Remove{{$relAlias}}(ctx context.Context, exec bob.Executor, related ...*{{$ftable.UpSingular}}) {
  }

  {{end -}}
{{else -}}
  func (o *{{$tAlias.UpSingular}}) Add{{$relAlias}}(ctx context.Context, exec bob.Executor,{{relDependencies $.Aliases $rel}} related ...*Optional{{$ftable.UpSingular}}) error {
    var err error

    {{if $rel.InsertEarly -}}
      rels, err := {{$ftable.UpPlural}}Table.InsertMany(ctx, exec, related...)
      if err != nil {
          return fmt.Errorf("inserting related objects: %w", err)
      }
			o.R.{{$relAlias}} = append(o.R.{{$relAlias}}, rels...)
    {{else -}}
      rels := related
    {{end}}

    {{$create := createDeps $.Aliases $rel}}
    {{$create}}

    for {{if $create}}i{{else}}_{{end}}, rel := range rels {
    {{setDeps $.Tables $.Aliases $rel false}}
    }

    {{insertDeps $.Aliases $rel}}

    {{if not $rel.InsertEarly -}}
      inserted, err := {{$ftable.UpPlural}}Table.InsertMany(ctx, exec, related...)
      if err != nil {
          return fmt.Errorf("inserting related objects: %w", err)
      }
			o.R.{{$relAlias}} = append(o.R.{{$relAlias}}, inserted...)
    {{end}}

    return nil
  }

  func (o *{{$tAlias.UpSingular}}) Attach{{$relAlias}}(ctx context.Context, exec bob.Executor,{{relDependencies $.Aliases $rel}} related ...*{{$ftable.UpSingular}}) error {
    var err error

    {{$create := createDeps $.Aliases $rel}}
    {{$create}}

    {{$attach := setDeps $.Tables $.Aliases $rel true}}
    {{with $attach}}
    for {{if $create}}i{{else}}_{{end}}, rel := range related {
      {{.}}
    }
    {{end}}

    {{insertDeps $.Aliases $rel}}

    {{$relatedVals := relatedUpdateValues $.Tables $.Aliases $rel true}}
    {{with $relatedVals}}
    _, err = {{$ftable.UpPlural}}Table.UpdateMany(
      ctx, exec, &Optional{{$ftable.UpSingular}}{
        {{.}}
      }, related...,
    )
    if err != nil {
        return fmt.Errorf("inserting related objects: %w", err)
    }
    {{end}}

		o.R.{{$relAlias}} = append(o.R.{{$relAlias}}, related...)
    return nil
  }

  {{if  $rel.IsRemovable -}}
  func (o *{{$tAlias.UpSingular}}) Detach{{$relAlias}}(ctx context.Context, exec bob.Executor, related ...*{{$ftable.UpSingular}}) {
  }
  {{end -}}
{{end -}}

{{end -}}

