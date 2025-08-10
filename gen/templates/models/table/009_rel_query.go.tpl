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
		sm.InnerJoin({{$from.UpPlural}}.NameAs()).On(
		{{end -}}
			{{range $i, $local := $side.FromColumns -}}
				{{- $fromCol := index $from.Columns $local -}}
				{{- $toCol := index $to.Columns (index $side.ToColumns $i) -}}
				{{- if gt $index 0 -}}
				{{$to.UpSingular}}Columns.{{$toCol}}.EQ({{$from.UpSingular}}Columns.{{$fromCol}}),
				{{- else -}}
				sm.Where({{$to.UpSingular}}Columns.{{$toCol}}.EQ({{$.Dialect}}.Arg(o.{{$fromCol}}))),
				{{- end -}}
			{{- end}}
			{{- range $where := $side.FromWhere}}
				{{- $fromCol := index $from.Columns $where.Column}}
				{{if eq $index 0 -}}sm.Where({{end -}}
				{{$from.UpSingular}}Columns.{{$fromCol}}.EQ({{$.Dialect}}.Arg({{quote $where.SQLValue}})),
				{{- if eq $index 0 -}}),{{- end -}}
			{{- end}}
			{{- range $where := $side.ToWhere}}
				{{- $toCol := index $to.Columns $where.Column}}
				{{if eq $index 0 -}}sm.Where({{end -}}
				{{$to.UpSingular}}Columns.{{$toCol}}.EQ({{$.Dialect}}.Arg({{quote $where.SQLValue}})),
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
    for _, o := range os {
      if o == nil {
        continue
      }
      {{- range $index, $local := $firstSide.FromColumns -}}
        {{$fromCol := index $firstFrom.Columns $local}}
        pk{{$fromCol}} = append(pk{{$fromCol}}, o.{{$fromCol}})
      {{- end}}
    }
    PKArgExpr := psql.Select(sm.Columns(
      {{- range $index, $local := $firstSide.FromColumns -}}
        {{$column := $.Table.GetColumn $local}}
        {{$fromCol := index $firstFrom.Columns $local -}}
        psql.F("unnest", psql.Cast(psql.Arg(pk{{$fromCol}}), "{{$column.DBType}}[]")),
      {{- end}}
    ))
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
		sm.InnerJoin({{$from.UpPlural}}.NameAs()).On(
			{{range $i, $local := $side.FromColumns -}}
				{{- $foreign := index $side.ToColumns $i -}}
				{{- $fromCol := index $from.Columns $local -}}
				{{- $toCol := index $to.Columns $foreign -}}
				{{$to.UpSingular}}Columns.{{$toCol}}.EQ({{$from.UpSingular}}Columns.{{$fromCol}}),
			{{- end}}
			{{- range $where := $side.FromWhere}}
				{{- $fromCol := index $from.Columns $where.Column}}
				{{$from.UpSingular}}Columns.{{$fromCol}}.EQ({{$.Dialect}}.Arg({{quote $where.SQLValue}})),
			{{- end}}
			{{- range $where := $side.ToWhere}}
				{{- $toCol := index $to.Columns $where.Column}}
				{{$to.UpSingular}}Columns.{{$toCol}}.EQ({{$.Dialect}}.Arg({{quote $where.SQLValue}})),
			{{- end}}
		),
		{{- else -}}
			{{if gt (len $side.FromColumns) 0 -}}
				sm.Where({{$.Dialect}}.Group(
				{{- range $index, $local := $side.FromColumns -}}
					{{- $fromCol := index $from.Columns $local -}}
					{{- $toCol := index $to.Columns (index $side.ToColumns $index) -}}
					{{$to.UpSingular}}Columns.{{$toCol}},
				{{- end}}).OP("IN", PKArgExpr)),
			{{- end}}
			{{- range $where := $side.FromWhere}}
				{{- $fromCol := index $from.Columns $where.Column}}
				sm.Where({{$from.UpSingular}}Columns.{{$fromCol}}.EQ({{$.Dialect}}.Arg({{quote $where.SQLValue}}))),
			{{- end}}
			{{- range $where := $side.ToWhere}}
				{{- $toCol := index $to.Columns $where.Column}}
				sm.Where({{$to.UpSingular}}Columns.{{$toCol}}.EQ({{$.Dialect}}.Arg({{quote $where.SQLValue}}))),
			{{- end}}
		{{- end -}}
		{{- end}}
	)...)
}


{{end -}}
