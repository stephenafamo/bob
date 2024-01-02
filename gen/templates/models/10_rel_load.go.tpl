{{$table := .Table}}
{{$tAlias := .Aliases.Table $table.Key -}}

{{if $.Relationships.Get $table.Key -}}
{{$.Importer.Import "fmt" -}}
{{$.Importer.Import "context" -}}
{{$.Importer.Import "database/sql" -}}
{{$.Importer.Import "errors" -}}
{{$.Importer.Import (printf "github.com/stephenafamo/bob/dialect/%s/sm" $.Dialect) -}}
func (o *{{$tAlias.UpSingular}}) Preload(name string, retrieved any) error {
	if o == nil {
		return nil
	}

	switch name {
	{{range $.Relationships.Get $table.Key -}}
	{{- $fAlias := $.Aliases.Table .Foreign -}}
	{{- $relAlias := $tAlias.Relationship .Name -}}
	{{- $invRel := $.Relationships.GetInverse $.Tables . -}}
	case "{{$relAlias}}":
		{{if .IsToMany -}}
			rels, ok := retrieved.({{$fAlias.UpSingular}}Slice)
			if !ok {
				return fmt.Errorf("{{$tAlias.DownSingular}} cannot load %T as %q", retrieved, name)
			}

			o.R.{{$relAlias}} = rels

			{{if and (not $.NoBackReferencing) $invRel.Name -}}
			{{- $invAlias := $fAlias.Relationship $invRel.Name -}}
			for _, rel := range rels {
				if rel != nil {
					{{if $invRel.IsToMany -}}
						rel.R.{{$invAlias}} = {{$tAlias.UpSingular}}Slice{o}
					{{- else -}}
						rel.R.{{$invAlias}} =  o
					{{- end}}
				}
			}
			{{- end}}
			return nil
		{{else -}}
			rel, ok := retrieved.(*{{$fAlias.UpSingular}})
			if !ok {
				return fmt.Errorf("{{$tAlias.DownSingular}} cannot load %T as %q", retrieved, name)
			}

			o.R.{{$relAlias}} = rel

			{{if and (not $.NoBackReferencing) $invRel.Name -}}
			{{- $invAlias := $fAlias.Relationship $invRel.Name -}}
				if rel != nil {
					{{if $invRel.IsToMany -}}
						rel.R.{{$invAlias}} = {{$tAlias.UpSingular}}Slice{o}
					{{- else -}}
						rel.R.{{$invAlias}} =  o
					{{- end}}
				}
			{{- end}}
			return nil
		{{end -}}

	{{end -}}
	default:
		return fmt.Errorf("{{$tAlias.DownSingular}} has no relationship %q", name)
	}
}
{{- end}}


{{range $rel := $.Relationships.Get $table.Key -}}
{{- $isToView := relIsView $.Tables $rel -}}
{{- $fAlias := $.Aliases.Table $rel.Foreign -}}
{{- $relAlias := $tAlias.Relationship $rel.Name -}}
{{- $invRel := $.Relationships.GetInverse $.Tables . -}}
{{- if not $rel.IsToMany -}}
{{$.Importer.Import "github.com/stephenafamo/bob/orm"}}
func Preload{{$tAlias.UpSingular}}{{$relAlias}}(opts ...{{$.Dialect}}.PreloadOption) {{$.Dialect}}.Preloader {
	return {{$.Dialect}}.Preload[*{{$fAlias.UpSingular}}, {{$fAlias.UpSingular}}Slice](orm.Relationship{
			Name: "{{$relAlias}}",
			Sides:  []orm.RelSide{
				{{- $toTable := $table }}{{/* To be able to access the last one after the loop */}}
				{{range $side := $rel.Sides -}}
				{{- $from := $.Aliases.Table $side.From -}}
				{{- $to := $.Aliases.Table $side.To -}}
				{{- $fromTable := getTable $.Tables $side.From -}}
				{{- $toTable = getTable $.Tables $side.To -}}
				{
					From: {{quote $fromTable.Key}},
					To: TableNames.{{$to.UpPlural}},
					ToExpr: func(ctx context.Context) bob.Expression {
					  return {{$to.UpPlural}}.Name(ctx)
					},
					{{if $side.FromColumns -}}
					FromColumns: []string{
						{{range $name := $side.FromColumns -}}
						{{- $colAlias := index $from.Columns $name -}}
						ColumnNames.{{$from.UpPlural}}.{{$colAlias}},
						{{- end}}
					},
					{{- end}}
					{{if $side.ToColumns -}}
					ToColumns: []string{
						{{range $name := $side.ToColumns -}}
						{{- $colAlias := index $to.Columns $name -}}
						ColumnNames.{{$to.UpPlural}}.{{$colAlias}},
						{{- end}}
					},
					{{end -}}
					{{if $side.FromWhere -}}
					FromWhere: []orm.RelWhere{
						{{range $where := $side.FromWhere -}}
						{{- $colAlias := index $from.Columns $where.Column -}}
						{
						  Column: ColumnNames.{{$from.UpPlural}}.{{$colAlias}},
							SQLValue: {{quote $where.SQLValue}},
							GoValue: {{quote $where.GoValue}},
						},
						{{end -}}
					},
					{{end -}}
					{{if $side.ToWhere -}}
					ToWhere: []orm.RelWhere{
						{{range $where := $side.ToWhere -}}
						{{- $colAlias := index $to.Columns $where.Column -}}
						{
							Column: ColumnNames.{{$to.UpPlural}}.{{$colAlias}},
							SQLValue: {{quote $where.SQLValue}},
							GoValue: {{quote $where.GoValue}},
						},
						{{end -}}
					},
					{{end -}}
				},
				{{- end}}
			},
		}, {{$fAlias.UpPlural}}.Columns().Names(), opts...)
}
{{- end}}

