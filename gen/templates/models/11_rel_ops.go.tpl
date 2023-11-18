{{if .Table.PKey -}}
{{$table := .Table}}
{{$tAlias := .Aliases.Table $table.Key -}}

{{range $rel := $table.Relationships -}}{{if not (relIsView $.Tables $rel) -}}
{{- $ftable := $.Aliases.Table $rel.Foreign -}}
{{- $relAlias := $tAlias.Relationship $rel.Name -}}
{{- $invRel := $table.GetRelationshipInverse $.Tables . -}}
{{- $from := printf "%s%d" $tAlias.DownSingular $rel.LocalPosition}}
{{- $to := printf "%s%d" $ftable.DownSingular $rel.ForeignPosition}}

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


  func attach{{$tAlias.UpSingular}}{{$relAlias}}{{$index}}(ctx context.Context, exec bob.Executor
  {{- if not $sideTable.IsJoinTable -}}
    {{- if and ($rel.IsToMany) (eq $side.Position $rel.ForeignPosition) -}}
      , {{$sideAlias.DownPlural}}{{$side.Position}} {{$sideAlias.UpSingular}}Slice
    {{- else -}}
      , {{$sideAlias.DownSingular}}{{$side.Position}} *{{$sideAlias.UpSingular}}
    {{- end -}}
  {{- end -}}
  {{- range $map := $side.UniqueExternals -}}
    {{- $a := $.Aliases.Table .ExternalTable -}}
    {{- if and ($rel.IsToMany) (eq .ExtPosition $rel.ForeignPosition) -}}
      , {{$a.DownPlural}}{{$map.ExtPosition}} {{$a.UpSingular}}Slice
    {{- else -}}
      , {{$a.DownSingular}}{{$map.ExtPosition}} *{{$a.UpSingular}}
    {{- end -}}
  {{- end -}}
  ) error {
    {{- if and $rel.IsToMany $sideTable.IsJoinTable}}
      setters := make([]*{{$sideAlias.UpSingular}}Setter, len({{$ftable.DownPlural}}{{$rel.ForeignPosition}}))
      for i, {{$ftable.DownSingular}}{{$rel.ForeignPosition}} := range {{$ftable.DownPlural}}{{$rel.ForeignPosition}} {
        setters[i] = &{{$sideAlias.UpSingular}}Setter{
    {{- else -}}
        setter := &{{$sideAlias.UpSingular}}Setter{
    {{- end -}}
      {{range $map := $side.Mapped -}}
        {{$a := $.Aliases.Table .ExternalTable -}}
        {{$t := getTable $.Tables .ExternalTable -}}
        {{$c := $t.GetColumn .ExternalColumn -}}
        {{$sideC := $sideTable.GetColumn .Column -}}
        {{if and $sideC.Nullable $c.Nullable -}}
          {{$.Importer.Import "github.com/aarondl/opt/omitnull"}}
          {{$sideAlias.Column $map.Column}}: omitnull.FromNull({{$a.DownSingular}}{{$map.ExtPosition}}.{{$a.Column $map.ExternalColumn}}),
        {{else if $sideC.Nullable -}}
          {{$.Importer.Import "github.com/aarondl/opt/omitnull"}}
          {{$sideAlias.Column $map.Column}}: omitnull.From({{$a.DownSingular}}{{$map.ExtPosition}}.{{$a.Column $map.ExternalColumn}}),
        {{else if $c.Nullable -}}
          {{$.Importer.Import "github.com/aarondl/opt/omit"}}
          {{$sideAlias.Column $map.Column}}: omit.From({{$a.DownSingular}}{{$map.ExtPosition}}.{{$a.Column $map.ExternalColumn}}.GetOrZero()),
        {{else -}}
          {{$.Importer.Import "github.com/aarondl/opt/omit"}}
          {{$sideAlias.Column $map.Column}}: omit.From({{$a.DownSingular}}{{$map.ExtPosition}}.{{$a.Column $map.ExternalColumn}}),
        {{- end}}
      {{- end}}
    }
    {{if and $rel.IsToMany $sideTable.IsJoinTable}}}{{end}}

    {{if $sideTable.IsJoinTable -}}
      {{if $rel.IsToMany -}}
      _, err := {{$sideAlias.UpPlural}}.InsertMany(ctx, exec, setters...)
      {{- else -}}
      _, err := {{$sideAlias.UpPlural}}.Insert(ctx, exec, setter)
      {{- end}}
    {{- else -}}
      {{if and ($rel.IsToMany) (eq $side.TableName $rel.Foreign) -}}
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
  func ({{$from}} *{{$tAlias.UpSingular}}) Insert{{$relAlias}}(ctx context.Context, exec bob.Executor,{{relDependenciesPos $.Aliases $rel}} related *{{$ftable.UpSingular}}Setter) error {
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
        err = attach{{$tAlias.UpSingular}}{{$relAlias}}{{$index}}(ctx, exec
        {{- if not $sideTable.IsJoinTable -}}
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

  func ({{$from}} *{{$tAlias.UpSingular}}) Attach{{$relAlias}}(ctx context.Context, exec bob.Executor,{{relDependenciesPos $.Aliases $rel}} {{$to}} *{{$ftable.UpSingular}}) error {
    var err error

    {{range $index, $side := (reverse $rel.ValuedSides) -}}
      {{$sideTable := getTable $.Tables $side.TableName}}
      {{$sideAlias := $.Aliases.Table $side.TableName}}
      err = attach{{$tAlias.UpSingular}}{{$relAlias}}{{$index}}(ctx, exec
      {{- if not $sideTable.IsJoinTable -}}
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

  {{if or $rel.ByJoinTable $rel.IsRemovable -}}
  func (o *{{$tAlias.UpSingular}}) Remove{{$relAlias}}(ctx context.Context, exec bob.Executor, related ...*{{$ftable.UpSingular}}) {
  }

  {{end -}}
{{else -}}
  func ({{$from}} *{{$tAlias.UpSingular}}) Insert{{$relAlias}}(ctx context.Context, exec bob.Executor,{{relDependenciesPos $.Aliases $rel}} related ...*{{$ftable.UpSingular}}Setter) error {
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
        err = attach{{$tAlias.UpSingular}}{{$relAlias}}{{$index}}(ctx, exec
        {{- if not $sideTable.IsJoinTable -}}
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

  func ({{$from}} *{{$tAlias.UpSingular}}) Attach{{$relAlias}}(ctx context.Context, exec bob.Executor,{{relDependenciesPos $.Aliases $rel}} related ...*{{$ftable.UpSingular}}) error {
    if len(related) == 0 {
      return nil
    }

    var err error
    {{$to}} := {{$ftable.UpSingular}}Slice(related)

    {{range $index, $side := (reverse $rel.ValuedSides) -}}
      {{$sideTable := getTable $.Tables $side.TableName}}
      {{$sideAlias := $.Aliases.Table $side.TableName}}
      err = attach{{$tAlias.UpSingular}}{{$relAlias}}{{$index}}(ctx, exec
      {{- if not $sideTable.IsJoinTable -}}
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
