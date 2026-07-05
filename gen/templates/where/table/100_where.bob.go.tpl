{{$.Importer.Import (printf "github.com/stephenafamo/bob/dialect/%s" $.Dialect)}}

{{$table := .Table}}
{{$tAlias := .Aliases.Table $table.Key -}}

type {{$tAlias.DownSingular}}Where[Q {{$.Dialect}}.Filterable] struct {
	cols {{$tAlias.DownSingular}}Columns
	{{range $column := $table.Columns -}}
    {{- $colAlias := $tAlias.Column $column.Name -}}
    {{- $colTyp := $.Types.Get $.CurrentPackage $.Importer $column.Type -}}
		{{- if $column.Nullable -}}
			{{$colAlias}} {{$.Dialect}}.WhereNullMod[Q, {{$colTyp}}]
		{{- else -}}
			{{$colAlias}} {{$.Dialect}}.WhereMod[Q, {{$colTyp}}]
		{{- end}}
  {{end -}}
}

func ({{$tAlias.DownSingular}}Where[Q]) AliasedAs(alias string) {{$tAlias.DownSingular}}Where[Q] {
	return build{{$tAlias.UpSingular}}Where[Q](build{{$tAlias.UpSingular}}Columns(alias))
}

func build{{$tAlias.UpSingular}}Where[Q {{$.Dialect}}.Filterable](cols {{$tAlias.DownSingular}}Columns) {{$tAlias.DownSingular}}Where[Q] {
	return {{$tAlias.DownSingular}}Where[Q]{
			cols: cols,
			{{range $column := $table.Columns -}}
      {{- $colAlias := $tAlias.Column $column.Name -}}
      {{- $colTyp := $.Types.Get $.CurrentPackage $.Importer $column.Type -}}
				{{- if $column.Nullable -}}
					{{$colAlias}}: {{$.Dialect}}.WhereNull[Q, {{$colTyp}}](cols.{{$colAlias}}.Expression),
				{{- else -}}
					{{$colAlias}}: {{$.Dialect}}.Where[Q, {{$colTyp}}](cols.{{$colAlias}}.Expression),
				{{- end}}
			{{end -}}
	}
}

{{/* EXISTS semi-join filter helpers.
     Stage 1: to-one relationships.
     Stage 2: has-many relationships. Both are single-side; the FK direction is
       absorbed by FromColumns/ToColumns so the same code path serves both.
     Generates Has{Rel}(filters...) that adds a correlated EXISTS subquery
     instead of an INNER JOIN, so the parent rows are not multiplied.
     Self-referential relations (foreign table == parent table) alias the
     subquery table so the correlation still targets the parent row. */}}
{{range $rel := $.Relationships.Get $table.Key -}}
{{- if eq (len $rel.Sides) 1 -}}
{{- $.Importer.Import "github.com/stephenafamo/bob" -}}
{{- $.Importer.Import "github.com/stephenafamo/bob/mods" -}}
{{- $.Importer.Import (printf "github.com/stephenafamo/bob/dialect/%s/dialect" $.Dialect) -}}
{{- $.Importer.Import (printf "github.com/stephenafamo/bob/dialect/%s/sm" $.Dialect) -}}
{{- $relAlias := $tAlias.Relationship $rel.Name -}}
{{- $side := index $rel.Sides 0 -}}
{{- $from := $.Aliases.Table $side.From -}}
{{- $to := $.Aliases.Table $side.To -}}
{{- $selfRef := eq $side.From $side.To -}}
{{- $fcols := printf "%s.Columns" $to.UpPlural -}}
{{- if $selfRef -}}{{- $fcols = "relTbl" -}}{{- end -}}

// Has{{$relAlias}} filters parents that have a matching {{$relAlias}} using a
// correlated EXISTS subquery (semi-join). Unlike an INNER JOIN it does not
// multiply parent rows, so no DISTINCT is needed. The optional filters are
// applied to the subquery (i.e. to {{$to.UpPlural}}).
func (w {{$tAlias.DownSingular}}Where[Q]) Has{{$relAlias}}(filters ...bob.Mod[*dialect.SelectQuery]) mods.Where[Q] {
	{{if $selfRef -}}
	// self-referential relation: alias the subquery table so the correlation
	// columns still reference the parent row rather than the subquery's own row.
	relTbl := {{$to.UpPlural}}.Columns.AliasedAs("{{$relAlias}}")
	{{end -}}
	q := {{$.Dialect}}.Select(
		sm.Columns({{$.Dialect}}.Raw("1")),
		{{if $selfRef -}}
		sm.From({{$to.UpPlural}}.NameExpr().As("{{$relAlias}}")),
		{{else -}}
		sm.From({{$to.UpPlural}}.NameExpr()),
		{{end -}}
		{{range $i, $local := $side.FromColumns -}}
		{{- $fromCol := index $from.Columns $local -}}
		{{- $toCol := index $to.Columns (index $side.ToColumns $i) -}}
		sm.Where({{$fcols}}.{{$toCol}}.EQ(w.cols.{{$fromCol}})),
		{{end -}}
		{{range $where := $side.FromWhere -}}
		{{- $fromCol := index $from.Columns $where.Column -}}
		sm.Where(w.cols.{{$fromCol}}.EQ({{$.Dialect}}.Arg({{quote $where.SQLValue}}))),
		{{end -}}
		{{range $where := $side.ToWhere -}}
		{{- $toCol := index $to.Columns $where.Column -}}
		sm.Where({{$fcols}}.{{$toCol}}.EQ({{$.Dialect}}.Arg({{quote $where.SQLValue}}))),
		{{end -}}
	)
	q.Apply(filters...)
	return mods.Where[Q]{E: {{$.Dialect}}.Exists(q)}
}
{{end -}}
{{end -}}

