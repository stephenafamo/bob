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
	{{$relAlias}} func(...bob.Mod[dialect.SelectEditor]) {{$.Dialect}}.Preloader
	{{end -}}
}

func build{{$tAlias.UpSingular}}CountPreloader() {{$tAlias.DownSingular}}CountPreloader {
	return {{$tAlias.DownSingular}}CountPreloader{
		{{range $rel := $rels -}}
		{{- if not $rel.IsToMany}}{{continue}}{{end -}}
		{{- $relAlias := $tAlias.Relationship $rel.Name -}}
		{{$relAlias}}: func(mods ...bob.Mod[dialect.SelectEditor]) {{$.Dialect}}.Preloader {
			return genruntime.PreloadCount("{{$relAlias}}", func(parent string) bob.QueryBuilder[dialect.SelectEditor] {
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
	{{$relAlias}} func(...bob.Mod[dialect.SelectEditor]) orm.Loader[Q]
	{{end -}}
}

func build{{$tAlias.UpSingular}}CountThenLoader[Q orm.Loadable]() {{$tAlias.DownSingular}}CountThenLoader[Q] {
	{{range $rel := $rels -}}
	{{- if not $rel.IsToMany}}{{continue}}{{end -}}
	{{$relAlias := $tAlias.Relationship $rel.Name -}}
	type {{$relAlias}}CountInterface interface {
		LoadCount{{$relAlias}}(context.Context, bob.Executor, ...bob.Mod[dialect.SelectEditor]) error
	}
	{{end}}

	return {{$tAlias.DownSingular}}CountThenLoader[Q]{
		{{range $rel := $rels -}}
		{{- if not $rel.IsToMany}}{{continue}}{{end -}}
		{{$relAlias := $tAlias.Relationship $rel.Name -}}
		{{$relAlias}}: countThenLoadBuilder[Q](
			"{{$relAlias}}",
			func(ctx context.Context, exec bob.Executor, retrieved {{$relAlias}}CountInterface, mods ...bob.Mod[dialect.SelectEditor]) error {
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
func (o *{{$tAlias.UpSingular}}) LoadCount{{$relAlias}}(ctx context.Context, exec bob.Executor, mods ...bob.Mod[dialect.SelectEditor]) error {
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
func (os {{$tAlias.UpSingular}}Slice) LoadCount{{$relAlias}}(ctx context.Context, exec bob.Executor, mods ...bob.Mod[dialect.SelectEditor]) error {
	if len(os) == 0 {
		return nil
	}

	{{$firstSide := index $rel.Sides 0 -}}
	parents := os{{if $firstSide.FromWhere}}.matching{{$relAlias}}Sources(){{end}}
	{{if $firstSide.FromColumns -}}
	{{$from := $.Aliases.Table $firstSide.From -}}
	{{$path := printf "%sPathFor%s" $tAlias.DownSingular $relAlias -}}
	{{$comparable := true -}}
	type countResult struct {
		{{range $i, $name := $firstSide.FromColumns -}}
		{{$col := $table.GetColumn $name -}}
		{{if not ($.Types.CanCompareWithEquals $.CurrentPackage $col.Type)}}{{$comparable = false}}{{end -}}
		K{{$i}} {{$.Types.Get $.CurrentPackage $.Importer $col.Type}}
		{{end -}}
		Count int64
	}
	q := parents.{{relQueryMethodName $tAlias $relAlias}}(mods...)
	results, err := genruntime.CountByKey[countResult](ctx, exec, q.QueryBuilder,
		[]any{ {{range $i, $name := $firstSide.ToColumns}}{{$.Dialect}}.Quote({{$path}}[0].Table.Alias, {{quote $name}}).As("k{{$i}}"),{{end}} },
		[]any{ {{range $firstSide.ToColumns}}{{$.Dialect}}.Quote({{$path}}[0].Table.Alias, {{quote .}}),{{end}} })
	if err != nil { return err }
	{{if $firstSide.FromWhere}}for _, o := range os { if o != nil { o.C.{{$relAlias}} = new(int64) } }{{end}}
	{{if $comparable -}}
	type countKey struct {
		{{range $i, $name := $firstSide.FromColumns -}}
		K{{$i}} {{$.Types.Get $.CurrentPackage $.Importer ($table.GetColumn $name).Type}}
		{{end -}}
	}
	genruntime.AssignCounts(parents, results,
		func(o *{{$tAlias.UpSingular}}) (countKey, bool) {
			var key countKey
			{{range $i, $name := $firstSide.FromColumns -}}
			{{$col := $table.GetColumn $name -}}
			{{$field := printf "o.%s" ($from.Column $name) -}}
			{{if $col.Nullable}}if !({{$.Types.GetNullTypeValid $.CurrentPackage $col.Type $field}}) { return key, false }{{end}}
			key.K{{$i}} = {{$.Types.UnwrapNullExpr $.CurrentPackage $.Importer $col.Type $field $col.Nullable}}
			{{end -}}
			return key, true
		},
		func(r countResult) (countKey, int64) { return countKey{ {{range $i, $_ := $firstSide.FromColumns}}r.K{{$i}},{{end}} }, r.Count },
		func(o *{{$tAlias.UpSingular}}, n int64) { o.C.{{$relAlias}} = &n })
	{{else -}}
	for _, o := range parents {
		if o == nil { continue }
		var count int64
		for _, r := range results {
			{{range $i, $name := $firstSide.FromColumns -}}
			{{$col := $table.GetColumn $name -}}
			{{$compare := $.Types.GetCompareExpr $.CurrentPackage $.Importer $col.Type $col.Nullable false -}}
			if !({{replace "AAA" (printf "o.%s" ($from.Column $name)) $compare | replace "BBB" (printf "r.K%d" $i)}}) { continue }
			{{end -}}
			count = r.Count
			break
		}
		o.C.{{$relAlias}} = &count
	}
	{{end -}}
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
