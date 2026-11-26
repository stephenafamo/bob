{{$table := .Table}}
{{$tAlias := .Aliases.Table $table.Key -}}
{{if $.Relationships.Get $table.Key -}}
  {{$.Importer.Import "github.com/stephenafamo/bob"}}
  {{$.Importer.Import "context"}}
  {{$.Importer.Import (printf "github.com/stephenafamo/bob/dialect/%s/sm" $.Dialect)}}
{{end}}

{{range $rel := $.Relationships.Get $table.Key -}}
{{- $fAlias := $.Aliases.Table $rel.Foreign -}}
{{- $relAlias := $tAlias.Relationship $rel.Name -}}
{{- $path := printf "%sPathFor%s" $tAlias.DownSingular $relAlias -}}
// {{$relAlias}} starts a query for related objects on {{$rel.Foreign}}
func (o *{{$tAlias.UpSingular}}) {{relQueryMethodName $tAlias $relAlias}}(mods ...bob.Mod[dialect.SelectEditor]) {{$fAlias.UpPlural}}Query {
	return {{$fAlias.UpPlural}}.Query(append(mods,
		{{- range $index := until (len $rel.Sides) | reverse -}}
		{{/* Index counts down */}}
		{{/* This also flips the meaning of $from and $to */}}
		{{- $side := index $rel.Sides $index -}}
		{{- $from := $.Aliases.Table $side.From -}}
		{{- if gt $index 0 -}}
		sm.InnerJoin({{$path}}[{{sub $index 1}}].Table).On({{$path}}[{{$index}}].On...),
		{{- else -}}
			{{range $i, $local := $side.FromColumns -}}
				{{- $fromCol := index $from.Columns $local -}}
				sm.Where({{$.Dialect}}.Quote({{$path}}[0].Table.Alias, {{quote (index $side.ToColumns $i)}}).EQ({{$.Dialect}}.Arg(o.{{$fromCol}}))),
			{{- end}}
			{{- range $where := $side.FromWhere}}
			sm.Where({{$.Dialect}}.Arg(o.{{$from.Column $where.Column}}).EQ({{relWhereSQL $.Importer $where}})),
			{{- end}}
			{{- range $where := $side.ToWhere}}
				sm.Where({{$.Dialect}}.Quote({{$path}}[0].Table.Alias, {{quote $where.Column}}).EQ({{relWhereSQL $.Importer $where}})),
			{{- end}}
		{{- end}}
		{{- end}}
	)...)
}

{{$firstSide := (index $rel.Sides 0) -}}
{{$firstFrom := $.Aliases.Table $firstSide.From -}}
{{$arrayKey := false -}}
{{if eq $.Dialect "psql" -}}
	{{range $name := $firstSide.ToColumns -}}
		{{if hasSuffix "[]" ($.Tables.GetColumn $firstSide.To $name).DBType -}}{{$arrayKey = true}}{{break}}{{end -}}
	{{end -}}
{{end -}}


{{if $firstSide.FromWhere -}}
// Only these source rows participate in the relationship, even if other
// rows in the slice have the same join key.
func (os {{$tAlias.UpSingular}}Slice) matching{{$relAlias}}Sources() {{$tAlias.UpSingular}}Slice {
  matched := make({{$tAlias.UpSingular}}Slice, 0, len(os))
  for _, o := range os {
    if o == nil { continue }
    {{range $where := $firstSide.FromWhere -}}
      {{$col := $table.GetColumn $where.Column -}}
      {{$compare := $.Types.GetCompareExpr $.CurrentPackage $.Importer $col.Type $col.Nullable false -}}
      if !({{replace "AAA" (printf "o.%s" ($firstFrom.Column $col.Name)) $compare | replace "BBB" (printf "(%s)" $where.GoValue)}}) { continue }
    {{end -}}
    matched = append(matched, o)
  }
  return matched
}
{{end -}}


{{if $firstSide.FromColumns -}}
// keysFor{{$relAlias}} filters the first join table by this batch of source keys.
func (os {{$tAlias.UpSingular}}Slice) keysFor{{$relAlias}}() bob.Mod[dialect.SelectEditor] {
	{{if and (eq $.Dialect "psql") (not $arrayKey) -}}
	{{$.Importer.Import "github.com/stephenafamo/bob/dialect/psql/psqlrel"}}
	return psqlrel.FilterRelationshipKeys({{$path}}[0].Table,
		{{range $i, $name := $firstSide.FromColumns -}}
		{{$col := $table.GetColumn $name -}}
		{{$type := $.Types.GetNullable $.CurrentPackage $.Importer $col.Type $col.Nullable -}}
		{{$dedup := and (eq (len $firstSide.FromColumns) 1) (not ($table.HasExactUnique $name)) ($.Types.CanCompareWithEquals $.CurrentPackage $col.Type) -}}
		psqlrel.BatchKey{Column: {{quote (index $firstSide.ToColumns $i)}},
			Values: psqlrel.{{if $dedup}}UniqueKeyValues{{else}}KeyValues{{end}}(os, func(o *{{$tAlias.UpSingular}}) {{$type}} { return o.{{$firstFrom.Column $name}} })},
		{{end -}}
	)
	{{else -}}
	keys := make([]bob.Expression, 0, len(os))
	for _, o := range os {
		if o != nil { keys = append(keys, {{$.Dialect}}.ArgGroup({{range $firstSide.FromColumns}}o.{{$firstFrom.Column .}},{{end}})) }
	}
	if len(keys) == 0 { return sm.Where({{$.Dialect}}.Raw("1 = 0")) }
	return sm.Where({{$.Dialect}}.Group({{range $firstSide.ToColumns}}{{$.Dialect}}.Quote({{$path}}[0].Table.Alias, {{quote .}}),{{end}}).OP("IN", {{$.Dialect}}.Group(keys...)))
	{{end -}}
}
{{end -}}

func (os {{$tAlias.UpSingular}}Slice) {{relQueryMethodName $tAlias $relAlias}}(mods ...bob.Mod[dialect.SelectEditor]) {{$fAlias.UpPlural}}Query {
  {{if $firstSide.FromWhere -}}
    os = os.matching{{$relAlias}}Sources()
    if len(os) == 0 {
      mods = append(mods, sm.Where({{$.Dialect}}.Raw("1 = 0")))
    }
  {{end -}}

	return {{$fAlias.UpPlural}}.Query(append(mods,
		{{- range $index := until (len $rel.Sides) | reverse -}}
		{{/* Index counts down */}}
		{{/* This also flips the meaning of $from and $to */}}
		{{- $side := index $rel.Sides $index -}}
		{{- if gt $index 0 -}}
		sm.InnerJoin({{$path}}[{{sub $index 1}}].Table).On({{$path}}[{{$index}}].On...),
		{{- else -}}
			{{if $side.FromColumns}}os.keysFor{{$relAlias}}(),{{end}}
			{{- range $where := $side.ToWhere}}
				sm.Where({{$.Dialect}}.Quote({{$path}}[0].Table.Alias, {{quote $where.Column}}).EQ({{relWhereSQL $.Importer $where}})),
			{{- end}}
		{{- end -}}
		{{- end}}
	)...)
}


{{end -}}
