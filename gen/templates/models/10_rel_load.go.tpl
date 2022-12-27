{{$table := .Table}}
{{$tAlias := .Aliases.Table $table.Name -}}

{{if $table.Relationships -}}
{{$.Importer.Import "fmt" -}}
{{$.Importer.Import "context" -}}
{{$.Importer.Import "database/sql" -}}
{{$.Importer.Import "errors" -}}
{{$.Importer.Import (printf "github.com/stephenafamo/bob/dialect/%s/select/qm" $.Dialect) -}}
func (o *{{$tAlias.UpSingular}}) EagerLoad(name string, retrieved any) error {
	if o == nil {
		return nil
	}

	switch name {
	{{range $table.Relationships -}}
	{{- $fAlias := $.Aliases.Table .Foreign -}}
	{{- $relAlias := $tAlias.Relationship .Name -}}
	{{- $invRel := $table.GetRelationshipInverse $.Tables . -}}
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
				{{if $invRel.IsToMany -}}
					rel.R.{{$invAlias}} = {{$tAlias.UpSingular}}Slice{o}
				{{else -}}
					rel.R.{{$invAlias}} =  o
				{{- end}}
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
				{{if $invRel.IsToMany -}}
					rel.R.{{$invAlias}} = {{$tAlias.UpSingular}}Slice{o}
				{{else -}}
					rel.R.{{$invAlias}} =  o
				{{- end}}
			{{- end}}
			return nil
		{{end -}}

	{{end -}}
	default:
		return fmt.Errorf("{{$tAlias.DownSingular}} has no relationship %q", name)
	}
}
{{- end}}


{{range $rel := $table.Relationships -}}
{{- $fAlias := $.Aliases.Table $rel.Foreign -}}
{{- $relAlias := $tAlias.Relationship $rel.Name -}}
{{- $invRel := $table.GetRelationshipInverse $.Tables . -}}
{{- if not $rel.IsToMany -}}
{{$.Importer.Import "github.com/stephenafamo/bob/orm"}}
func Preload{{$tAlias.UpSingular}}{{$relAlias}}(opts ...{{$.Dialect}}.EagerLoadOption) {{$.Dialect}}.EagerLoader {
	return {{$.Dialect}}.Preload[*{{$fAlias.UpSingular}}, {{$fAlias.UpSingular}}Slice](orm.Relationship{
			Name: "{{$relAlias}}",
			Sides:  []orm.RelSide{
				{{range $side := $rel.Sides -}}
				{{- $from := $.Aliases.Table $side.From -}}
				{{- $to := $.Aliases.Table $side.To -}}
				{
					From:   TableNames.{{$from.UpPlural}},
					To: TableNames.{{$to.UpPlural}},
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
							Value: {{$where.Value}},
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
							Value: {{$where.Value}},
						},
						{{end -}}
					},
					{{end -}}
				},
				{{- end}}
			},
		}, {{$fAlias.UpPlural}}Table.Columns(), opts...)
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

		return loader.Load{{$tAlias.UpSingular}}{{$relAlias}}(ctx, exec, queryMods...)
	})
}

// Load{{$tAlias.UpSingular}}{{$relAlias}} loads the {{$tAlias.DownSingular}}'s {{$relAlias}} into the .R struct
func (o *{{$tAlias.UpSingular}}) Load{{$tAlias.UpSingular}}{{$relAlias}}(ctx context.Context, exec bob.Executor, mods ...bob.Mod[*dialect.SelectQuery]) error {
	{{if $rel.IsToMany -}}
	related, err := o.{{$relAlias}}(mods...).All(ctx, exec)
	{{else -}}
	related, err := o.{{$relAlias}}(mods...).One(ctx, exec)
	{{end -}}
	if err != nil && !errors.Is(err, sql.ErrNoRows){
		return err
	}

	{{if and (not $.NoBackReferencing) $invRel.Name -}}
	{{- $invAlias := $fAlias.Relationship $invRel.Name -}}
	{{if $rel.IsToMany -}}
		for _, rel := range related {
			{{if $invRel.IsToMany -}}
				rel.R.{{$invAlias}} = {{$tAlias.UpSingular}}Slice{o}
			{{else -}}
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

	{{$fAlias.DownPlural}}, err := os.{{$relAlias}}(mods...).All(ctx, exec)
	if err != nil && !errors.Is(err, sql.ErrNoRows){
		return err
	}

	{{if $rel.IsToMany -}}
		for _, o := range os {
			o.R.{{$relAlias}} = nil
		}
	{{- end}}

	for _, rel := range {{$fAlias.DownPlural}} {
		for _, o := range os {
			{{range $index, $local := $side.FromColumns -}}
			{{- $foreign := index $side.ToColumns $index -}}
			{{- $fromColGet := columnGetter $.Tables $side.From $fromAlias $local -}}
			{{- $toColGet := columnGetter $.Tables $side.To $toAlias $foreign -}}
			if o.{{$fromColGet}} != rel.{{$toColGet}} {
			  continue
			}
			{{- end}}

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

	q := os.{{$relAlias}}(mods...)
	q.Apply(
		{{range $index, $local := $firstSide.FromColumns -}}
			{{- $toCol := index $firstTo.Columns (index $firstSide.ToColumns $index) -}}
			{{- $fromCol := index $firstFrom.Columns $local -}}
			qm.Columns({{$firstTo.UpSingular}}Columns.{{$toCol}}.As("related_{{$firstSide.From}}.{{$fromCol}}")),
		{{- end}}
	)

	{{$.Importer.Import "github.com/stephenafamo/scan" -}}
	{{$fAlias.DownPlural}}, err := bob.All(ctx, exec, q, scan.StructMapper[*struct{
	  {{$fAlias.UpSingular}}
		{{range $index, $local := $firstSide.FromColumns -}}
			{{- $fromColAlias := index $firstFrom.Columns $local -}}
			{{- $fromCol := getColumn $.Tables $firstSide.From $firstFrom $local -}}
			Related{{$fromColAlias}} {{$fromCol.Type}} `db:"related_{{$firstSide.From}}.{{$fromColAlias}}"`
		{{- end}}
	}]())
	if err != nil && !errors.Is(err, sql.ErrNoRows){
		return err
	}

	{{if $rel.IsToMany -}}
		for _, o := range os {
			o.R.{{$relAlias}} = nil
		}
	{{- end}}

	for _, rel := range {{$fAlias.DownPlural}} {
		for _, o := range os {
			{{range $index, $local := $firstSide.FromColumns -}}
			{{- $fromCol := index $firstFrom.Columns $local -}}
			if o.{{$fromCol}} != rel.Related{{$fromCol}} {
			  continue
			}
			{{- end}}

			{{if and (not $.NoBackReferencing) $invRel.Name -}}
			{{- $invAlias := $fAlias.Relationship $invRel.Name -}}
				{{if $invRel.IsToMany -}}
					rel.R.{{$invAlias}} = append(rel.R.{{$invAlias}}, o)
				{{else -}}
					rel.R.{{$invAlias}} =  o
				{{- end}}
			{{- end}}


			{{if $rel.IsToMany -}}
				o.R.{{$relAlias}} = append(o.R.{{$relAlias}}, &rel.{{$fAlias.UpSingular}})
			{{else -}}
				o.R.{{$relAlias}} =  &rel.{{$fAlias.UpSingular}}
				break
			{{end -}}
		}
	}

	return nil
}

{{end -}}
{{end -}}
