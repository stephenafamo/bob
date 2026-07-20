{{$table := .Table}}
{{$tAlias := .Aliases.Table $table.Key -}}
{{if $.Relationships.Get $table.Key -}}
  {{$.Importer.Import "github.com/stephenafamo/bob"}}
  {{$.Importer.Import "context"}}
{{end}}

{{range $rel := $.Relationships.Get $table.Key -}}
{{- $fAlias := $.Aliases.Table $rel.Foreign -}}
{{- $relAlias := $tAlias.Relationship $rel.Name -}}
// {{$relAlias}} starts a query for related objects on {{$rel.Foreign}}
func (o *{{$tAlias.UpSingular}}) {{relQueryMethodName $tAlias $relAlias}}(mods ...bob.Mod[*dialect.SelectQuery]) {{$fAlias.UpPlural}}Query {
	return {{$fAlias.UpPlural}}.Query(append(mods,
		{{- range $index := until (len $rel.Sides) | reverse -}}
		{{/* Index counts down */}}
		{{/* This also flips the meaning of $from and $to */}}
		{{- $side := index $rel.Sides $index -}}
		{{- $from := $.Aliases.Table $side.From -}}
		{{- $to := $.Aliases.Table $side.To -}}
		{{- $fromTable := $.Tables.Get $side.From -}}
		{{- if gt $index 0 -}}
		sm.InnerJoin({{$from.UpPlural}}.NameAsExpr()).On(
		{{end -}}
			{{range $i, $local := $side.FromColumns -}}
				{{- $fromCol := index $from.Columns $local -}}
				{{- $toCol := index $to.Columns (index $side.ToColumns $i) -}}
				{{- if gt $index 0 -}}
				{{$to.UpPlural}}.Columns.{{$toCol}}.EQ({{$from.UpPlural}}.Columns.{{$fromCol}}),
				{{- else -}}
				sm.Where({{$to.UpPlural}}.Columns.{{$toCol}}.EQ({{$.Dialect}}.Arg(o.{{$fromCol}}))),
				{{- end -}}
			{{- end}}
			{{- range $where := $side.FromWhere}}
				{{- $fromCol := index $from.Columns $where.Column}}
				{{if eq $index 0 -}}sm.Where({{end -}}
				{{$from.UpPlural}}.Columns.{{$fromCol}}.EQ({{$.Dialect}}.Arg({{quote $where.SQLValue}})),
				{{- if eq $index 0 -}}),{{- end -}}
			{{- end}}
			{{- range $where := $side.ToWhere}}
				{{- $toCol := index $to.Columns $where.Column}}
				{{if eq $index 0 -}}sm.Where({{end -}}
				{{$to.UpPlural}}.Columns.{{$toCol}}.EQ({{$.Dialect}}.Arg({{quote $where.SQLValue}})),
				{{- if eq $index 0 -}}),{{- end -}}
			{{- end}}
		{{- if gt $index 0 -}}
		),
		{{- end -}}
		{{- end}}
	)...)
}

{{$firstSide := (index $rel.Sides 0) -}}
{{$firstFrom := $.Aliases.Table $firstSide.From -}}
{{$firstTo := $.Aliases.Table $firstSide.To -}}

{{$lastSide := (index $rel.Sides (sub (len $rel.Sides) 1)) -}}
{{$lastFrom := $.Aliases.Table $lastSide.From -}}
{{$lastTo := $.Aliases.Table $lastSide.To -}}

