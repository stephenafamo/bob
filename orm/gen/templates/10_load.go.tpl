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
	{{- $ftable := $.Aliases.Table .Foreign -}}
	{{- $relAlias := $tAlias.Relationship .Name -}}
	{{- $invRel := $table.GetRelationshipInverse $.Tables . -}}
	case "{{$relAlias}}":
		{{if .IsToMany -}}
			rels, ok := retrieved.({{$ftable.UpSingular}}Slice)
			if !ok {
				return fmt.Errorf("{{$tAlias.DownSingular}} cannot load %T as %q", retrieved, name)
			}

			o.R.{{$relAlias}} = rels

			{{if and (not $.NoBackReferencing) $invRel.Name -}}
			{{- $invAlias := $ftable.Relationship $invRel.Name -}}
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
			rel, ok := retrieved.(*{{$ftable.UpSingular}})
			if !ok {
				return fmt.Errorf("{{$tAlias.DownSingular}} cannot load %T as %q", retrieved, name)
			}

			o.R.{{$relAlias}} = rel

			{{if and (not $.NoBackReferencing) $invRel.Name -}}
			{{- $invAlias := $ftable.Relationship $invRel.Name -}}
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
{{- $ftable := $.Aliases.Table $rel.Foreign -}}
{{- $relAlias := $tAlias.Relationship $rel.Name -}}
{{- if not $rel.IsToMany -}}
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
			{{range $i, $local := $side.FromColumns -}}
				{{- $fromCol := index $from.Columns $local -}}
				{{- $toCol := index $to.Columns (index $side.ToColumns $i) -}}
				{{- if gt $index 0 -}}
				{{$to.UpSingular}}Columns.{{$toCol}}.EQ({{$from.UpSingular}}Columns.{{$fromCol}}),
				{{- else -}}
				qm.Where({{$to.UpSingular}}Columns.{{$toCol}}.EQ({{$.Dialect}}.Arg(o.{{$fromCol}}))),
				{{- end -}}
			{{- end}}
			{{- range $where := $side.FromWhere}}
				{{- $fromCol := index $from.Columns $where.Column}}
				{{if eq $index 0 -}}qm.Where({{end -}}
				{{$.Dialect}}.X({{$from.UpSingular}}Columns.{{$fromCol}}, "=", {{$.Dialect}}.Arg({{$where.Value}})),
				{{- if eq $index 0 -}}),{{- end -}}
			{{- end}}
			{{- range $where := $side.ToWhere}}
				{{- $toCol := index $to.Columns $where.Column}}
				{{if eq $index 0 -}}qm.Where({{end -}}
				{{$.Dialect}}.X({{$to.UpSingular}}Columns.{{$toCol}}, "=", {{$.Dialect}}.Arg({{$where.Value}}),
				{{- if eq $index 0 -}}),{{- end -}}
			{{- end}}
		{{- if gt $index 0 -}}
		),
		{{- end -}}
		{{- end}}
	)

	{{if $rel.IsToMany}}
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

	{{range $index, $local := $side.FromColumns -}}
		{{- $fromCol := index $fromAlias.Columns $local -}}
		{{$fromCol}}Args := make([]any, 0, len(os))
		for _, o := range os {
			{{$fromCol}}Args = append({{$fromCol}}Args, {{$.Dialect}}.Arg(o.{{$fromCol}}))
		}
	{{- end}}

	q := {{$ftable.UpPlural}}(mods...)
	q.Apply(
		{{range $index, $local := $side.FromColumns -}}
			{{- $fromCol := index $fromAlias.Columns $local -}}
			{{- $toCol := index $toAlias.Columns (index $side.ToColumns $index) -}}
			qm.Where({{$ftable.UpSingular}}Columns.{{$toCol}}.In({{$fromCol}}Args...)),
		{{- end}}
		{{- range $where := $side.FromWhere}}
			{{- $fromCol := index $fromAlias.Columns $where.Column}}
			qm.Where({{$.Dialect}}.X({{$fromAlias.UpSingular}}Columns.{{$fromCol}}, "=", {{$.Dialect}}.Arg({{$where.Value}}))),
		{{- end}}
		{{- range $where := $side.ToWhere}}
			{{- $toCol := index $toAlias.Columns $where.Column}}
			qm.Where({{$.Dialect}}.X({{$toAlias.UpSingular}}Columns.{{$toCol}}, "=", {{$.Dialect}}.Arg({{$where.Value}}))),
		{{- end}}
	)


	{{$ftable.DownPlural}}, err := q.All(ctx, exec)
	if err != nil && !errors.Is(err, sql.ErrNoRows){
		return err
	}

	for _, rel := range {{$ftable.DownPlural}} {
		for _, o := range os {
			{{range $index, $local := $side.FromColumns -}}
			{{- $foreign := index $side.ToColumns $index -}}
			{{- $fromColGet := columnGetter $.Tables $side.From $fromAlias $local -}}
			{{- $toColGet := columnGetter $.Tables $side.To $toAlias $foreign -}}
			if o.{{$fromColGet}} != rel.{{$toColGet}} {
			  continue
			}
			{{- end}}

			{{if .IsToMany -}}
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

	{{range $index, $local := $firstSide.FromColumns -}}
		{{- $fromCol := index $firstFrom.Columns $local -}}
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
			{{range $i, $local := $side.FromColumns -}}
				{{- $foreign := index $side.ToColumns $i -}}
				{{- $fromCol := index $from.Columns $local -}}
				{{- $toCol := index $to.Columns $foreign -}}
				{{- if gt $index 0 -}}
				{{$to.UpSingular}}Columns.{{$toCol}}.EQ({{$from.UpSingular}}Columns.{{$fromCol}}),
				{{- else -}}
					qm.Columns({{$to.UpSingular}}Columns.{{$toCol}}.As("related_{{$side.From}}.{{$fromCol}}")),
					qm.Where({{$to.UpSingular}}Columns.{{$toCol}}.In({{$fromCol}}Args...)),
				{{- end}}
			{{- end}}
			{{- range $where := $side.FromWhere}}
				{{- $fromCol := index $from.Columns $where.Column}}
				qm.Where({{$.Dialect}}.X({{$from.UpSingular}}Columns.{{$fromCol}}, "=", {{$.Dialect}}.Arg({{$where.Value}}))),
			{{- end}}
			{{- range $where := $side.ToWhere}}
				{{- $toCol := index $to.Columns $where.Column}}
				qm.Where({{$.Dialect}}.X({{$to.UpSingular}}Columns.{{$toCol}}, "=", {{$.Dialect}}.Arg({{$where.Value}}))),
			{{- end}}
		{{- if gt $index 0}}
		),
		{{- end -}}
		{{- end}}
	)


	{{$.Importer.Import "github.com/stephenafamo/scan" -}}
	{{$ftable.DownPlural}}, err := bob.All(ctx, exec, q, scan.StructMapper[*struct{
	  {{$ftable.UpSingular}}
		{{range $index, $local := $firstSide.FromColumns -}}
			{{- $fromColAlias := index $firstFrom.Columns $local -}}
			{{- $fromCol := getColumn $.Tables $firstSide.From $firstFrom $local -}}
			Related{{$fromColAlias}} {{$fromCol.Type}} `db:"related_{{$firstSide.From}}.{{$fromColAlias}}"`
		{{- end}}
	}]())
	if err != nil && !errors.Is(err, sql.ErrNoRows){
		return err
	}

	for _, rel := range {{$ftable.DownPlural}} {
		for _, o := range os {
			{{range $index, $local := $firstSide.FromColumns -}}
			{{- $fromCol := index $firstFrom.Columns $local -}}
			if o.{{$fromCol}} != rel.Related{{$fromCol}} {
			  continue
			}
			{{- end}}

			{{if .IsToMany -}}
				o.R.{{$relAlias}} = append(o.R.{{$relAlias}}, &rel.{{$ftable.UpSingular}})
			{{else -}}
				o.R.{{$relAlias}} =  &rel.{{$ftable.UpSingular}}
				break
			{{end -}}
		}
	}

	return nil
}

{{end -}}
{{end -}}
