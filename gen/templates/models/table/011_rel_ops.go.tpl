{{if and .Table.Constraints.Primary $.RelationshipMutationMethods -}}
{{$table := .Table}}
{{$tAlias := .Aliases.Table $table.Key -}}

{{if $.Relationships.Get $table.Key -}}
{{$.Importer.Import "fmt"}}
{{end}}

{{range $rel := $.Relationships.Get $table.Key -}}{{if not ($.AllTables.RelIsView $rel) -}}
{{- $ftable := $.Aliases.Table $rel.Foreign -}}
{{- $relAlias := $tAlias.Relationship $rel.Name -}}
{{- $invRel := $.Relationships.GetInverse . -}}
{{- $from := printf "%s%d" $tAlias.DownSingular $rel.LocalPosition}}
{{- $to := printf "%s%d" $ftable.DownSingular $rel.ForeignPosition}}
{{if $rel.NeedsMany $rel.ForeignPosition -}}
  {{- $to = printf "%s%d" $ftable.DownPlural $rel.ForeignPosition}}
{{- end}}
{{- $valuedSides := $rel.ValuedSides}}


{{range $index, $side := reverse $valuedSides -}}
  {{$sideTable := $.AllTables.Get $side.TableName}}
  {{$sideAlias := $.Aliases.Table $side.TableName}}


  {{if eq $rel.ForeignPosition $side.Position}}
  func insert{{$tAlias.UpSingular}}{{$relAlias}}{{$index}}(ctx context.Context, exec bob.Executor
  {{- if $rel.IsToMany -}}
    , {{$sideAlias.DownPlural}}{{$side.Position}} []*{{$.SetterType $side.TableName}}
  {{- else -}}
    , {{$to}} *{{$.SetterType $side.TableName}}
  {{- end -}}
  {{- range $map := $side.UniqueExternals -}}
    {{- $a := $.Aliases.Table .ExternalTable -}}
    {{- if $rel.NeedsMany .ExtPosition -}}
      , {{$a.DownPlural}}{{$map.ExtPosition}} {{$.SliceType .ExternalTable}}
    {{- else -}}
      , {{$a.DownSingular}}{{$map.ExtPosition}} *{{$.ModelType .ExternalTable}}
    {{- end -}}
  {{- end -}}
  ) ({{if $rel.IsToMany}}{{$.SliceType $side.TableName}}{{else}}*{{$.ModelType $side.TableName}}{{end}}, error) {
    {{$tblName := $to -}}
    {{if $rel.IsToMany -}}
      {{$tblName = printf "%s%d[i]" $ftable.DownPlural $rel.ForeignPosition -}}
      for i := range {{$ftable.DownPlural}}{{$rel.ForeignPosition}} {
    {{- end -}}
      {{range $map := $side.Mapped -}}
        {{$sideC := $sideTable.GetColumn .Column -}}
        {{$colName := $sideAlias.Column $map.Column -}}
        {{if .HasValue -}}
          {{$val := index .Value 1 -}}
          {{$tblName}}.{{$colName}} = {{$.Types.ToOptional $.CurrentPackage $.Importer $sideC.Type $val $sideC.Nullable false}}
        {{else}}
          {{$a := $.Aliases.Table .ExternalTable -}}
          {{$t := $.AllTables.Get .ExternalTable -}}
          {{$c := $t.GetColumn .ExternalColumn -}}
          {{$colVal := printf "%s%d" $a.DownSingular $map.ExtPosition -}}
          {{if $rel.NeedsMany .ExtPosition -}}
            {{$colVal = printf "%s%d[i]" $a.DownPlural $map.ExtPosition -}}
          {{end -}}
          {{$tblName}}.{{$colName}} = {{$.AllTables.ColumnAssigner $.CurrentPackage $.Importer $.Types $.Aliases $rel.Foreign .ExternalTable $map.Column .ExternalColumn $colVal true}}
        {{- end}}
      {{- end}}
    {{- if $rel.IsToMany}}}{{end}}

    {{if $rel.IsToMany -}}
      ret, err := {{$.TableVar $side.TableName}}.Insert(bob.ToMods({{$ftable.DownPlural}}{{$rel.ForeignPosition}}...)).All(ctx, exec)
    {{- else -}}
      ret, err := {{$.TableVar $side.TableName}}.Insert({{$to}}).One(ctx, exec)
    {{- end}}
    if err != nil {
        return ret, fmt.Errorf("insert{{$tAlias.UpSingular}}{{$relAlias}}{{$index}}: %w", err)
    }

    return ret, nil
  }
  {{end}}


  func attach{{$tAlias.UpSingular}}{{$relAlias}}{{$index}}(ctx context.Context, exec bob.Executor, count int
  {{- if not ($sideTable.IsJoinTableForRel $rel $side.Position) -}}
    {{- if $rel.NeedsMany $side.Position  -}}
      , {{$sideAlias.DownPlural}}{{$side.Position}} {{$.SliceType $side.TableName}}
    {{- else -}}
      , {{$sideAlias.DownSingular}}{{$side.Position}} *{{$.ModelType $side.TableName}}
    {{- end -}}
  {{- end -}}
  {{- range $map := $side.UniqueExternals -}}
    {{- $a := $.Aliases.Table .ExternalTable -}}
    {{- if $rel.NeedsMany .ExtPosition -}}
      , {{$a.DownPlural}}{{$map.ExtPosition}} {{$.SliceType .ExternalTable}}
    {{- else -}}
      , {{$a.DownSingular}}{{$map.ExtPosition}} *{{$.ModelType .ExternalTable}}
    {{- end -}}
  {{- end -}}
  ) ({{if $rel.NeedsMany $side.Position}}{{$.SliceType $side.TableName}}{{else}}*{{$.ModelType $side.TableName}}{{end}}, error) {
    {{- $uniqueEnd := and $side.End (not (index $rel.Sides (sub $side.Position 1)).ToUnique) -}}
    {{- $needsIndividualUpdate := (and ($rel.NeedsMany $side.Position) (not ($sideTable.IsJoinTableForRel $rel $side.Position)) (or (not $uniqueEnd) (not $.SliceMutationMethods))) -}}
    {{if $needsIndividualUpdate}}
    for i := range {{$sideAlias.DownPlural}}{{$side.Position}} {
      setter := &{{$.SetterType $side.TableName}}{
        {{range $map := $side.Mapped -}}
          {{$colName := $sideAlias.Column $map.Column -}}
          {{$sideColumn := $sideTable.GetColumn .Column -}}
          {{$tableAlias := $.Aliases.Table .ExternalTable -}}
          {{$table := $.AllTables.Get .ExternalTable -}}
          {{$column := $table.GetColumn .ExternalColumn -}}
          {{if .HasValue -}}
            {{$val := index .Value 1 -}}
            {{$colName}}: {{$.Types.ToOptional $.CurrentPackage $.Importer $sideColumn.Type $val $sideColumn.Nullable false}},
          {{else}}
            {{$colVal := printf "%s%d" $tableAlias.DownSingular $map.ExtPosition -}}
            {{if $rel.NeedsMany .ExtPosition -}}
              {{$colVal = printf "%s%d[i]" $tableAlias.DownPlural $map.ExtPosition -}}
            {{end -}}
            {{$colName}}: {{$.AllTables.ColumnAssigner $.CurrentPackage $.Importer $.Types $.Aliases $side.TableName $table.Key $map.Column .ExternalColumn $colVal true}},
          {{- end}}
        {{- end}}
      }

      err := {{$sideAlias.DownPlural}}{{$side.Position}}[i].Update(ctx, exec, setter)
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
    {{if and ($rel.NeedsMany $side.Position) ($sideTable.IsJoinTableForRel $rel $side.Position) -}}
      setters := make([]*{{$.SetterType $side.TableName}}, count)
      for i := range count {
        setters[i] = &{{$.SetterType $side.TableName}}{
    {{- else -}}
        setter := &{{$.SetterType $side.TableName}}{
    {{- end -}}
      {{range $map := $side.Mapped -}}
        {{if not (and $needsIndividualUpdate ($rel.NeedsMany .ExtPosition)) -}}
          {{$sideC := $sideTable.GetColumn .Column -}}
          {{$colName := $sideAlias.Column $map.Column -}}
          {{if .HasValue -}}
            {{$val := index .Value 1 -}}
            {{$colName}}: {{$.Types.ToOptional $.CurrentPackage $.Importer $sideC.Type $val $sideC.Nullable false}},
          {{else}}
            {{$a := $.Aliases.Table .ExternalTable -}}
            {{$t := $.AllTables.Get .ExternalTable -}}
            {{$c := $t.GetColumn .ExternalColumn -}}
            {{$colVal := printf "%s%d" $a.DownSingular $map.ExtPosition -}}
            {{if $rel.NeedsMany .ExtPosition -}}
              {{$colVal = printf "%s%d[i]" $a.DownPlural $map.ExtPosition -}}
            {{end -}}
            {{$colName}}: {{$.AllTables.ColumnAssigner $.CurrentPackage $.Importer $.Types $.Aliases $side.TableName .ExternalTable $map.Column .ExternalColumn $colVal true}},
          {{- end}}
        {{- end}}
      {{- end}}
    }
    {{if and ($rel.NeedsMany $side.Position) ($sideTable.IsJoinTableForRel $rel $side.Position) -}}}{{end}}

    {{if ($sideTable.IsJoinTableForRel $rel $side.Position) -}}
      {{if $rel.NeedsMany $side.Position -}}
        {{$sideAlias.DownPlural}}{{$side.Position}}, err := {{$.TableVar $side.TableName}}.Insert(bob.ToMods(setters...)).All(ctx, exec)
      {{- else -}}
        {{$sideAlias.DownSingular}}{{$side.Position}}, err := {{$.TableVar $side.TableName}}.Insert(setter).One(ctx, exec)
      {{- end}}
    {{- else -}}
      {{if $rel.NeedsMany $side.Position -}}
        err := {{$sideAlias.DownPlural}}{{$side.Position}}.UpdateAll(ctx, exec, *setter)
      {{- else -}}
        err := {{$sideAlias.DownSingular}}{{$side.Position}}.Update(ctx, exec, setter)
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
	  func ({{$from}} *{{$tAlias.UpSingular}}) Insert{{$relAlias}}(ctx context.Context, exec bob.Executor,{{$.RelDependenciesPos $rel}} related *{{$.SetterType $rel.Foreign}}) error {
	    var err error

	    {{if $rel.InsertEarly -}}
	      {{$to}}, err := {{$.TableVar $rel.Foreign}}.Insert(related).One(ctx, exec)
      if err != nil {
          return fmt.Errorf("inserting related objects: %w", err)
      }
	    {{end}}

	    {{range $index, $side := (reverse $valuedSides) -}}
	      {{$sideTable := $.AllTables.Get $side.TableName}}

	      {{if eq $side.Position $rel.ForeignPosition -}}
	        {{$to}}, err := insert{{$tAlias.UpSingular}}{{$relAlias}}{{$index}}(ctx, exec, related
	        {{- range $map := $side.UniqueExternals -}}
	          {{- $a := $.Aliases.Table .ExternalTable -}}
	          , {{$a.DownSingular}}{{$map.ExtPosition}}
	        {{- end -}}
	        )
      if err != nil {
	        return err
	      }
	      {{- end}}
	    {{- end}}

	    return {{$from}}.Attach{{$relAlias}}(ctx, exec,{{$.RelDependenciesPosArgs $rel}} {{$to}})
	  }

  func ({{$from}} *{{$tAlias.UpSingular}}) Attach{{$relAlias}}(ctx context.Context, exec bob.Executor,{{$.RelDependenciesPos $rel}} {{$to}} *{{$.ModelType $rel.Foreign}}) error {
    var err error

    {{range $index, $side := (reverse $valuedSides) -}}
      {{$sideTable := $.AllTables.Get $side.TableName}}
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
      {{- if and $show ($sideTable.IsJoinTableForRel $rel $side.Position) -}}:{{- end -}}
      = attach{{$tAlias.UpSingular}}{{$relAlias}}{{$index}}(ctx, exec, 1
      {{- if not ($sideTable.IsJoinTableForRel $rel $side.Position) -}}
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
	  func ({{$from}} *{{$tAlias.UpSingular}}) Insert{{$relAlias}}(ctx context.Context, exec bob.Executor,{{$.RelDependenciesPos $rel}} related ...*{{$.SetterType $rel.Foreign}}) error {
	    if len(related) == 0 {
	      return nil
	    }

    var err error

    {{if $rel.InsertEarly -}}
      inserted, err := {{$.TableVar $rel.Foreign}}.Insert(bob.ToMods(related...)).All(ctx, exec)
      if err != nil {
          return fmt.Errorf("inserting related objects: %w", err)
      }
      {{$to}} := {{$.SliceType $rel.Foreign}}(inserted)
    {{end}}

	    {{range $index, $side := (reverse $valuedSides) -}}
	      {{$sideTable := $.AllTables.Get $side.TableName}}
	      {{if eq $side.Position $rel.ForeignPosition -}}
	        {{$to}}, err := insert{{$tAlias.UpSingular}}{{$relAlias}}{{$index}}(ctx, exec, related
	      {{- range $map := $side.UniqueExternals -}}
	        {{- $a := $.Aliases.Table .ExternalTable -}}
	        , {{if $rel.NeedsMany .ExtPosition}}{{$a.DownPlural}}{{else}}{{$a.DownSingular}}{{end}}{{$map.ExtPosition}}
	      {{- end -}}
      )
      if err != nil {
	        return err
	      }
	      {{- end}}
	    {{- end}}

	    return {{$from}}.Attach{{$relAlias}}(ctx, exec,{{$.RelDependenciesPosArgs $rel}} {{$to}}...)
	  }


  func ({{$from}} *{{$tAlias.UpSingular}}) Attach{{$relAlias}}(ctx context.Context, exec bob.Executor,{{$.RelDependenciesPos $rel}} related ...*{{$.ModelType $rel.Foreign}}) error {
    if len(related) == 0 {
      return nil
    }

    var err error
    {{$to}} := {{$.SliceType $rel.Foreign}}(related)

    {{range $index, $side := (reverse $valuedSides) -}}
      {{$sideTable := $.AllTables.Get $side.TableName}}
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
      {{- if and $show ($sideTable.IsJoinTableForRel $rel $side.Position) -}}:{{- end -}}
      = attach{{$tAlias.UpSingular}}{{$relAlias}}{{$index}}(ctx, exec, len(related)
      {{- if not ($sideTable.IsJoinTableForRel $rel $side.Position) -}}
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
  func (o *{{$tAlias.UpSingular}}) Detach{{$relAlias}}(ctx context.Context, exec bob.Executor, related ...*{{$.ModelType $rel.Foreign}}) {
  }
  {{end -}}
{{end -}}

{{end -}}{{end -}}

{{- end}}