{{/* Stage 3: many-to-many relationships (multiple sides, through join table(s)).
     The subquery selects FROM the first side's join table, joins the remaining
     sides, and correlates only the first side back to the parent row.
     A subquery table that resolves to the parent table (self-referential m2m)
     is aliased so the first-side correlation keeps targeting the parent row;
     other subquery tables keep their default alias so user filters resolve. */}}
{{range $rel := $.Relationships.Get $table.Key -}}
{{- if gt (len $rel.Sides) 1 -}}
{{- $.Importer.Import "github.com/stephenafamo/bob" -}}
{{- $.Importer.Import "github.com/stephenafamo/bob/mods" -}}
{{- $.Importer.Import (printf "github.com/stephenafamo/bob/dialect/%s/dialect" $.Dialect) -}}
{{- $.Importer.Import (printf "github.com/stephenafamo/bob/dialect/%s/sm" $.Dialect) -}}
{{- $relAlias := $tAlias.Relationship $rel.Name -}}
{{- $firstSide := index $rel.Sides 0 -}}
{{- $firstFrom := $.Aliases.Table $firstSide.From -}}
{{- $firstTo := $.Aliases.Table $firstSide.To -}}
{{- $lastSide := index $rel.Sides (sub (len $rel.Sides) 1) -}}
{{- $lastTo := $.Aliases.Table $lastSide.To -}}

// Has{{$relAlias}} filters parents that have a matching {{$relAlias}} (a
// many-to-many relationship) using a correlated EXISTS subquery instead of
// INNER JOINs, so the parent rows are not multiplied. The optional filters are
// applied to the subquery (i.e. to {{$lastTo.UpPlural}}).
func (w {{$tAlias.DownSingular}}Where[Q]) Has{{$relAlias}}(filters ...bob.Mod[*dialect.SelectQuery]) mods.Where[Q] {
	{{range $index, $side := $rel.Sides -}}
	{{- if and (ne $index 0) (eq $side.To $table.Key) -}}
	{{- $sTo := $.Aliases.Table $side.To -}}
	// self-referential: alias the subquery copy of the parent table so the
	// first-side correlation below still references the outer (parent) row.
	rel{{$index}} := {{$sTo.UpPlural}}.Columns.AliasedAs("{{$relAlias}}{{$index}}")
	{{end -}}
	{{end -}}
	q := {{$.Dialect}}.Select(
		sm.Columns({{$.Dialect}}.Raw("1")),
		sm.From({{$firstTo.UpPlural}}.NameExpr()),
		{{range $index, $side := $rel.Sides -}}
		{{- if ne $index 0 -}}
		{{- $sFrom := $.Aliases.Table $side.From -}}
		{{- $sTo := $.Aliases.Table $side.To -}}
		{{- $sSelf := eq $side.To $table.Key -}}
		{{- $sToRef := printf "%s.Columns" $sTo.UpPlural -}}
		{{- if $sSelf -}}{{- $sToRef = printf "rel%d" $index -}}{{- end -}}
		sm.InnerJoin({{$sTo.UpPlural}}.NameExpr(){{if $sSelf}}.As("{{$relAlias}}{{$index}}"){{end}}).On(
			{{range $i, $local := $side.FromColumns -}}
			{{- $fromCol := index $sFrom.Columns $local -}}
			{{- $toCol := index $sTo.Columns (index $side.ToColumns $i) -}}
			{{$sToRef}}.{{$toCol}}.EQ({{$sFrom.UpPlural}}.Columns.{{$fromCol}}),
			{{- end}}
			{{- range $where := $side.ToWhere}}
			{{- $toCol := index $sTo.Columns $where.Column}}
			{{$sToRef}}.{{$toCol}}.EQ({{$.Dialect}}.Arg({{quote $where.SQLValue}})),
			{{- end}}
		),
		{{end -}}
		{{end -}}
		{{range $i, $local := $firstSide.FromColumns -}}
		{{- $fromCol := index $firstFrom.Columns $local -}}
		{{- $toCol := index $firstTo.Columns (index $firstSide.ToColumns $i) -}}
		sm.Where({{$firstTo.UpPlural}}.Columns.{{$toCol}}.EQ(w.cols.{{$fromCol}})),
		{{end -}}
		{{range $where := $firstSide.FromWhere -}}
		{{- $fromCol := index $firstFrom.Columns $where.Column -}}
		sm.Where(w.cols.{{$fromCol}}.EQ({{$.Dialect}}.Arg({{quote $where.SQLValue}}))),
		{{end -}}
		{{range $where := $firstSide.ToWhere -}}
		{{- $toCol := index $firstTo.Columns $where.Column -}}
		sm.Where({{$firstTo.UpPlural}}.Columns.{{$toCol}}.EQ({{$.Dialect}}.Arg({{quote $where.SQLValue}}))),
		{{end -}}
	)
	q.Apply(filters...)
	return mods.Where[Q]{E: {{$.Dialect}}.Exists(q)}
}
{{end -}}
{{end -}}
