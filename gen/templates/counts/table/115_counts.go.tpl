{{- $table := .Table -}}
{{- $tAlias := .Aliases.Table $table.Key -}}
{{- $rels := $.Relationships.Get $table.Key -}}
{{- $hasToMany := false -}}
{{- range $rel := $rels -}}
	{{- if $rel.IsToMany -}}{{- $hasToMany = true -}}{{- end -}}
{{- end -}}

{{if $hasToMany -}}
{{$.Importer.Import "context"}}
{{$.Importer.Import "github.com/stephenafamo/bob"}}
{{$.Importer.Import "github.com/stephenafamo/bob/orm"}}
{{$.Importer.Import (printf "github.com/stephenafamo/bob/dialect/%s" $.Dialect)}}
{{$.Importer.Import (printf "github.com/stephenafamo/bob/dialect/%s/dialect" $.Dialect)}}
{{$.Importer.Import (printf "github.com/stephenafamo/bob/dialect/%s/sm" $.Dialect)}}

// {{$tAlias.DownSingular}}C is where relationship counts are stored.
type {{$tAlias.DownSingular}}C struct {
	{{range $rel := $rels -}}
	{{- if not $rel.IsToMany}}{{continue}}{{end -}}
	{{- $relAlias := $tAlias.Relationship $rel.Name -}}
	{{$relAlias}} *int64 {{if $.Tags}}`{{generateTags $.Tags $relAlias | trim}}`{{end}}
	{{end -}}
}

// PreloadCount sets a count in the C struct by name
func (o *{{$tAlias.UpSingular}}) PreloadCount(name string, count int64) error {
	if o == nil {
		return nil
	}

	switch name {
	{{range $rel := $rels -}}
	{{- if not $rel.IsToMany}}{{continue}}{{end -}}
	{{- $relAlias := $tAlias.Relationship $rel.Name -}}
	case "{{$relAlias}}":
		o.C.{{$relAlias}} = &count
	{{end -}}
	}
	return nil
}

type {{$tAlias.DownSingular}}CountPreloader struct {
	{{range $rel := $rels -}}
	{{- if not $rel.IsToMany}}{{continue}}{{end -}}
	{{- $relAlias := $tAlias.Relationship $rel.Name -}}
	{{$relAlias}} func(...bob.Mod[*dialect.SelectQuery]) {{$.Dialect}}.Preloader
	{{end -}}
}

func build{{$tAlias.UpSingular}}CountPreloader() {{$tAlias.DownSingular}}CountPreloader {
	return {{$tAlias.DownSingular}}CountPreloader{
		{{range $rel := $rels -}}
		{{- if not $rel.IsToMany}}{{continue}}{{end -}}
		{{- $relAlias := $tAlias.Relationship $rel.Name -}}
		{{- $fAlias := $.Aliases.Table $rel.Foreign -}}
		{{$relAlias}}: func(mods ...bob.Mod[*dialect.SelectQuery]) {{$.Dialect}}.Preloader {
			return countPreloader[*{{$tAlias.UpSingular}}]("{{$relAlias}}", func(parent string) bob.Expression {
				// Build a correlated subquery: (SELECT COUNT(*) FROM related WHERE fk = parent.pk)
				if parent == "" {
					parent = {{$tAlias.UpPlural}}.Alias()
				}
				{{$firstSide := index $rel.Sides 0 -}}
				{{$fromAlias := $.Aliases.Table $firstSide.From -}}
				{{$lastSide := index $rel.Sides (sub (len $rel.Sides) 1) -}}
				{{$toAlias := $.Aliases.Table $lastSide.To}}
				subqueryMods := []bob.Mod[*dialect.SelectQuery]{
					sm.Columns({{$.Dialect}}.Raw("count(*)")),
					{{- if eq (len $rel.Sides) 1}}
					{{/* Simple one-hop relationship */}}
					sm.From({{$fAlias.UpPlural}}.Name()),
					{{- range $index, $fromCol := $firstSide.FromColumns -}}
					{{- $toCol := index $firstSide.ToColumns $index}}
					sm.Where({{$.Dialect}}.Quote({{$fAlias.UpPlural}}.Alias(), {{quote $toCol}}).EQ({{$.Dialect}}.Quote(parent, {{quote $fromCol}}))),
					{{- end}}
					{{- else}}
					{{/* Multi-hop relationship - need to join through intermediate tables */}}
					{{- $firstSideToAlias := $.Aliases.Table $firstSide.To}}
					sm.From({{$firstSideToAlias.UpPlural}}.Name()),
					{{- range $index, $fromCol := $firstSide.FromColumns -}}
					{{- $toCol := index $firstSide.ToColumns $index}}
					sm.Where({{$.Dialect}}.Quote({{$firstSideToAlias.UpPlural}}.Alias(), {{quote $toCol}}).EQ({{$.Dialect}}.Quote(parent, {{quote $fromCol}}))),
					{{- end}}
					{{- range $sideIndex, $side := $rel.Sides -}}
					{{- if eq $sideIndex 0 -}}{{continue}}{{- end}}
					{{- $sideFromAlias := $.Aliases.Table $side.From -}}
					{{- $sideToAlias := $.Aliases.Table $side.To}}
					sm.InnerJoin({{$sideToAlias.UpPlural}}.Name()).On(
						{{- range $index, $fromCol := $side.FromColumns -}}
						{{- $toCol := index $side.ToColumns $index}}
						{{$.Dialect}}.Quote({{$sideToAlias.UpPlural}}.Alias(), {{quote $toCol}}).EQ({{$.Dialect}}.Quote({{$sideFromAlias.UpPlural}}.Alias(), {{quote $fromCol}})),
						{{- end}}
					),
					{{- end}}
					{{- end}}
				}
				subqueryMods = append(subqueryMods, mods...)
				return {{$.Dialect}}.Group({{$.Dialect}}.Select(subqueryMods...).Expression)
			})
		},
		{{end -}}
	}
}