func ThenLoad{{$tAlias.UpSingular}}{{$relAlias}}(queryMods ...bob.Mod[*dialect.SelectQuery]) {{$.Dialect}}.Loader {
	return {{$.Dialect}}.Loader(func(ctx context.Context, exec bob.Executor, retrieved any) error {
		loader, isLoader := retrieved.(interface{
			Load{{$tAlias.UpSingular}}{{$relAlias}}(context.Context, bob.Executor, ...bob.Mod[*dialect.SelectQuery]) error
		})
		if !isLoader {
			return fmt.Errorf("object %T cannot load {{$tAlias.UpSingular}}{{$relAlias}}", retrieved)
		}

		err := loader.Load{{$tAlias.UpSingular}}{{$relAlias}}(ctx, exec, queryMods...)

		// Don't cause an issue due to missing relationships
		if errors.Is(err, sql.ErrNoRows) {
		  return nil
		}

		return err
	})
}

// Load{{$tAlias.UpSingular}}{{$relAlias}} loads the {{$tAlias.DownSingular}}'s {{$relAlias}} into the .R struct
func (o *{{$tAlias.UpSingular}}) Load{{$tAlias.UpSingular}}{{$relAlias}}(ctx context.Context, exec bob.Executor, mods ...bob.Mod[*dialect.SelectQuery]) error {
  if o == nil {
	  return nil
	}

	// Reset the relationship
	o.R.{{$relAlias}} = nil

	{{if $rel.IsToMany -}}
	related, err := o.{{relQueryMethodName $tAlias $relAlias}}(ctx, exec, mods...).All()
	{{else -}}
	related, err := o.{{relQueryMethodName $tAlias $relAlias}}(ctx, exec, mods...).One()
	{{end -}}
	if err != nil {
		return err
	}

	{{if and (not $.NoBackReferencing) $invRel.Name -}}
	{{- $invAlias := $fAlias.Relationship $invRel.Name -}}
	{{if $rel.IsToMany -}}
		for _, rel := range related {
			{{if $invRel.IsToMany -}}
				rel.R.{{$invAlias}} = {{$tAlias.UpSingular}}Slice{o}
			{{- else -}}
				rel.R.{{$invAlias}} =  o
			{{- end}}
		}
	{{else -}}
		{{if $invRel.IsToMany -}}
			related.R.{{$invAlias}} = {{$tAlias.UpSingular}}Slice{o}
		{{else -}}
			related.R.{{$invAlias}} =  o
		{{- end}}
	{{- end}}
	{{- end}}

	o.R.{{$relAlias}} = related
	return nil
}

// Load{{$tAlias.UpSingular}}{{$relAlias}} loads the {{$tAlias.DownSingular}}'s {{$relAlias}} into the .R struct
{{if le (len $rel.Sides) 1 -}}
func (os {{$tAlias.UpSingular}}Slice) Load{{$tAlias.UpSingular}}{{$relAlias}}(ctx context.Context, exec bob.Executor, mods ...bob.Mod[*dialect.SelectQuery]) error {
	{{- $side := (index $rel.Sides 0) -}}
	{{- $fromAlias := $.Aliases.Table $side.From -}}
	{{- $toAlias := $.Aliases.Table $side.To -}}
  if len(os) == 0 {
	  return nil
	}

	{{$fAlias.DownPlural}}, err := os.{{relQueryMethodName $tAlias $relAlias}}(ctx, exec, mods...).All()
	if err != nil {
		return err
	}

	{{if $rel.IsToMany -}}
		for _, o := range os {
			o.R.{{$relAlias}} = nil
		}
	{{- end}}

	for _, o := range os {
		for _, rel := range {{$fAlias.DownPlural}} {
			{{range $index, $local := $side.FromColumns -}}
			{{- $foreign := index $side.ToColumns $index -}}
			{{- $fromColGet := columnGetter $.Tables $side.From $fromAlias $local -}}
			{{- $toColGet := columnGetter $.Tables $side.To $toAlias $foreign -}}
			if o.{{$fromColGet}} != rel.{{$toColGet}} {
			  continue
			}
			{{end}}

			{{if and (not $.NoBackReferencing) $invRel.Name -}}
			{{- $invAlias := $fAlias.Relationship $invRel.Name -}}
				{{if $invRel.IsToMany -}}
					rel.R.{{$invAlias}} = append(rel.R.{{$invAlias}}, o)
				{{else -}}
					rel.R.{{$invAlias}} =  o
				{{- end}}
			{{- end}}

			{{if $rel.IsToMany -}}
				o.R.{{$relAlias}} = append(o.R.{{$relAlias}}, rel)
			{{else -}}
				o.R.{{$relAlias}} =  rel
				break
			{{end -}}
		}
	}

	return nil
}