func (os {{$tAlias.UpSingular}}Slice) {{relQueryMethodName $tAlias $relAlias}}(mods ...bob.Mod[*dialect.SelectQuery]) {{$fAlias.UpPlural}}Query {
  {{if gt (len $firstSide.FromColumns) 0 -}}
  {{if (ne $.Dialect "psql")}}
    PKArgSlice := make([]bob.Expression, len(os))
    for i, o := range os {
      PKArgSlice[i] = {{$.Dialect}}.ArgGroup(
      {{- range $index, $local := $firstSide.FromColumns -}}
        {{- $fromCol := index $firstFrom.Columns $local -}}
        o.{{$fromCol}},
      {{- end -}})
    }
    PKArgExpr := {{$.Dialect}}.Group(PKArgSlice...)
  {{else}}
    {{$.Importer.Import "github.com/stephenafamo/bob/types/pgtypes"}}
    {{$.Importer.Import "github.com/stephenafamo/bob/dialect/psql/sm"}}
		{{- range $index, $local := $firstSide.FromColumns -}}
      {{ $column := $.Table.GetColumn $local }}
      {{ $colTyp := $.Types.GetNullable $.CurrentPackage $.Importer $column.Type $column.Nullable }}
			{{$fromCol := index $firstFrom.Columns $local -}}
      pk{{$fromCol}} := make(pgtypes.Array[{{$colTyp}}], 0, len(os))
		{{- end}}
    {{if eq (len $firstSide.FromColumns) 1}}
    {{- $local := index $firstSide.FromColumns 0}}
    {{- $column := $.Table.GetColumn $local}}
    {{- $colTyp := $.Types.GetNullable $.CurrentPackage $.Importer $column.Type $column.Nullable}}
    {{- $fromCol := index $firstFrom.Columns $local}}
    {{- /* keys that are unique by construction (the parent's own PK or a
           unique column) never contain duplicates, so the seen-map would be
           pure overhead; dedup only de-duplicatable, ==-comparable keys */ -}}
    {{- $canDedup := and (not ($.Table.HasExactUnique $local)) ($.Types.CanCompareWithEquals $.CurrentPackage $column.Type)}}
    {{- if $canDedup}}
    // the array is only a filter (semi-join), so duplicate keys can be
    // dropped before they are sent over the wire
    seen{{$fromCol}} := make(map[{{$colTyp}}]struct{}, len(os))
    for _, o := range os {
      if o == nil {
        continue
      }
      if _, ok := seen{{$fromCol}}[o.{{$fromCol}}]; ok {
        continue
      }
      seen{{$fromCol}}[o.{{$fromCol}}] = struct{}{}
      pk{{$fromCol}} = append(pk{{$fromCol}}, o.{{$fromCol}})
    }
    {{- else}}
    for _, o := range os {
      if o == nil {
        continue
      }
      pk{{$fromCol}} = append(pk{{$fromCol}}, o.{{$fromCol}})
    }
    {{- end}}
    PKArgExpr := psql.Any(psql.Cast(psql.Arg(pk{{$fromCol}}), "{{$column.DBType}}[]"))
    {{else}}
    for _, o := range os {
      if o == nil {
        continue
      }
      {{- range $index, $local := $firstSide.FromColumns -}}
        {{$fromCol := index $firstFrom.Columns $local}}
        pk{{$fromCol}} = append(pk{{$fromCol}}, o.{{$fromCol}})
      {{- end}}
    }
    {{end}}
  {{end}}
	{{- end}}


	return {{$fAlias.UpPlural}}.Query(append(mods,
		{{- range $index := until (len $rel.Sides) | reverse -}}
		{{/* Index counts down */}}
		{{/* This also flips the meaning of $from and $to */}}
		{{- $side := index $rel.Sides $index -}}
		{{- $from := $.Aliases.Table $side.From -}}
		{{- $to := $.Aliases.Table $side.To -}}
		{{- $fromTable := $.Tables.Get $side.From -}}
		{{- if gt $index 0 -}}
		sm.InnerJoin({{$from.UpPlural}}.NameAsExpr()).On(
			{{range $i, $local := $side.FromColumns -}}
				{{- $foreign := index $side.ToColumns $i -}}
				{{- $fromCol := index $from.Columns $local -}}
				{{- $toCol := index $to.Columns $foreign -}}
				{{$to.UpPlural}}.Columns.{{$toCol}}.EQ({{$from.UpPlural}}.Columns.{{$fromCol}}),
			{{- end}}
			{{- range $where := $side.FromWhere}}
				{{- $fromCol := index $from.Columns $where.Column}}
				{{$from.UpPlural}}.Columns.{{$fromCol}}.EQ({{$.Dialect}}.Arg({{quote $where.SQLValue}})),
			{{- end}}
			{{- range $where := $side.ToWhere}}
				{{- $toCol := index $to.Columns $where.Column}}
				{{$to.UpPlural}}.Columns.{{$toCol}}.EQ({{$.Dialect}}.Arg({{quote $where.SQLValue}})),
			{{- end}}
		),
		{{- else -}}
			{{if gt (len $side.FromColumns) 0 -}}
				{{if and (eq $.Dialect "psql") (eq (len $side.FromColumns) 1) -}}
					{{- $toCol := index $to.Columns (index $side.ToColumns 0) -}}
					sm.Where({{$to.UpPlural}}.Columns.{{$toCol}}.EQ(PKArgExpr)),
				{{- else if eq $.Dialect "psql" -}}
					sm.InnerJoin(psql.Select(
						sm.Distinct(),
						sm.Columns(
							{{- range $index, $local := $side.FromColumns -}}
							{{- $toCol := index $to.Columns (index $side.ToColumns $index) -}}
							psql.Quote("bob_rel_keys_src", {{quote $toCol}}),
							{{- end}}
						),
						sm.From(psql.F("unnest",
							{{- range $index, $local := $side.FromColumns -}}
							{{- $fromCol := index $from.Columns $local -}}
							{{- $column := $.Table.GetColumn $local -}}
							psql.Cast(psql.Arg(pk{{$fromCol}}), "{{$column.DBType}}[]"),
							{{- end}}
						)).As("bob_rel_keys_src"
							{{- range $index, $local := $side.FromColumns -}}
							{{- $toCol := index $to.Columns (index $side.ToColumns $index) -}}
							, {{quote $toCol}}
							{{- end -}}
						),
					)).As("bob_rel_keys").On(
						{{- range $index, $local := $side.FromColumns -}}
						{{- $toCol := index $to.Columns (index $side.ToColumns $index) -}}
						{{$to.UpPlural}}.Columns.{{$toCol}}.EQ(psql.Quote("bob_rel_keys", {{quote $toCol}})),
						{{- end}}
					),
				{{- else -}}
					sm.Where({{$.Dialect}}.Group(
					{{- range $index, $local := $side.FromColumns -}}
						{{- $fromCol := index $from.Columns $local -}}
						{{- $toCol := index $to.Columns (index $side.ToColumns $index) -}}
						{{$to.UpPlural}}.Columns.{{$toCol}},
					{{- end}}).OP("IN", PKArgExpr)),
				{{- end}}
			{{- end}}
			{{- range $where := $side.FromWhere}}
				{{- $fromCol := index $from.Columns $where.Column}}
				sm.Where({{$from.UpPlural}}.Columns.{{$fromCol}}.EQ({{$.Dialect}}.Arg({{quote $where.SQLValue}}))),
			{{- end}}
			{{- range $where := $side.ToWhere}}
				{{- $toCol := index $to.Columns $where.Column}}
				sm.Where({{$to.UpPlural}}.Columns.{{$toCol}}.EQ({{$.Dialect}}.Arg({{quote $where.SQLValue}}))),
			{{- end}}
		{{- end -}}
		{{- end}}
	)...)
}


{{end -}}