type {{$tAlias.DownSingular}}CountThenLoader[Q orm.Loadable] struct {
	{{range $rel := $rels -}}
	{{- if not $rel.IsToMany}}{{continue}}{{end -}}
	{{- $relAlias := $tAlias.Relationship $rel.Name -}}
	{{$relAlias}} func(...bob.Mod[*dialect.SelectQuery]) orm.Loader[Q]
	{{end -}}
}

func build{{$tAlias.UpSingular}}CountThenLoader[Q orm.Loadable]() {{$tAlias.DownSingular}}CountThenLoader[Q] {
	{{range $rel := $rels -}}
	{{- if not $rel.IsToMany}}{{continue}}{{end -}}
	{{$relAlias := $tAlias.Relationship $rel.Name -}}
	type {{$relAlias}}CountInterface interface {
		LoadCount{{$relAlias}}(context.Context, bob.Executor, ...bob.Mod[*dialect.SelectQuery]) error
	}
	{{end}}

	return {{$tAlias.DownSingular}}CountThenLoader[Q]{
		{{range $rel := $rels -}}
		{{- if not $rel.IsToMany}}{{continue}}{{end -}}
		{{$relAlias := $tAlias.Relationship $rel.Name -}}
		{{$relAlias}}: countThenLoadBuilder[Q](
			"{{$relAlias}}",
			func(ctx context.Context, exec bob.Executor, retrieved {{$relAlias}}CountInterface, mods ...bob.Mod[*dialect.SelectQuery]) error {
				return retrieved.LoadCount{{$relAlias}}(ctx, exec, mods...)
			},
		),
		{{end}}
	}
}

{{range $rel := $rels -}}
{{- if not $rel.IsToMany}}{{continue}}{{end -}}
{{- $fAlias := $.Aliases.Table $rel.Foreign -}}
{{- $relAlias := $tAlias.Relationship $rel.Name -}}

// LoadCount{{$relAlias}} loads the count of {{$relAlias}} into the C struct
func (o *{{$tAlias.UpSingular}}) LoadCount{{$relAlias}}(ctx context.Context, exec bob.Executor, mods ...bob.Mod[*dialect.SelectQuery]) error {
	if o == nil {
		return nil
	}

	count, err := o.{{relQueryMethodName $tAlias $relAlias}}(mods...).Count(ctx, exec)
	if err != nil {
		return err
	}

	o.C.{{$relAlias}} = &count
	return nil
}

// LoadCount{{$relAlias}} loads the count of {{$relAlias}} for a slice
func (os {{$tAlias.UpSingular}}Slice) LoadCount{{$relAlias}}(ctx context.Context, exec bob.Executor, mods ...bob.Mod[*dialect.SelectQuery]) error {
	if len(os) == 0 {
		return nil
	}

	for _, o := range os {
		if err := o.LoadCount{{$relAlias}}(ctx, exec, mods...); err != nil {
			return err
		}
	}

	return nil
}

{{end -}}
{{end -}}
