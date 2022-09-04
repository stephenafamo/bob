{{$table := .Table}}
{{$tAlias := .Aliases.Table $table.Name -}}

{{if $table.Relationships -}}
{{$.Importer.Import "fmt" -}}
{{$.Importer.Import "database/sql" -}}
{{$.Importer.Import "errors" -}}
func (o *{{$tAlias.UpSingular}}) EagerLoad(name string, retrieved any) error {
	if o == nil {
		return nil
	}

	switch name {
	{{range $table.Relationships -}}
	{{- $ftable := $.Aliases.Table .Foreign -}}
	{{- $relAlias := $tAlias.Relationship .Name -}}
	case "{{$relAlias}}":
		{{if relIsToMany . -}}
		if rel, ok := retrieved.({{$ftable.UpSingular}}Slice); ok {
			o.R.{{$relAlias}} = append(o.R.{{$relAlias}}, rel...)
			return nil
		}
		{{else -}}
		if rel, ok := retrieved.(*{{$ftable.UpSingular}}); ok {
			o.R.{{$relAlias}} = rel
			return nil
		}
		{{end -}}
		return fmt.Errorf("{{$tAlias.DownSingular}} cannot load %T as %q", retrieved, name)

	{{end -}}
	default:
		return fmt.Errorf("{{$tAlias.DownSingular}} has no relationship %q", name)
	}
}
{{- end}}


{{range $rel := $table.Relationships -}}
{{- $ftable := $.Aliases.Table $rel.Foreign -}}
{{- $relAlias := $tAlias.Relationship $rel.Name -}}
{{- if not (relIsToMany $rel) -}}
{{$.Importer.Import "github.com/stephenafamo/bob/orm"}}
func Preload{{$tAlias.UpSingular}}{{$relAlias}}(opts ...model.EagerLoadOption) model.EagerLoader {
	return model.Preload[*{{$ftable.UpSingular}}, {{$ftable.UpSingular}}Slice](orm.Relationship{
			Name: "{{$relAlias}}",
			Sides:  []orm.RelSide{
				{{range $side := $rel.Sides -}}
				{{- $from := $.Aliases.Table $side.From -}}
				{{- $to := $.Aliases.Table $side.To -}}
				{
					From:   TableNames.{{$from.UpPlural}},
					To: TableNames.{{$to.UpPlural}},
					Pairs:  map[string]string{
					{{range $l, $f := $side.Pairs -}}
						{{- $fromCol := index $from.Columns $l -}}
						{{- $toCol := index $to.Columns $f -}}
						ColumnNames.{{$from.UpPlural}}.{{$fromCol}}: ColumnNames.{{$to.UpPlural}}.{{$toCol}},
					{{- end}}
					},
				},
				{{- end}}
			},
		}, {{$ftable.UpPlural}}Table.Columns(), opts...)
}
{{- end}}

func ThenLoad{{$tAlias.UpSingular}}{{$relAlias}}(queryMods ...bob.Mod[*{{$.Dialect}}.SelectQuery]) model.Loader {
	return model.Loader(func(ctx context.Context, exec bob.Executor, retrieved any) error {
		loader, isLoader := retrieved.(interface{
			Load{{$tAlias.UpSingular}}{{$relAlias}}(context.Context, bob.Executor, ...bob.Mod[*{{$.Dialect}}.SelectQuery]) error
		})
		if !isLoader {
			return fmt.Errorf("object %T cannot load {{$tAlias.UpSingular}}{{$relAlias}}", retrieved)
		}

		return loader.Load{{$tAlias.UpSingular}}{{$relAlias}}(ctx, exec, queryMods...)
	})
}

func (o *{{$tAlias.UpSingular}}) Load{{$tAlias.UpSingular}}{{$relAlias}}(ctx context.Context, exec bob.Executor, mods ...bob.Mod[*{{$.Dialect}}.SelectQuery]) error {
	q := {{$ftable.UpPlural}}(mods...)
	q.Apply(
		{{- range $index := until (len $rel.Sides) | reverse -}}
		{{/* Index counts down */}}
		{{/* This also flips the meaning of $from and $to */}}
		{{- $side := index $rel.Sides $index -}}
		{{- $from := $.Aliases.Table $side.From -}}
		{{- $to := $.Aliases.Table $side.To -}}
		{{- if gt $index 0 -}}
		qm.InnerJoin({{$from.UpPlural}}Table.Name()).On(
		{{end -}}
			{{range $l, $f := $side.Pairs -}}
				{{- $fromCol := index $from.Columns $l -}}
				{{- $toCol := index $to.Columns $f -}}
				{{- if gt $index 0 -}}
				{{$to.UpSingular}}Columns.{{$toCol}}.EQ({{$from.UpSingular}}Columns.{{$fromCol}}),
				{{- else -}}
				qm.Where({{$to.UpSingular}}Columns.{{$toCol}}.EQ({{$.Dialect}}.Arg(o.{{$fromCol}}))),
				{{- end}}
			{{- end}}
		{{- if gt $index 0}}
		),
		{{- end -}}
		{{- end}}
	)

	{{if relIsToMany $rel}}
	related, err := q.All(ctx, exec)
	{{- else}}
	related, err := q.One(ctx, exec)
	{{- end}}
	if err != nil && !errors.Is(err, sql.ErrNoRows){
		return err
	}


	o.R.{{$relAlias}} = related
	return nil
}

