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
{{$.Importer.Import "github.com/stephenafamo/scan"}}
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
		{{$relAlias}}: func(mods ...bob.Mod[*dialect.SelectQuery]) {{$.Dialect}}.Preloader {
			return orm.PreloadCount[*dialect.SelectQuery]("{{$relAlias}}", func(parent string) bob.Expression {
				if parent == "" { parent = {{$tAlias.UpPlural}}.Alias() }
				q := correlatedQuery({{$tAlias.DownSingular}}Rel{{$relAlias}}, parent, {{$.Dialect}}.Raw("count(*)"), mods)
				return {{$.Dialect}}.Group(q.Expression)
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
	{{$firstFrom := $.Aliases.Table $firstSide.From -}}
	{{$path := printf "%sPathFor%s" $tAlias.DownSingular $relAlias -}}

	{{if $firstSide.FromWhere}}matched := os.matching{{$relAlias}}Sources(){{end}}
	{{$comparable := true -}}
	// countResult holds one scanned row from the batch count query.
	// Keys that matched the SQL filter are non-NULL, even for nullable FKs.
	type countResult struct {
		{{range $index, $local := $firstSide.FromColumns -}}
		{{- $column := $.Table.GetColumn $local -}}
		{{- if not ($.Types.CanCompareWithEquals $.CurrentPackage $column.Type)}}{{$comparable = false}}{{end}}
		K{{$index}} {{$.Types.Get $.CurrentPackage $.Importer $column.Type}} `db:{{quote $local}}`
		{{end -}}
		Count int64
	}

	batchMods := []bob.Mod[*dialect.SelectQuery]{
		// SELECT fk AS parent_pk, count(*)
		sm.Columns(
			{{range $index, $local := $firstSide.FromColumns -}}
			{{$toLocal := index $firstSide.ToColumns $index -}}
			{{$.Dialect}}.Quote({{$path}}[0].Table.Alias, {{quote $toLocal}}).As({{quote $local}}),
			{{end -}}
			{{$.Dialect}}.Raw("count(*) as count"),
		),
		sm.From({{$path}}[0].Table),
		{{range $where := $firstSide.ToWhere -}}
		sm.Where({{$.Dialect}}.Quote({{$path}}[0].Table.Alias, {{quote $where.Column}}).EQ({{relWhereSQL $.Importer $where}})),
		{{end -}}
		{{range $sideIndex, $side := $rel.Sides -}}
		{{if eq $sideIndex 0 -}}{{continue}}{{end -}}
		sm.InnerJoin({{$path}}[{{$sideIndex}}].Table).On({{$path}}[{{$sideIndex}}].On...),
		{{- end}}
		{{if $firstSide.FromColumns -}}
		{{if $firstSide.FromWhere}}matched{{else}}os{{end}}.keysFor{{$relAlias}}(),
		{{end -}}
		// GROUP BY fk columns
		{{range $index, $local := $firstSide.FromColumns -}}
		{{$toLocal := index $firstSide.ToColumns $index -}}
		sm.GroupBy({{$.Dialect}}.Quote({{$path}}[0].Table.Alias, {{quote $toLocal}})),
		{{end -}}
	}
	batchMods = append(batchMods, mods...)

	results, err := bob.All(ctx, exec,
		{{$.Dialect}}.Select(batchMods...),
		scan.StructMapper[countResult](),
	)
	if err != nil {
		return err
	}

	{{if $firstSide.FromWhere -}}
	// Rows excluded by the source predicate have a loaded count of zero,
	// even if an included parent shares their join key.
	for _, o := range os { if o != nil { o.C.{{$relAlias}} = new(int64) } }
	os = matched
	{{end -}}
	{{if $comparable -}}
	type countKey struct {
		{{range $index, $local := $firstSide.FromColumns -}}
		{{- $column := $.Table.GetColumn $local -}}
		K{{$index}} {{$.Types.Get $.CurrentPackage $.Importer $column.Type}}
		{{end -}}
	}
	orm.AssignCounts(os, results,
		func(o *{{$tAlias.UpSingular}}) (countKey, bool) {
			var key countKey
			{{range $i, $name := $firstSide.FromColumns -}}
			{{$col := $.Table.GetColumn $name -}}
			{{$field := printf "o.%s" ($firstFrom.Column $name) -}}
			{{if $col.Nullable}}if !({{$.Types.GetNullTypeValid $.CurrentPackage $col.Type $field}}) { return key, false }{{end}}
			key.K{{$i}} = {{$.Types.UnwrapNullExpr $.CurrentPackage $.Importer $col.Type $field $col.Nullable}}
			{{end -}}
			return key, true
		},
		func(r countResult) (countKey, int64) { return countKey{ {{range $i, $_ := $firstSide.FromColumns}}r.K{{$i}},{{end}} }, r.Count },
		func(o *{{$tAlias.UpSingular}}, n int64) { o.C.{{$relAlias}} = &n })
	{{else -}}
	for _, o := range os {
		if o == nil { continue }
		var count int64
		for _, r := range results {
			{{range $i, $name := $firstSide.FromColumns -}}
			{{$col := $.Table.GetColumn $name -}}
			{{$compare := $.Types.GetCompareExpr $.CurrentPackage $.Importer $col.Type $col.Nullable false -}}
			if !({{replace "AAA" (printf "o.%s" ($firstFrom.Column $name)) $compare | replace "BBB" (printf "r.K%d" $i)}}) { continue }
			{{end -}}
			count = r.Count
			break
		}
		o.C.{{$relAlias}} = &count
	}
	{{- end}}

	return nil
}

{{end -}}
{{end -}}
