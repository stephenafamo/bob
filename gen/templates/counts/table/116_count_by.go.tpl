{{- $table := .Table -}}
{{- $tAlias := .Aliases.Table $table.Key -}}
{{- $rels := $.Relationships.Get $table.Key -}}
{{- $hasToMany := false -}}
{{- range $rel := $rels -}}
	{{- if $rel.IsToMany -}}{{- $hasToMany = true -}}{{- end -}}
{{- end -}}

{{if $hasToMany -}}
{{range $rel := $rels -}}
{{- if not $rel.IsToMany}}{{continue}}{{end -}}
{{- $fAlias := $.Aliases.Table $rel.Foreign -}}
{{- $relAlias := $tAlias.Relationship $rel.Name -}}
{{- $firstSide := index $rel.Sides 0 -}}
{{- $firstFrom := $.Aliases.Table $firstSide.From -}}
{{- $firstTo := $.Aliases.Table $firstSide.To -}}

// Only generate for to-many relationships with a single-column parent key.
{{- if ne (len $firstSide.FromColumns) 1}}{{continue}}{{end -}}
{{- $local := index $firstSide.FromColumns 0 -}}
{{- $column := $.Table.GetColumn $local -}}
{{- $colTyp := $.Types.GetNullable $.CurrentPackage $.Importer $column.Type $column.Nullable -}}
{{- $fromCol := index $firstFrom.Columns $local -}}
{{- $toLocal := index $firstSide.ToColumns 0 -}}
{{- $firstToColAlias := index $firstTo.Columns $toLocal -}}
// Imports are inside the loop to avoid unused imports for composite-key-only tables.
{{$.Importer.Import "context"}}
{{$.Importer.Import "github.com/stephenafamo/bob"}}
{{$.Importer.Import "github.com/stephenafamo/scan"}}
{{$.Importer.Import (printf "github.com/stephenafamo/bob/dialect/%s" $.Dialect)}}
{{$.Importer.Import (printf "github.com/stephenafamo/bob/dialect/%s/dialect" $.Dialect)}}
{{$.Importer.Import (printf "github.com/stephenafamo/bob/dialect/%s/sm" $.Dialect)}}

func LoadCount{{$relAlias}}By{{$tAlias.UpSingular}}[G comparable](
	ctx context.Context, exec bob.Executor, os {{$tAlias.UpSingular}}Slice,
	groupCol bob.Expression, mods ...bob.Mod[*dialect.SelectQuery],
) (map[{{$colTyp}}]map[G]int64, error) {
	if len(os) == 0 {
		return map[{{$colTyp}}]map[G]int64{}, nil
	}

	// Build the parent-PK arg expression.
	{{- if ne $.Dialect "psql"}}
	PKArgSlice := make([]bob.Expression, 0, len(os))
	for _, o := range os {
		if o == nil {
			continue
		}
		PKArgSlice = append(PKArgSlice, {{$.Dialect}}.Arg(o.{{$fromCol}}))
	}
	PKArgExpr := {{$.Dialect}}.Group(PKArgSlice...)
	{{- else}}
	{{$.Importer.Import "github.com/stephenafamo/bob/types/pgtypes" -}}
	// psql: = ANY(array) so the planner can use the FK index (avoids IN(unnest) Seq Scan).
	pk{{$fromCol}} := make(pgtypes.Array[{{$colTyp}}], 0, len(os))
	for _, o := range os {
		if o == nil {
			continue
		}
		pk{{$fromCol}} = append(pk{{$fromCol}}, o.{{$fromCol}})
	}
	PKArgExpr := {{$.Dialect}}.Any({{$.Dialect}}.Cast({{$.Dialect}}.Arg(pk{{$fromCol}}), "{{$column.DBType}}[]"))
	{{- end}}

	batchMods := []bob.Mod[*dialect.SelectQuery]{
		sm.Columns(
			{{$firstTo.UpPlural}}.Columns.{{$firstToColAlias}},
			groupCol,
			{{$.Dialect}}.Raw("count(*)"),
		),
		{{if eq (len $rel.Sides) 1 -}}
		// Single-hop: FROM related table directly
		sm.From({{$fAlias.UpPlural}}.NameAsExpr()),
		{{range $where := $firstSide.ToWhere -}}
		{{$whereColAlias := index $firstTo.Columns $where.Column -}}
		sm.Where({{$firstTo.UpPlural}}.Columns.{{$whereColAlias}}.EQ({{$.Dialect}}.Arg({{quote $where.SQLValue}}))),
		{{end -}}
		{{- else -}}
		// Multi-hop: FROM first join table, JOIN through to the final related table
		sm.From({{$firstTo.UpPlural}}.NameAsExpr()),
		{{range $where := $firstSide.ToWhere -}}
		{{$whereColAlias := index $firstTo.Columns $where.Column -}}
		sm.Where({{$firstTo.UpPlural}}.Columns.{{$whereColAlias}}.EQ({{$.Dialect}}.Arg({{quote $where.SQLValue}}))),
		{{end -}}
		{{range $sideIndex, $side := $rel.Sides -}}
		{{if eq $sideIndex 0 -}}{{continue}}{{end -}}
		{{$sideFrom := $.Aliases.Table $side.From -}}
		{{$sideTo := $.Aliases.Table $side.To -}}
		sm.InnerJoin({{$sideTo.UpPlural}}.NameAsExpr()).On(
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
		{{if eq $.Dialect "psql" -}}
		sm.Where({{$firstTo.UpPlural}}.Columns.{{$firstToColAlias}}.EQ(PKArgExpr)),
		{{- else -}}
		sm.Where({{$firstTo.UpPlural}}.Columns.{{$firstToColAlias}}.OP("IN", PKArgExpr)),
		{{- end}}
		sm.GroupBy({{$.Dialect}}.Raw("1")),
		sm.GroupBy({{$.Dialect}}.Raw("2")),
	}
	batchMods = append(batchMods, mods...)

	type countByRow struct {
		PK    {{$colTyp}}
		Group G
		Count int64
	}
	var mapper scan.Mapper[*countByRow] = func(context.Context, []string) (scan.BeforeFunc, func(any) (*countByRow, error)) {
		return func(row *scan.Row) (any, error) {
				r := new(countByRow)
				row.ScheduleScanByIndex(0, &r.PK)
				row.ScheduleScanByIndex(1, &r.Group)
				row.ScheduleScanByIndex(2, &r.Count)
				return r, nil
			}, func(v any) (*countByRow, error) {
				return v.(*countByRow), nil
			}
	}

	results, err := bob.All(ctx, exec, {{$.Dialect}}.Select(batchMods...), mapper)
	if err != nil {
		return nil, err
	}

	out := make(map[{{$colTyp}}]map[G]int64, len(os))
	for _, r := range results {
		m, ok := out[r.PK]
		if !ok {
			m = make(map[G]int64)
			out[r.PK] = m
		}
		m[r.Group] = r.Count
	}
	return out, nil
}

{{end -}}
{{end -}}