{{else -}}
func (os {{$tAlias.UpSingular}}Slice) Load{{$tAlias.UpSingular}}{{$relAlias}}(ctx context.Context, exec bob.Executor, mods ...bob.Mod[*dialect.SelectQuery]) error {
	{{- $firstSide := (index $rel.Sides 0) -}}
	{{- $firstFrom := $.Aliases.Table $firstSide.From -}}
	{{- $firstTo := $.Aliases.Table $firstSide.To -}}
  if len(os) == 0 {
	  return nil
	}
  
  // since we are changing the columns, we need to check if the original columns were set or add the defaults
  sq := dialect.SelectQuery{}
  for _, mod := range mods {
   mod.Apply(&sq)
  }

	if len(sq.SelectList.Columns) == 0 {
		mods = append(mods, sm.Columns({{$fAlias.UpPlural}}.Columns()))
	}

	q := os.{{relQueryMethodName $tAlias $relAlias}}(ctx, exec, append(
		mods, 
		{{range $index, $local := $firstSide.FromColumns -}}
			{{- $toCol := index $firstTo.Columns (index $firstSide.ToColumns $index) -}}
			{{- $fromCol := index $firstFrom.Columns $local -}}
			sm.Columns({{$firstTo.UpSingular}}Columns.{{$toCol}}.As("related_{{$firstSide.From}}.{{$fromCol}}")),
		{{- end}}
	)...)

  {{range $index, $local := $firstSide.FromColumns -}}
    {{- $fromColAlias := index $firstFrom.Columns $local -}}
    {{- $fromCol := getColumn $.Tables $firstSide.From $firstFrom $local -}}
    {{$fromColAlias}}Slice := []{{$fromCol.Type}}{}
  {{end}}

	{{$.Importer.Import "github.com/stephenafamo/scan" -}}
  mapper := scan.Mod(scan.StructMapper[*{{$fAlias.UpSingular}}](), func(ctx context.Context, cols []string) (scan.BeforeFunc, func(any, any) error) {
    return func(row *scan.Row) (any, error) {
      {{range $index, $local := $firstSide.FromColumns -}}
        {{- $fromColAlias := index $firstFrom.Columns $local -}}
        {{- $fromCol := getColumn $.Tables $firstSide.From $firstFrom $local -}}
        {{$fromColAlias}}Slice = append({{$fromColAlias}}Slice, *new({{$fromCol.Type}}))
        row.ScheduleScan("related_{{$firstSide.From}}.{{$fromColAlias}}", &{{$fromColAlias}}Slice[len({{$fromColAlias}}Slice)-1])
      {{end}}

      return nil, nil
    },
    func(any, any) error {
      return nil
    }
  })

	{{$fAlias.DownPlural}}, err := bob.Allx[*{{$fAlias.UpSingular}}, {{$fAlias.UpSingular}}Slice](ctx, exec, q, mapper)
	if err != nil {
		return err
	}

	{{if $rel.IsToMany -}}
		for _, o := range os {
			o.R.{{$relAlias}} = nil
		}
	{{- end}}

	for _, o := range os {
		for i, rel := range {{$fAlias.DownPlural}} {
			{{range $index, $local := $firstSide.FromColumns -}}
			{{$fromCol := index $firstFrom.Columns $local -}}
			if o.{{$fromCol}} != {{$fromCol}}Slice[i] {
			  continue
			}
			{{end}}

			{{if and (not $.NoBackReferencing) $invRel.Name -}}
			{{- $invAlias := $fAlias.Relationship $invRel.Name -}}
				{{if $invRel.IsToMany -}}
					rel.R.{{$invAlias}} = append(rel.R.{{$invAlias}}, o)
				{{else -}}
					rel.R.{{$invAlias}} =  o
				{{- end}}
			{{- end}}


			{{if $rel.IsToMany -}}
				o.R.{{$relAlias}} = append(o.R.{{$relAlias}}, rel)
			{{else -}}
				o.R.{{$relAlias}} =  rel
				break
			{{end -}}
		}
	}

	return nil
}

{{end -}}
{{end -}}
