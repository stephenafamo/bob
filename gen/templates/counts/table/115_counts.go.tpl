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

// LoadCount{{$relAlias}} loads the count of {{$relAlias}} for a slice in a single batch query
func (os {{$tAlias.UpSingular}}Slice) LoadCount{{$relAlias}}(ctx context.Context, exec bob.Executor, mods ...bob.Mod[*dialect.SelectQuery]) error {
	if len(os) == 0 {
		return nil
	}

	{{$firstSide := index $rel.Sides 0 -}}
	{{$firstFrom := $.Aliases.Table $firstSide.From -}}
	{{$firstTo := $.Aliases.Table $firstSide.To -}}

	// Build the IN arg expression from parent PKs
	{{- if ne $.Dialect "psql"}}
	PKArgSlice := make([]bob.Expression, 0, len(os))
	for _, o := range os {
		if o == nil {
			continue
		}
		PKArgSlice = append(PKArgSlice, {{$.Dialect}}.ArgGroup(
			{{- range $index, $local := $firstSide.FromColumns -}}
			{{- $fromCol := index $firstFrom.Columns $local -}}
			o.{{$fromCol}},
			{{- end -}}
		))
	}
	PKArgExpr := {{$.Dialect}}.Group(PKArgSlice...)
	{{- else}}
	{{$.Importer.Import "github.com/stephenafamo/bob/types/pgtypes" -}}
	{{- range $index, $local := $firstSide.FromColumns -}}
	{{- $column := $.Table.GetColumn $local -}}
	{{- $colTyp := $.Types.GetNullable $.CurrentPackage $.Importer $column.Type $column.Nullable -}}
	{{- $fromCol := index $firstFrom.Columns $local}}
	pk{{$fromCol}} := make(pgtypes.Array[{{$colTyp}}], 0, len(os))
	{{- end}}
	for _, o := range os {
		if o == nil {
			continue
		}
		{{- range $index, $local := $firstSide.FromColumns -}}
		{{- $fromCol := index $firstFrom.Columns $local}}
		pk{{$fromCol}} = append(pk{{$fromCol}}, o.{{$fromCol}})
		{{- end}}
	}
	PKArgExpr := {{$.Dialect}}.Select(sm.Columns(
		{{- range $index, $local := $firstSide.FromColumns -}}
		{{- $column := $.Table.GetColumn $local -}}
		{{- $fromCol := index $firstFrom.Columns $local}}
		{{$.Dialect}}.F("unnest", {{$.Dialect}}.Cast({{$.Dialect}}.Arg(pk{{$fromCol}}), "{{$column.DBType}}[]")),
		{{- end}}
	))
	{{- end}}

	// countResult holds one scanned row from the batch count query.
	// FK columns are aliased to the parent PK column names for direct map lookup.
	type countResult struct {
		{{range $index, $local := $firstSide.FromColumns -}}
		{{- $column := $.Table.GetColumn $local -}}
		{{- $colTyp := $.Types.GetNullable $.CurrentPackage $.Importer $column.Type $column.Nullable -}}
		{{- $fromCol := index $firstFrom.Columns $local}}
		{{$fromCol}} {{$colTyp}}
		{{end -}}
		Count int64
	}

	batchMods := []bob.Mod[*dialect.SelectQuery]{
		// SELECT fk AS parent_pk, count(*)
		sm.Columns(
			{{range $index, $local := $firstSide.FromColumns -}}
			{{$toLocal := index $firstSide.ToColumns $index -}}
			{{$firstToColAlias := index $firstTo.Columns $toLocal -}}
			{{$firstTo.UpPlural}}.Columns.{{$firstToColAlias}}.As({{quote $local}}),
			{{end -}}
			{{$.Dialect}}.Raw("count(*) as count"),
		),
		{{if eq (len $rel.Sides) 1 -}}
		// Single-hop: FROM related table directly
		sm.From({{$fAlias.UpPlural}}.NameAs()),
		{{range $where := $firstSide.ToWhere -}}
		{{$whereColAlias := index $firstTo.Columns $where.Column -}}
		sm.Where({{$firstTo.UpPlural}}.Columns.{{$whereColAlias}}.EQ({{$.Dialect}}.Arg({{quote $where.SQLValue}}))),
		{{end -}}
		{{- else -}}
		// Multi-hop: FROM first join table, JOIN through to final related table
		sm.From({{$firstTo.UpPlural}}.NameAs()),
		{{range $where := $firstSide.ToWhere -}}
		{{$whereColAlias := index $firstTo.Columns $where.Column -}}
		sm.Where({{$firstTo.UpPlural}}.Columns.{{$whereColAlias}}.EQ({{$.Dialect}}.Arg({{quote $where.SQLValue}}))),
		{{end -}}
		{{range $sideIndex, $side := $rel.Sides -}}
		{{if eq $sideIndex 0 -}}{{continue}}{{end -}}
		{{$sideFrom := $.Aliases.Table $side.From -}}
		{{$sideTo := $.Aliases.Table $side.To -}}
		sm.InnerJoin({{$sideTo.UpPlural}}.NameAs()).On(
			{{range $i, $fromColKey := $side.FromColumns -}}
			{{$toColKey := index $side.ToColumns $i -}}
			{{$sideToColAlias := index $sideTo.Columns $toColKey -}}
			{{$sideFromColAlias := index $sideFrom.Columns $fromColKey -}}
			{{$sideTo.UpPlural}}.Columns.{{$sideToColAlias}}.EQ({{$sideFrom.UpPlural}}.Columns.{{$sideFromColAlias}}),
			{{end -}}
			{{range $where := $side.FromWhere -}}
			{{$fromWhereColAlias := index $sideFrom.Columns $where.Column -}}
			{{$sideFrom.UpPlural}}.Columns.{{$fromWhereColAlias}}.EQ({{$.Dialect}}.Arg({{quote $where.SQLValue}})),
			{{end -}}
			{{range $where := $side.ToWhere -}}
			{{$toWhereColAlias := index $sideTo.Columns $where.Column -}}
			{{$sideTo.UpPlural}}.Columns.{{$toWhereColAlias}}.EQ({{$.Dialect}}.Arg({{quote $where.SQLValue}})),
			{{end -}}
		),
		{{end -}}
		{{- end}}
		// WHERE fk IN (parent PKs)
		sm.Where({{$.Dialect}}.Group(
			{{range $index, $local := $firstSide.FromColumns -}}
			{{$toLocal := index $firstSide.ToColumns $index -}}
			{{$firstToColAlias := index $firstTo.Columns $toLocal -}}
			{{$firstTo.UpPlural}}.Columns.{{$firstToColAlias}},
			{{end -}}
		).In(PKArgExpr)),
		// GROUP BY fk columns
		{{range $index, $local := $firstSide.FromColumns -}}
		{{$toLocal := index $firstSide.ToColumns $index -}}
		{{$firstToColAlias := index $firstTo.Columns $toLocal -}}
		sm.GroupBy({{$firstTo.UpPlural}}.Columns.{{$firstToColAlias}}),
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

	{{if eq (len $firstSide.FromColumns) 1 -}}
	{{$local := index $firstSide.FromColumns 0 -}}
	{{$column := $.Table.GetColumn $local -}}
	{{$colTyp := $.Types.GetNullable $.CurrentPackage $.Importer $column.Type $column.Nullable -}}
	{{$fromCol := index $firstFrom.Columns $local -}}
	// Single-column FK: direct map lookup
	countMap := make(map[{{$colTyp}}]int64, len(results))
	for _, r := range results {
		countMap[r.{{$fromCol}}] = r.Count
	}
	for _, o := range os {
		if o == nil {
			continue
		}
		count := countMap[o.{{$fromCol}}]
		o.C.{{$relAlias}} = &count
	}
	{{- else -}}
	// Composite FK: use a key struct
	type countKey struct {
		{{range $index, $local := $firstSide.FromColumns -}}
		{{- $column := $.Table.GetColumn $local -}}
		{{- $colTyp := $.Types.GetNullable $.CurrentPackage $.Importer $column.Type $column.Nullable -}}
		{{- $fromCol := index $firstFrom.Columns $local}}
		{{$fromCol}} {{$colTyp}}
		{{end -}}
	}
	countMap := make(map[countKey]int64, len(results))
	for _, r := range results {
		countMap[countKey{
			{{range $index, $local := $firstSide.FromColumns -}}
			{{- $fromCol := index $firstFrom.Columns $local}}
			{{$fromCol}}: r.{{$fromCol}},
			{{end -}}
		}] = r.Count
	}
	for _, o := range os {
		if o == nil {
			continue
		}
		count := countMap[countKey{
			{{range $index, $local := $firstSide.FromColumns -}}
			{{- $fromCol := index $firstFrom.Columns $local}}
			{{$fromCol}}: o.{{$fromCol}},
			{{end -}}
		}]
		o.C.{{$relAlias}} = &count
	}
	{{- end}}

	return nil
}

{{end -}}
{{end -}}
