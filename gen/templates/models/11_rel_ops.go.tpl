{{if .Table.Constraints.Primary -}}
{{$table := .Table}}
{{$tAlias := .Aliases.Table $table.Key -}}

{{range $rel := $table.Relationships -}}{{if not (relIsView $.Tables $rel) -}}
{{- $ftable := $.Aliases.Table $rel.Foreign -}}
{{- $relAlias := $tAlias.Relationship $rel.Name -}}
{{- $invRel := $table.GetRelationshipInverse $.Tables . -}}
{{- $from := printf "%s%d" $tAlias.DownSingular $rel.LocalPosition}}
{{- $to := printf "%s%d" $ftable.DownSingular $rel.ForeignPosition}}
{{if $rel.NeedsMany $rel.ForeignPosition -}}
  {{- $to = printf "%s%d" $ftable.DownPlural $rel.ForeignPosition}}
{{- end}}


{{range $index, $side := reverse $rel.ValuedSides -}}
  {{$sideTable := getTable $.Tables $side.TableName}}
  {{$sideAlias := $.Aliases.Table $side.TableName}}


  {{if eq $rel.ForeignPosition $side.Position}}
  func insert{{$tAlias.UpSingular}}{{$relAlias}}{{$index}}(ctx context.Context, exec bob.Executor
  {{- if $rel.IsToMany -}}
    , {{$sideAlias.DownPlural}}{{$side.Position}} []*{{$sideAlias.UpSingular}}Setter
  {{- else -}}
    , {{$to}} *{{$sideAlias.UpSingular}}Setter
  {{- end -}}
  {{- range $map := $side.UniqueExternals -}}
    {{- $a := $.Aliases.Table .ExternalTable -}}
    , {{$a.DownSingular}}{{$map.ExtPosition}} *{{$a.UpSingular}}
  {{- end -}}
  ) ({{if $rel.IsToMany}}{{$sideAlias.UpSingular}}Slice{{else}}*{{$sideAlias.UpSingular}}{{end}}, error) {
    {{if $rel.IsToMany -}}
      for _, {{$to}} := range {{$ftable.DownPlural}}{{$rel.ForeignPosition}} {
    {{- end -}}
      {{range $map := $side.Mapped -}}
        {{$a := $.Aliases.Table .ExternalTable -}}
        {{$t := getTable $.Tables .ExternalTable -}}
        {{$c := $t.GetColumn .ExternalColumn -}}
        {{$sideC := $sideTable.GetColumn .Column -}}
        {{if and $sideC.Nullable $c.Nullable }}
          {{$.Importer.Import "github.com/aarondl/opt/omitnull" -}}
          {{$to}}.{{$sideAlias.Column $map.Column}} = omitnull.FromNull({{$a.DownSingular}}{{$map.ExtPosition}}.{{$a.Column $map.ExternalColumn}})
        {{else if $sideC.Nullable }}
          {{$.Importer.Import "github.com/aarondl/opt/omitnull" -}}
          {{$to}}.{{$sideAlias.Column $map.Column}} = omitnull.From({{$a.DownSingular}}{{$map.ExtPosition}}.{{$a.Column $map.ExternalColumn}})
        {{else if $c.Nullable}}
          {{$.Importer.Import "github.com/aarondl/opt/omit" -}}
          {{$to}}.{{$sideAlias.Column $map.Column}} = omit.From({{$a.DownSingular}}{{$map.ExtPosition}}.{{$a.Column $map.ExternalColumn}}.GetOrZero())
        {{else}}
          {{$.Importer.Import "github.com/aarondl/opt/omit" -}}
          {{$to}}.{{$sideAlias.Column $map.Column}} = omit.From({{$a.DownSingular}}{{$map.ExtPosition}}.{{$a.Column $map.ExternalColumn}})
        {{- end}}
      {{- end}}
    {{- if $rel.IsToMany}}}{{end}}

    {{if $rel.IsToMany -}}
      ret, err := {{$sideAlias.UpPlural}}.InsertMany(ctx, exec, {{$ftable.DownPlural}}{{$rel.ForeignPosition}}...)
    {{- else -}}
      ret, err := {{$sideAlias.UpPlural}}.Insert(ctx, exec, {{$to}})
    {{- end}}
    if err != nil {
        return ret, fmt.Errorf("insert{{$tAlias.UpSingular}}{{$relAlias}}{{$index}}: %w", err)
    }

    return ret, nil
  }
  {{end}}


  func attach{{$tAlias.UpSingular}}{{$relAlias}}{{$index}}(ctx context.Context, exec bob.Executor, count int
  {{- if not (isJoinTable $sideTable $rel $side.Position) -}}
    {{- if $rel.NeedsMany $side.Position  -}}
      , {{$sideAlias.DownPlural}}{{$side.Position}} {{$sideAlias.UpSingular}}Slice
    {{- else -}}
      , {{$sideAlias.DownSingular}}{{$side.Position}} *{{$sideAlias.UpSingular}}
    {{- end -}}
  {{- end -}}
  {{- range $map := $side.UniqueExternals -}}
    {{- $a := $.Aliases.Table .ExternalTable -}}
    {{- if $rel.NeedsMany .ExtPosition -}}
      , {{$a.DownPlural}}{{$map.ExtPosition}} {{$a.UpSingular}}Slice
    {{- else -}}
      , {{$a.DownSingular}}{{$map.ExtPosition}} *{{$a.UpSingular}}
    {{- end -}}
  {{- end -}}
  ) error {
    {{- $needsIndividualUpdate := (and (not $side.End) ($rel.NeedsMany $side.Position)  (not (isJoinTable $sideTable $rel $side.Position))) -}}
    {{if $needsIndividualUpdate}}
    for i := range {{$sideAlias.DownPlural}}{{$side.Position}} {
      setter := &{{$sideAlias.UpSingular}}Setter{
        {{range $map := $side.Mapped -}}
          {{if gt .ExtPosition $side.Position -}}
            {{$a := $.Aliases.Table .ExternalTable -}}
            {{$t := getTable $.Tables .ExternalTable -}}
            {{$c := $t.GetColumn .ExternalColumn -}}
            {{$sideC := $sideTable.GetColumn .Column -}}
            {{$colName := $sideAlias.Column $map.Column -}}
            {{$colVal := printf "%s%d.%s" $a.DownSingular $map.ExtPosition ($a.Column $map.ExternalColumn) -}}
            {{if $rel.NeedsMany .ExtPosition -}}
              {{$colVal = printf "%s%d[i].%s" $a.DownPlural $map.ExtPosition ($a.Column $map.ExternalColumn) -}}
            {{end}}
            {{if and $sideC.Nullable $c.Nullable -}}
              {{$.Importer.Import "github.com/aarondl/opt/omitnull"}}
              {{$colName}}: omitnull.FromNull({{$colVal}}),
            {{else if $sideC.Nullable -}}
              {{$.Importer.Import "github.com/aarondl/opt/omitnull"}}
              {{$colName}}: omitnull.From({{$colVal}}),
            {{else if $c.Nullable -}}
              {{$.Importer.Import "github.com/aarondl/opt/omit"}}
              {{$colName}}: omit.From({{$colVal}}),
            {{else -}}
              {{$.Importer.Import "github.com/aarondl/opt/omit"}}
              {{$colName}}: omit.From({{$colVal}}),
            {{- end}}
          {{- end}}
        {{- end}}
      }

      err := {{$sideAlias.UpPlural}}.Update(ctx, exec, setter, {{$sideAlias.DownPlural}}{{$side.Position}}[i])
      if err != nil {
          return fmt.Errorf("attach{{$tAlias.UpSingular}}{{$relAlias}}{{$index}}, update %d: %w", i, err)
      }
    }
    {{end}}

    {{if and ($rel.NeedsMany $side.Position) (isJoinTable $sideTable $rel $side.Position) -}}
      setters := make([]*{{$sideAlias.UpSingular}}Setter, count)
      for i := 0; i < count; i++ {
      {{if and ($rel.IsToMany) (ge $side.Position (sub $rel.ForeignPosition 1)) -}}
        {{$ftable.DownSingular}}{{$rel.ForeignPosition}} := {{$ftable.DownPlural}}{{$rel.ForeignPosition}}[i]
      {{end}}
        setters[i] = &{{$sideAlias.UpSingular}}Setter{
    {{- else -}}
        setter := &{{$sideAlias.UpSingular}}Setter{
    {{- end -}}
      {{range $map := $side.Mapped -}}
        {{if not (and $needsIndividualUpdate (gt .ExtPosition $side.Position)) -}}
          {{$a := $.Aliases.Table .ExternalTable -}}
          {{$t := getTable $.Tables .ExternalTable -}}
          {{$c := $t.GetColumn .ExternalColumn -}}
          {{$sideC := $sideTable.GetColumn .Column -}}
          {{$colName := $sideAlias.Column $map.Column -}}
          {{$colVal := printf "%s%d.%s" $a.DownSingular $map.ExtPosition ($a.Column $map.ExternalColumn) -}}
          {{if and $sideC.Nullable $c.Nullable -}}
            {{$.Importer.Import "github.com/aarondl/opt/omitnull"}}
            {{$colName}}: omitnull.FromNull({{$colVal}}),
          {{else if $sideC.Nullable -}}
            {{$.Importer.Import "github.com/aarondl/opt/omitnull"}}
            {{$colName}}: omitnull.From({{$colVal}}),
          {{else if $c.Nullable -}}
            {{$.Importer.Import "github.com/aarondl/opt/omit"}}
            {{$colName}}: omit.From({{$colVal}}),
          {{else -}}
            {{$.Importer.Import "github.com/aarondl/opt/omit"}}
            {{$colName}}: omit.From({{$colVal}}),
          {{- end}}
        {{- end}}
      {{- end}}
    }
    {{if and ($rel.NeedsMany $side.Position) (isJoinTable $sideTable $rel $side.Position) -}}}{{end}}

    {{if (isJoinTable $sideTable $rel $side.Position) -}}
      {{if $rel.NeedsMany $side.Position -}}
      _, err := {{$sideAlias.UpPlural}}.InsertMany(ctx, exec, setters...)
      {{- else -}}
      _, err := {{$sideAlias.UpPlural}}.Insert(ctx, exec, setter)
      {{- end}}
    {{- else -}}
      {{if $rel.NeedsMany $side.Position -}}
        err := {{$sideAlias.UpPlural}}.Update(ctx, exec, setter, {{$sideAlias.DownPlural}}{{$side.Position}}...)
      {{- else -}}
        err := {{$sideAlias.UpPlural}}.Update(ctx, exec, setter, {{$sideAlias.DownSingular}}{{$side.Position}})
      {{- end}}
    {{- end}}
    if err != nil {
        return fmt.Errorf("attach{{$tAlias.UpSingular}}{{$relAlias}}{{$index}}: %w", err)
    }

    return nil
  }


{{end}}

{{if not $rel.IsToMany -}}
  func ({{$from}} *{{$tAlias.UpSingular}}) Insert{{$relAlias}}(ctx context.Context, exec bob.Executor,{{relDependenciesPos $.Tables $.Aliases $rel}} related *{{$ftable.UpSingular}}Setter) error {
    {{if $rel.InsertEarly -}}
      {{$to}}, err := {{$ftable.UpPlural}}.Insert(ctx, exec, related)
      if err != nil {
          return fmt.Errorf("inserting related objects: %w", err)
      }
    {{end}}

    {{range $index, $side := (reverse $rel.ValuedSides) -}}
      {{$sideTable := getTable $.Tables $side.TableName}}
      {{$sideAlias := $.Aliases.Table $side.TableName}}

      {{if eq $side.Position $rel.ForeignPosition -}}
        {{$to}}, err := insert{{$tAlias.UpSingular}}{{$relAlias}}{{$index}}(ctx, exec, related
      {{- else -}}
        err = attach{{$tAlias.UpSingular}}{{$relAlias}}{{$index}}(ctx, exec, 1
        {{- if not (isJoinTable $sideTable $rel $side.Position) -}}
          , {{$sideAlias.DownSingular}}{{$side.Position}}
        {{- end -}}
      {{- end}}
        {{- range $map := $side.UniqueExternals -}}
          {{- $a := $.Aliases.Table .ExternalTable -}}
          , {{$a.DownSingular}}{{$map.ExtPosition}}
        {{- end -}}
        )
      if err != nil {
        return err
      }
    {{- end}}


    {{$from}}.R.{{$relAlias}} = {{$to}}

    {{if and (not $.NoBackReferencing) $invRel.Name -}}
    {{- $invAlias := $ftable.Relationship $invRel.Name -}}
      {{if $invRel.IsToMany -}}
        {{$to}}.R.{{$invAlias}} = append({{$to}}.R.{{$invAlias}}, {{$from}})
      {{- else -}}
        {{$to}}.R.{{$invAlias}} = {{$from}}
      {{- end}}
    {{- end}}

    return nil
  }

  func ({{$from}} *{{$tAlias.UpSingular}}) Attach{{$relAlias}}(ctx context.Context, exec bob.Executor,{{relDependenciesPos $.Tables $.Aliases $rel}} {{$to}} *{{$ftable.UpSingular}}) error {
    var err error

    {{range $index, $side := (reverse $rel.ValuedSides) -}}
      {{$sideTable := getTable $.Tables $side.TableName}}
      {{$sideAlias := $.Aliases.Table $side.TableName}}
      err = attach{{$tAlias.UpSingular}}{{$relAlias}}{{$index}}(ctx, exec, 1
      {{- if not (isJoinTable $sideTable $rel $side.Position) -}}
        , {{$sideAlias.DownSingular}}{{$side.Position}}
      {{- end -}}
      {{- range $map := $side.UniqueExternals -}}
        {{- $a := $.Aliases.Table .ExternalTable -}}
        , {{$a.DownSingular}}{{$map.ExtPosition}}
      {{- end -}}
      )
      if err != nil {
        return err
      }
    {{- end}}


    {{$from}}.R.{{$relAlias}} = {{$to}}

    {{if and (not $.NoBackReferencing) $invRel.Name -}}
    {{- $invAlias := $ftable.Relationship $invRel.Name -}}
      {{if $invRel.IsToMany -}}
        {{$to}}.R.{{$invAlias}} = append({{$to}}.R.{{$invAlias}}, {{$from}})
      {{- else -}}
        {{$to}}.R.{{$invAlias}} = {{$from}}
      {{- end}}
    {{- end}}

    return nil
  }

{{else -}}
  func ({{$from}} *{{$tAlias.UpSingular}}) Insert{{$relAlias}}(ctx context.Context, exec bob.Executor,{{relDependenciesPos $.Tables $.Aliases $rel}} related ...*{{$ftable.UpSingular}}Setter) error {
    if len(related) == 0 {
      return nil
    }

    {{if $rel.InsertEarly -}}
      inserted, err := {{$ftable.UpPlural}}.InsertMany(ctx, exec, related...)
      if err != nil {
          return fmt.Errorf("inserting related objects: %w", err)
      }
      {{$to}} := {{$ftable.UpSingular}}Slice(inserted)
    {{end}}

    {{range $index, $side := (reverse $rel.ValuedSides) -}}
      {{$sideTable := getTable $.Tables $side.TableName}}
      {{$sideAlias := $.Aliases.Table $side.TableName}}
      {{if eq $side.Position $rel.ForeignPosition -}}
        {{$to}}, err := insert{{$tAlias.UpSingular}}{{$relAlias}}{{$index}}(ctx, exec, related
      {{- else -}}
        err = attach{{$tAlias.UpSingular}}{{$relAlias}}{{$index}}(ctx, exec, len(related)
        {{- if not (isJoinTable $sideTable $rel $side.Position) -}}
          , {{if $rel.NeedsMany $side.Position}}{{$sideAlias.DownPlural}}{{else}}{{$sideAlias.DownSingular}}{{end}}{{$side.Position}}
        {{- end -}}
      {{- end}}
      {{- range $map := $side.UniqueExternals -}}
        {{- $a := $.Aliases.Table .ExternalTable -}}
        , {{if $rel.NeedsMany .ExtPosition}}{{$a.DownPlural}}{{else}}{{$a.DownSingular}}{{end}}{{$map.ExtPosition}}
      {{- end -}}
      )
      if err != nil {
        return err
      }
    {{- end}}


    {{$from}}.R.{{$relAlias}} = append({{$from}}.R.{{$relAlias}}, {{$to}}...)

    {{if and (not $.NoBackReferencing) $invRel.Name -}}
    {{- $invAlias := $ftable.Relationship $invRel.Name -}}
      for _, rel := range {{$to}} {
        {{if $invRel.IsToMany -}}
          rel.R.{{$invAlias}} = append(rel.R.{{$invAlias}}, {{$from}})
        {{- else -}}
          rel.R.{{$invAlias}} = {{$from}}
        {{- end}}
      }
    {{- end}}
    return nil
  }


  func ({{$from}} *{{$tAlias.UpSingular}}) Attach{{$relAlias}}(ctx context.Context, exec bob.Executor,{{relDependenciesPos $.Tables $.Aliases $rel}} related ...*{{$ftable.UpSingular}}) error {
    if len(related) == 0 {
      return nil
    }

    var err error
    {{$to}} := {{$ftable.UpSingular}}Slice(related)

    {{range $index, $side := (reverse $rel.ValuedSides) -}}
      {{$sideTable := getTable $.Tables $side.TableName}}
      {{$sideAlias := $.Aliases.Table $side.TableName}}
      err = attach{{$tAlias.UpSingular}}{{$relAlias}}{{$index}}(ctx, exec, len(related)
      {{- if not (isJoinTable $sideTable $rel $side.Position) -}}
        , {{if $rel.NeedsMany $side.Position}}{{$sideAlias.DownPlural}}{{else}}{{$sideAlias.DownSingular}}{{end}}{{$side.Position}}
      {{- end -}}
      {{- range $map := $side.UniqueExternals -}}
        {{- $a := $.Aliases.Table .ExternalTable -}}
        , {{if $rel.NeedsMany .ExtPosition}}{{$a.DownPlural}}{{else}}{{$a.DownSingular}}{{end}}{{$map.ExtPosition}}
      {{- end -}}
      )
      if err != nil {
        return err
      }
    {{- end}}


    {{$from}}.R.{{$relAlias}} = append({{$from}}.R.{{$relAlias}}, {{$to}}...)

    {{if and (not $.NoBackReferencing) $invRel.Name -}}
    {{- $invAlias := $ftable.Relationship $invRel.Name -}}
      for _, rel := range related {
        {{if $invRel.IsToMany -}}
          rel.R.{{$invAlias}} = append(rel.R.{{$invAlias}}, {{$from}})
        {{- else -}}
          rel.R.{{$invAlias}} = {{$from}}
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

{{end -}}{{end -}}

{{- end}}