{{if le (len $rel.Sides) 1 -}}
func (os {{$tAlias.UpSingular}}Slice) Load{{$tAlias.UpSingular}}{{$relAlias}}(ctx context.Context, exec bob.Executor, mods ...bob.Mod[*{{$.Dialect}}.SelectQuery]) error {
	{{- $side := (index $rel.Sides 0) -}}
	{{- $fromAlias := $.Aliases.Table $side.From -}}
	{{- $toAlias := $.Aliases.Table $side.To -}}

	{{range $l, $f := $side.Pairs -}}
		{{- $fromCol := index $fromAlias.Columns $l -}}
		{{$fromCol}}Args := make([]any, 0, len(os))
		for _, o := range os {
			{{$fromCol}}Args = append({{$fromCol}}Args, {{$.Dialect}}.Arg(o.{{$fromCol}}))
		}
	{{- end}}

	q := {{$ftable.UpPlural}}(mods...)
	q.Apply(
		{{range $l, $f := $side.Pairs -}}
			{{- $fromCol := index $fromAlias.Columns $l -}}
			{{- $toCol := index $toAlias.Columns $f -}}
			qm.Where({{$ftable.UpSingular}}Columns.{{$toCol}}.In({{$fromCol}}Args...)),
		{{- end}}
	)


	{{$ftable.DownPlural}}, err := q.All(ctx, exec)
	if err != nil && !errors.Is(err, sql.ErrNoRows){
		return err
	}

	for _, rel := range {{$ftable.DownPlural}} {
		for _, o := range os {
			{{range $l, $f := $side.Pairs -}}
			{{- $fromColGet := columnGetter $.Tables $side.From $fromAlias $l -}}
			{{- $toColGet := columnGetter $.Tables $side.To $toAlias $f -}}
			if o.{{$fromColGet}} != rel.{{$toColGet}} {
			  continue
			}
			{{- end}}

			{{if relIsToMany . -}}
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
func (os {{$tAlias.UpSingular}}Slice) Load{{$tAlias.UpSingular}}{{$relAlias}}(ctx context.Context, exec bob.Executor, mods ...bob.Mod[*{{$.Dialect}}.SelectQuery]) error {
	{{- $firstSide := (index $rel.Sides 0) -}}
	{{- $firstFrom := $.Aliases.Table $firstSide.From -}}
	{{- $firstTo := $.Aliases.Table $firstSide.To -}}

	{{- $lastSide := (index $rel.Sides (sub (len $rel.Sides) 1)) -}}
	{{- $lastFrom := $.Aliases.Table $lastSide.From -}}
	{{- $lastTo := $.Aliases.Table $lastSide.To -}}

	{{range $l, $f := $firstSide.Pairs -}}
		{{- $fromCol := index $firstFrom.Columns $l -}}
		{{$fromCol}}Args := make([]any, 0, len(os))
		for _, o := range os {
			{{$fromCol}}Args = append({{$fromCol}}Args, {{$.Dialect}}.Arg(o.{{$fromCol}}))
		}
	{{- end}}

	q := {{$ftable.UpPlural}}(mods...)
	q.Apply(
		{{- range $index := until (len $rel.Sides) | reverse -}}
		{{/* Index counts down */}}
		{{/* This also flips the meaning of $from and $to */}}
		{{- $side := index $rel.Sides $index -}}
		{{- $from := $.Aliases.Table $side.From -}}
		{{- $to := $.Aliases.Table $side.To -}}
		{{- if gt $index 0 -}}
		qm.InnerJoin({{$from.UpPlural}}Table.Name()).On(
		{{end -}}
			{{range $l, $f := $side.Pairs -}}
				{{- $fromCol := index $from.Columns $l -}}
				{{- $toCol := index $to.Columns $f -}}
				{{- if gt $index 0 -}}
				{{$to.UpSingular}}Columns.{{$toCol}}.EQ({{$from.UpSingular}}Columns.{{$fromCol}}),
				{{- else -}}
					qm.Columns({{$to.UpSingular}}Columns.{{$toCol}}.As("related_{{$side.From}}.{{$fromCol}}")),
					qm.Where({{$to.UpSingular}}Columns.{{$toCol}}.In({{$fromCol}}Args...)),
				{{- end}}
			{{- end}}
		{{- if gt $index 0}}
		),
		{{- end -}}
		{{- end}}
	)


	{{$.Importer.Import "github.com/stephenafamo/scan" -}}
	{{$ftable.DownPlural}}, err := bob.All(ctx, exec, q, scan.StructMapper[*struct{
	  {{$ftable.UpSingular}}
		{{range $l, $f := $firstSide.Pairs -}}
			{{- $fromColAlias := index $firstFrom.Columns $l -}}
			{{- $fromCol := getColumn $.Tables $firstSide.From $firstFrom $l -}}
			Related{{$fromColAlias}} {{$fromCol.Type}} `db:"related_{{$firstSide.From}}.{{$fromColAlias}}"`
		{{- end}}
	}]())
	if err != nil && !errors.Is(err, sql.ErrNoRows){
		return err
	}

	for _, rel := range {{$ftable.DownPlural}} {
		for _, o := range os {
			{{range $l, $f := $firstSide.Pairs -}}
			{{- $fromCol := index $firstFrom.Columns $l -}}
			if o.{{$fromCol}} != rel.Related{{$fromCol}} {
			  continue
			}
			{{- end}}

			{{if relIsToMany . -}}
				o.R.{{$relAlias}} = append(o.R.{{$relAlias}}, &rel.{{$ftable.UpSingular}})
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
