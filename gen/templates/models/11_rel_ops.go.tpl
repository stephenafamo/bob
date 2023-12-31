{{if .Table.Constraints.Primary -}}
{{$table := .Table}}
{{$tAlias := .Aliases.Table $table.Key -}}

{{range $rel := $.Relationships.Get $table.Key -}}{{if not (relIsView $.Tables $rel) -}}
{{- $ftable := $.Aliases.Table $rel.Foreign -}}
{{- $relAlias := $tAlias.Relationship $rel.Name -}}
{{- $invRel := $.Relationships.GetInverse $.Tables . -}}
{{- $from := printf "%s%d" $tAlias.DownSingular $rel.LocalPosition}}
{{- $to := printf "%s%d" $ftable.DownSingular $rel.ForeignPosition}}
{{if $rel.NeedsMany $rel.ForeignPosition -}}
  {{- $to = printf "%s%d" $ftable.DownPlural $rel.ForeignPosition}}
{{- end}}
{{- $valuedSides := $rel.ValuedSides}}


{{range $index, $side := reverse $valuedSides -}}
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
    {{- if $rel.NeedsMany .ExtPosition -}}
      , {{$a.DownPlural}}{{$map.ExtPosition}} {{$a.UpSingular}}Slice
    {{- else -}}
      , {{$a.DownSingular}}{{$map.ExtPosition}} *{{$a.UpSingular}}
    {{- end -}}
  {{- end -}}
  ) ({{if $rel.IsToMany}}{{$sideAlias.UpSingular}}Slice{{else}}*{{$sideAlias.UpSingular}}{{end}}, error) {
    {{$tblName := $to -}}
    {{if $rel.IsToMany -}}
      {{$tblName = printf "%s%d[i]" $ftable.DownPlural $rel.ForeignPosition -}}
      for i := range {{$ftable.DownPlural}}{{$rel.ForeignPosition}} {
    {{- end -}}
      {{range $map := $side.Mapped -}}
        {{$sideC := $sideTable.GetColumn .Column -}}
        {{$colName := $sideAlias.Column $map.Column -}}
        {{if .HasValue -}}
          {{if $sideC.Nullable }}
            {{$.Importer.Import "github.com/aarondl/opt/omitnull" -}}
            {{$tblName}}.{{$colName}} = omitnull.From({{index .Value 1}})
          {{else}}
            {{$.Importer.Import "github.com/aarondl/opt/omit" -}}
            {{$tblName}}.{{$colName}} = omit.From({{index .Value 1}})
          {{end}}
        {{- else -}}
          {{$a := $.Aliases.Table .ExternalTable -}}
          {{$t := getTable $.Tables .ExternalTable -}}
          {{$c := $t.GetColumn .ExternalColumn -}}
          {{$colVal := printf "%s%d.%s" $a.DownSingular $map.ExtPosition ($a.Column $map.ExternalColumn) -}}
          {{if $rel.NeedsMany .ExtPosition -}}
            {{$colVal = printf "%s%d[i].%s" $a.DownPlural $map.ExtPosition ($a.Column $map.ExternalColumn) -}}
          {{end -}}
          {{if and $sideC.Nullable $c.Nullable }}
            {{$.Importer.Import "github.com/aarondl/opt/omitnull" -}}
            {{$tblName}}.{{$colName}} = omitnull.FromNull({{$colVal}})
          {{else if $sideC.Nullable }}
            {{$.Importer.Import "github.com/aarondl/opt/omitnull" -}}
            {{$tblName}}.{{$colName}} = omitnull.From({{$colVal}})
          {{else if $c.Nullable}}
            {{$.Importer.Import "github.com/aarondl/opt/omit" -}}
            {{$tblName}}.{{$colName}} = omit.FromCond({{$colVal}}.GetOrZero(), {{$colVal}}.IsSet())
          {{else}}
            {{$.Importer.Import "github.com/aarondl/opt/omit" -}}
            {{$tblName}}.{{$colName}} = omit.From({{$colVal}})
          {{- end}}
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
  ) ({{if $rel.NeedsMany $side.Position}}{{$sideAlias.UpSingular}}Slice{{else}}*{{$sideAlias.UpSingular}}{{end}}, error) {
    {{- $uniqueEnd := and $side.End (not (index $rel.Sides (sub $side.Position 1)).ToUnique) -}}
    {{- $needsIndividualUpdate := (and (not $uniqueEnd) ($rel.NeedsMany $side.Position)  (not (isJoinTable $sideTable $rel $side.Position))) -}}
    {{if $needsIndividualUpdate}}
    for i := range {{$sideAlias.DownPlural}}{{$side.Position}} {
      setter := &{{$sideAlias.UpSingular}}Setter{
        {{range $map := $side.Mapped -}}
          {{if $rel.NeedsMany .ExtPosition -}}
            {{$sideC := $sideTable.GetColumn .Column -}}
            {{$colName := $sideAlias.Column $map.Column -}}
            {{if .HasValue -}}
              {{if $sideC.Nullable }}
                {{$.Importer.Import "github.com/aarondl/opt/omitnull" -}}
                {{$colName}}: omitnull.From({{index .Value 1}}),
              {{else}}
                {{$.Importer.Import "github.com/aarondl/opt/omit" -}}
                {{$colName}}: omit.From({{index .Value 1}}),
              {{end}}
            {{- else -}}
              {{$a := $.Aliases.Table .ExternalTable -}}
              {{$t := getTable $.Tables .ExternalTable -}}
              {{$c := $t.GetColumn .ExternalColumn -}}
              {{$colVal := printf "%s%d.%s" $a.DownSingular $map.ExtPosition ($a.Column $map.ExternalColumn) -}}
              {{if $rel.NeedsMany .ExtPosition -}}
                {{$colVal = printf "%s%d[i].%s" $a.DownPlural $map.ExtPosition ($a.Column $map.ExternalColumn) -}}
              {{end -}}
              {{if and $sideC.Nullable $c.Nullable -}}
                {{$.Importer.Import "github.com/aarondl/opt/omitnull"}}
                {{$colName}}: omitnull.FromNull({{$colVal}}),
              {{else if $sideC.Nullable -}}
                {{$.Importer.Import "github.com/aarondl/opt/omitnull"}}
                {{$colName}}: omitnull.From({{$colVal}}),
              {{else if $c.Nullable -}}
                {{$.Importer.Import "github.com/aarondl/opt/omit"}}
                {{$colName}}: omit.FromCond({{$colVal}}.GetOrZero(), {{$colVal}}.IsSet()),
              {{else -}}
                {{$.Importer.Import "github.com/aarondl/opt/omit"}}
                {{$colName}}: omit.From({{$colVal}}),
              {{- end}}
            {{- end}}
          {{- end}}
        {{- end}}
      }

      err := {{$sideAlias.UpPlural}}.Update(ctx, exec, setter, {{$sideAlias.DownPlural}}{{$side.Position}}[i])
      if err != nil {
          return nil, fmt.Errorf("attach{{$tAlias.UpSingular}}{{$relAlias}}{{$index}}, update %d: %w", i, err)
      }
    }
    {{end}}

    {{$needsBulkUpdate := false}}
    {{range $map := $side.Mapped -}}
        {{if not (or $needsIndividualUpdate ($rel.NeedsMany .ExtPosition)) -}}
          {{$needsBulkUpdate = true}}{{break}}
        {{end}}
    {{end}}

    {{if $needsBulkUpdate -}}
    {{if and ($rel.NeedsMany $side.Position) (isJoinTable $sideTable $rel $side.Position) -}}
      setters := make([]*{{$sideAlias.UpSingular}}Setter, count)
      for i := 0; i < count; i++ {
        setters[i] = &{{$sideAlias.UpSingular}}Setter{
    {{- else -}}
        setter := &{{$sideAlias.UpSingular}}Setter{
    {{- end -}}
      {{range $map := $side.Mapped -}}
        {{if not (and $needsIndividualUpdate ($rel.NeedsMany .ExtPosition)) -}}
          {{$sideC := $sideTable.GetColumn .Column -}}
          {{$colName := $sideAlias.Column $map.Column -}}
          {{if .HasValue -}}
            {{if $sideC.Nullable }}
              {{$.Importer.Import "github.com/aarondl/opt/omitnull" -}}
              {{$colName}}: omitnull.From({{index .Value 1}}),
            {{else}}
              {{$.Importer.Import "github.com/aarondl/opt/omit" -}}
              {{$colName}}: omit.From({{index .Value 1}}),
            {{end}}
          {{- else -}}
            {{$a := $.Aliases.Table .ExternalTable -}}
            {{$t := getTable $.Tables .ExternalTable -}}
            {{$c := $t.GetColumn .ExternalColumn -}}
            {{$colVal := printf "%s%d.%s" $a.DownSingular $map.ExtPosition ($a.Column $map.ExternalColumn) -}}
            {{if $rel.NeedsMany .ExtPosition -}}
              {{$colVal = printf "%s%d[i].%s" $a.DownPlural $map.ExtPosition ($a.Column $map.ExternalColumn) -}}
            {{end -}}
            {{if and $sideC.Nullable $c.Nullable -}}
              {{$.Importer.Import "github.com/aarondl/opt/omitnull"}}
              {{$colName}}: omitnull.FromNull({{$colVal}}),
            {{else if $sideC.Nullable -}}
              {{$.Importer.Import "github.com/aarondl/opt/omitnull"}}
              {{$colName}}: omitnull.From({{$colVal}}),
            {{else if $c.Nullable -}}
              {{$.Importer.Import "github.com/aarondl/opt/omit"}}
              {{$colName}}: omit.FromCond({{$colVal}}.GetOrZero(), {{$colVal}}.IsSet()),
            {{else -}}
              {{$.Importer.Import "github.com/aarondl/opt/omit"}}
              {{$colName}}: omit.From({{$colVal}}),
            {{- end}}
          {{- end}}
        {{- end}}
      {{- end}}
    }
    {{if and ($rel.NeedsMany $side.Position) (isJoinTable $sideTable $rel $side.Position) -}}}{{end}}

    {{if (isJoinTable $sideTable $rel $side.Position) -}}
      {{if $rel.NeedsMany $side.Position -}}
        {{$sideAlias.DownPlural}}{{$side.Position}}, err := {{$sideAlias.UpPlural}}.InsertMany(ctx, exec, setters...)
      {{- else -}}
        {{$sideAlias.DownSingular}}{{$side.Position}}, err := {{$sideAlias.UpPlural}}.Insert(ctx, exec, setter)
      {{- end}}
    {{- else -}}
      {{if $rel.NeedsMany $side.Position -}}
        err := {{$sideAlias.UpPlural}}.Update(ctx, exec, setter, {{$sideAlias.DownPlural}}{{$side.Position}}...)
      {{- else -}}
        err := {{$sideAlias.UpPlural}}.Update(ctx, exec, setter, {{$sideAlias.DownSingular}}{{$side.Position}})
      {{- end}}
    {{- end}}
    if err != nil {
        return nil, fmt.Errorf("attach{{$tAlias.UpSingular}}{{$relAlias}}{{$index}}: %w", err)
    }
    {{end}}

    {{if $rel.NeedsMany $side.Position}}
      return {{$sideAlias.DownPlural}}{{$side.Position}}, nil
    {{else}}
      return {{$sideAlias.DownSingular}}{{$side.Position}}, nil
    {{end}}
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

    {{range $index, $side := (reverse $valuedSides) -}}
      {{$sideTable := getTable $.Tables $side.TableName}}
      {{$sideAlias := $.Aliases.Table $side.TableName}}

      {{if eq $side.Position $rel.ForeignPosition -}}
        {{$to}}, err := insert{{$tAlias.UpSingular}}{{$relAlias}}{{$index}}(ctx, exec, related
      {{- else -}}
        {{$show := lt $index (len $valuedSides | add -1)}}
        {{if $show -}}
          {{if $rel.NeedsMany $side.Position -}}
            {{$sideAlias.DownPlural}}{{$side.Position}}, err
          {{- else -}}
            {{$sideAlias.DownSingular}}{{$side.Position}}, err
          {{- end -}}
        {{- else -}}
          _, err
        {{- end -}}
        {{- if and $show (isJoinTable $sideTable $rel $side.Position) -}}:{{- end -}}
        = attach{{$tAlias.UpSingular}}{{$relAlias}}{{$index}}(ctx, exec, 1
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

    {{range $index, $side := (reverse $valuedSides) -}}
      {{$sideTable := getTable $.Tables $side.TableName}}
      {{$sideAlias := $.Aliases.Table $side.TableName}}
      {{$show := lt $index (len $valuedSides | add -1)}}
      {{if $show -}}
        {{if $rel.NeedsMany $side.Position -}}
          {{$sideAlias.DownPlural}}{{$side.Position}}, err
        {{- else -}}
          {{$sideAlias.DownSingular}}{{$side.Position}}, err
        {{- end -}}
      {{- else -}}
        _, err
      {{- end -}}
      {{- if and $show (isJoinTable $sideTable $rel $side.Position) -}}:{{- end -}}
      = attach{{$tAlias.UpSingular}}{{$relAlias}}{{$index}}(ctx, exec, 1
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

    {{range $index, $side := (reverse $valuedSides) -}}
      {{$sideTable := getTable $.Tables $side.TableName}}
      {{$sideAlias := $.Aliases.Table $side.TableName}}
      {{if eq $side.Position $rel.ForeignPosition -}}
        {{$to}}, err := insert{{$tAlias.UpSingular}}{{$relAlias}}{{$index}}(ctx, exec, related
      {{- else -}}
        {{$show := lt $index (len $valuedSides | add -1)}}
        {{if $show -}}
          {{if $rel.NeedsMany $side.Position -}}
            {{$sideAlias.DownPlural}}{{$side.Position}}, err
          {{- else -}}
            {{$sideAlias.DownSingular}}{{$side.Position}}, err
          {{- end -}}
        {{- else -}}
          _, err
        {{- end -}}
        {{- if and $show (isJoinTable $sideTable $rel $side.Position) -}}:{{- end -}}
        = attach{{$tAlias.UpSingular}}{{$relAlias}}{{$index}}(ctx, exec, len(related)
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

    {{range $index, $side := (reverse $valuedSides) -}}
      {{$sideTable := getTable $.Tables $side.TableName}}
      {{$sideAlias := $.Aliases.Table $side.TableName}}
      {{$show := lt $index (len $valuedSides | add -1)}}
      {{if $show -}}
        {{if $rel.NeedsMany $side.Position -}}
          {{$sideAlias.DownPlural}}{{$side.Position}}, err
        {{- else -}}
          {{$sideAlias.DownSingular}}{{$side.Position}}, err
        {{- end -}}
      {{- else -}}
        _, err
      {{- end -}}
      {{- if and $show (isJoinTable $sideTable $rel $side.Position) -}}:{{- end -}}
      = attach{{$tAlias.UpSingular}}{{$relAlias}}{{$index}}(ctx, exec, len(related)
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
