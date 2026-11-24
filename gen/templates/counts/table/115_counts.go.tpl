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
{{$.Importer.Import "github.com/stephenafamo/bob/orm/genruntime"}}
{{$.Importer.Import (printf "github.com/stephenafamo/bob/dialect/%s" $.Dialect)}}
{{$.Importer.Import (printf "github.com/stephenafamo/bob/dialect/%s/dialect" $.Dialect)}}

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
		{{$relAlias}}: func(mods ...bob.Mod[*dialect.SelectQuery]) {{$.Dialect}}.Preloader {
			return genruntime.PreloadCount("{{$relAlias}}", func(parent string) bob.BaseQuery[*dialect.SelectQuery] {
				if parent == "" { parent = {{$tAlias.UpPlural}}.Alias() }
				return correlatedQuery({{$tAlias.DownSingular}}Rel{{$relAlias}}, parent, {{$.Dialect}}.Raw("count(*)"), mods)
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

// LoadCount{{$relAlias}} loads the count of {{$relAlias}} for a slice in a single batch query
func (os {{$tAlias.UpSingular}}Slice) LoadCount{{$relAlias}}(ctx context.Context, exec bob.Executor, mods ...bob.Mod[*dialect.SelectQuery]) error {
	if len(os) == 0 {
		return nil
	}

	{{$firstSide := index $rel.Sides 0 -}}
	parents := os{{if $firstSide.FromWhere}}.matching{{$relAlias}}Sources(){{end}}
	{{if $firstSide.FromColumns -}}
	parents, keys, owners := parents.parentsFor{{$relAlias}}()
	q := {{$tAlias.DownSingular}}QueryFor{{$relAlias}}(keys, mods...)
	counts, err := genruntime.ParentCounts(ctx, exec, q.ExecQuery, len(parents), {{$.Dialect}}.Raw("max(?)", owners).As("parents"),
		{{range $col := $firstSide.ToColumns}}{{$.Dialect}}.Quote({{$tAlias.DownSingular}}PathFor{{$relAlias}}[0].Table.Alias, {{quote $col}}),{{end}}
	)
	if err != nil { return err }
	{{if $firstSide.FromWhere}}for _, o := range os { if o != nil { o.C.{{$relAlias}} = new(int64) } }{{end}}
	for i, o := range parents { if o != nil { n := counts[i]; o.C.{{$relAlias}} = &n } }
	{{else -}}
	// No join keys: one shared count for all eligible parents.
	count, err := os.{{relQueryMethodName $tAlias $relAlias}}(mods...).Count(ctx, exec)
	if err != nil { return err }
	{{if $firstSide.FromWhere -}}
	for _, o := range os { if o != nil { o.C.{{$relAlias}} = new(int64) } }
	{{end -}}
	for _, o := range parents { if o != nil { n := count; o.C.{{$relAlias}} = &n } }

	{{end -}}
	return nil
}

{{end -}}
{{end -}}
