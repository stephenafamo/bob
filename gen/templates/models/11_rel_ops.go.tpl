{{$table := .Table}}
{{$tAlias := .Aliases.Table $table.Key -}}

{{range $rel := $table.Relationships -}}
{{- $ftable := $.Aliases.Table $rel.Foreign -}}
{{- $relAlias := $tAlias.Relationship $rel.Name -}}
{{- $invRel := $table.GetRelationshipInverse $.Tables . -}}
{{- if not $rel.IsToMany -}}
  func (o *{{$tAlias.UpSingular}}) Insert{{$relAlias}}(ctx context.Context, exec bob.Executor,{{relDependencies $.Aliases $rel}} related *Optional{{$ftable.UpSingular}}) error {
    {{if $rel.InsertEarly -}}
      rel, err := {{$ftable.UpPlural}}Table.Insert(ctx, exec, related)
      if err != nil {
          return fmt.Errorf("inserting related objects: %w", err)
      }
			o.R.{{$relAlias}} = rel
    {{else -}}
      rel := related
    {{end}}

    {{$create := createDeps $.Aliases $rel false}}
    {{$create}}

    {{setModelDeps $.Importer $.Tables $.Aliases $rel false true}}

    {{insertDeps $.Aliases $rel false}}

    {{if not $rel.InsertEarly -}}
      inserted, err := {{$ftable.UpPlural}}Table.Insert(ctx, exec, related)
      if err != nil {
          return fmt.Errorf("inserting related objects: %w", err)
      }
			o.R.{{$relAlias}} = inserted
    {{end}}

    {{if and (not $.NoBackReferencing) $invRel.Name -}}
    {{- $invAlias := $ftable.Relationship $invRel.Name -}}
      {{if $invRel.IsToMany -}}
        o.R.{{$relAlias}}.R.{{$invAlias}} = {{$tAlias.UpSingular}}Slice{o}
      {{else -}}
        o.R.{{$relAlias}}.R.{{$invAlias}} = o
      {{- end}}
    {{- end}}

    return nil
  }

  func (o *{{$tAlias.UpSingular}}) Attach{{$relAlias}}(ctx context.Context, exec bob.Executor,{{relDependencies $.Aliases $rel}} rel *{{$ftable.UpSingular}}) error {
    var err error

    {{$create := createDeps $.Aliases $rel false}}
    {{$create}}

    {{setModelDeps $.Importer $.Tables $.Aliases $rel false false}}

    {{insertDeps $.Aliases $rel false}}

    _, err = rel.Update(ctx, exec)
    if err != nil {
        return fmt.Errorf("inserting related objects: %w", err)
    }
    o.R.{{$relAlias}} = rel

    {{if and (not $.NoBackReferencing) $invRel.Name -}}
    {{- $invAlias := $ftable.Relationship $invRel.Name -}}
      {{if $invRel.IsToMany -}}
        rel.R.{{$invAlias}} = append(rel.R.{{$invAlias}}, o)
      {{else -}}
        rel.R.{{$invAlias}} = o
      {{- end}}
    {{- end}}

    return nil
  }

  {{if or $rel.ByJoinTable $rel.IsRemovable -}}
  func (o *{{$tAlias.UpSingular}}) Remove{{$relAlias}}(ctx context.Context, exec bob.Executor, related ...*{{$ftable.UpSingular}}) {
  }

  {{end -}}
{{else -}}
  func (o *{{$tAlias.UpSingular}}) Insert{{$relAlias}}(ctx context.Context, exec bob.Executor,{{relDependencies $.Aliases $rel}} related ...*Optional{{$ftable.UpSingular}}) error {
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

    {{$create := createDeps $.Aliases $rel true}}
    {{$create}}

    {{$set := setModelDeps $.Importer $.Tables $.Aliases $rel true true}}

    {{if or $create $set}}
    for {{if $create}}i{{else}}_{{end}}, rel := range rels {
      {{$set}}
    }
    {{end}}

    {{insertDeps $.Aliases $rel true}}

    {{if not $rel.InsertEarly -}}
      newRels, err := {{$ftable.UpPlural}}Table.InsertMany(ctx, exec, related...)
      if err != nil {
          return fmt.Errorf("inserting related objects: %w", err)
      }
			o.R.{{$relAlias}} = append(o.R.{{$relAlias}}, newRels...)
    {{- end}}

    {{if and (not $.NoBackReferencing) $invRel.Name -}}
    {{- $invAlias := $ftable.Relationship $invRel.Name -}}
      {{if $rel.InsertEarly -}}
        newRels := rels
      {{- end}}
      for _, rel := range newRels {
        {{if $invRel.IsToMany -}}
          rel.R.{{$invAlias}} = {{$tAlias.UpSingular}}Slice{o}
        {{else -}}
          rel.R.{{$invAlias}} = o
        {{- end}}
      }
    {{- end}}

    return nil
  }

  func (o *{{$tAlias.UpSingular}}) Attach{{$relAlias}}(ctx context.Context, exec bob.Executor,{{relDependencies $.Aliases $rel}} related ...*{{$ftable.UpSingular}}) error {
    {{$create := createDeps $.Aliases $rel true}}
    {{with $create}}
      var err error
      {{.}}
    {{end}}

    {{$set := setModelDeps $.Importer $.Tables $.Aliases $rel true false}}

    {{if or $create $set}}
    for {{if $create}}i{{else}}_{{end}}, rel := range related {
      {{$set}}
    }
    {{end}}

    {{insertDeps $.Aliases $rel true}}

    {{$relatedVals := relatedUpdateValues $.Importer $.Tables $.Aliases $rel true}}
    {{with $relatedVals}}
    if _, err := {{$ftable.UpPlural}}Table.UpdateMany(
      ctx, exec, &Optional{{$ftable.UpSingular}}{
        {{.}}
      }, related...,
    ); err != nil {
        return fmt.Errorf("inserting related objects: %w", err)
    }
    {{end}}

		o.R.{{$relAlias}} = append(o.R.{{$relAlias}}, related...)

    {{if and (not $.NoBackReferencing) $invRel.Name -}}
    {{- $invAlias := $ftable.Relationship $invRel.Name -}}
      for _, rel := range related {
        {{if $invRel.IsToMany -}}
          rel.R.{{$invAlias}} = append(rel.R.{{$invAlias}}, o)
        {{else -}}
          rel.R.{{$invAlias}} = o
        {{- end}}
      }
    {{- end}}

    return nil
  }

  {{if  $rel.IsRemovable -}}
  func (o *{{$tAlias.UpSingular}}) Detach{{$relAlias}}(ctx context.Context, exec bob.Executor, related ...*{{$ftable.UpSingular}}) {
  }
  {{end -}}
{{end -}}

{{end -}}

