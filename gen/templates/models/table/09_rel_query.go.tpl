{{$table := .Table}}
{{$tAlias := .Aliases.Table $table.Key -}}
{{if $.Relationships.Get $table.Key -}}
  {{$.Importer.Import "github.com/stephenafamo/bob"}}
  {{$.Importer.Import "github.com/stephenafamo/bob/mods"}}
  {{$.Importer.Import "context"}}
  type {{$tAlias.DownSingular}}Joins[Q dialect.Joinable] struct {
    typ string
    {{range $.Relationships.Get $table.Key -}}
    {{- $relAlias := $tAlias.Relationship .Name -}}
    {{- $fAlias := $.Aliases.Table .Foreign -}}
    {{$relAlias}} modAs[Q, {{$fAlias.DownSingular}}Columns]
    {{end -}}
  }

  func (j {{$tAlias.DownSingular}}Joins[Q]) aliasedAs(alias string) {{$tAlias.DownSingular}}Joins[Q] {
    return build{{$tAlias.UpSingular}}Joins[Q](build{{$tAlias.UpSingular}}Columns(alias), j.typ)
  }

  func build{{$tAlias.UpSingular}}Joins[Q dialect.Joinable](cols {{$tAlias.DownSingular}}Columns, typ string) {{$tAlias.DownSingular}}Joins[Q] {
    return {{$tAlias.DownSingular}}Joins[Q]{
      typ: typ,
      {{range $rel := $.Relationships.Get $table.Key -}}
        {{- $fAlias := $.Aliases.Table $rel.Foreign -}}
        {{- $relAlias := $tAlias.Relationship $rel.Name -}}
        {{$relAlias}}: modAs[Q, {{$fAlias.DownSingular}}Columns] {
          c: {{$fAlias.UpSingular}}Columns,
          f: func(to {{$fAlias.DownSingular}}Columns) bob.Mod[Q] {
            {{if gt (len $rel.Sides) 1 -}}{{$.Importer.Import "strconv" -}}
              random := strconv.FormatInt(randInt(), 10)
            {{- end}}
            mods := make(mods.QueryMods[Q], 0, {{len $rel.Sides}})

            {{range $index, $side := $rel.Sides -}}
            {{- $from := $.Aliases.Table $side.From -}}
            {{- $fromCols := printf "%sColumns" $from.UpSingular -}}
            {{- $to := $.Aliases.Table $side.To -}}
            {{- $toCols := printf "%sColumns" $to.UpSingular -}}
            {{- $toTable := $.Tables.Get $side.To -}}
            {
              {{if ne $index 0 -}}
              cols := {{$fromCols}}.AliasedAs({{$fromCols}}.Alias() + random)
              {{end -}}
              {{if ne $index (sub (len $rel.Sides) 1) -}}
              to := {{$toCols}}.AliasedAs({{$toCols}}.Alias() + random)
              {{end -}}
              mods = append(mods, dialect.Join[Q](typ, {{$to.UpPlural}}.Name().As(to.Alias())).On(
                  {{range $i, $local := $side.FromColumns -}}
                    {{- $fromCol := index $from.Columns $local -}}
                    {{- $toCol := index $to.Columns (index $side.ToColumns $i) -}}
                    to.{{$toCol}}.EQ(cols.{{$fromCol}}),
                  {{- end}}
                  {{- range $where := $side.FromWhere}}
                    {{- $fromCol := index $from.Columns $where.Column}}
                    cols.{{$fromCol}}.EQ({{$.Dialect}}.Arg({{quote $where.SQLValue}})),
                  {{- end}}
                  {{- range $where := $side.ToWhere}}
                    {{- $toCol := index $to.Columns $where.Column}}
                    to.{{$toCol}}.EQ({{$.Dialect}}.Arg({{quote $where.SQLValue}})),
                  {{- end}}
              ))
            }
            {{end}}

            return mods
          },
        },
      {{end}}
    }
  }
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
      pk{{$fromCol}} := make(pgtypes.Array[{{$colTyp}}], len(os))
		{{- end}}
    for i, o := range os {
      {{- range $index, $local := $firstSide.FromColumns -}}
        {{$fromCol := index $firstFrom.Columns $local}}
        pk{{$fromCol}}[i] = o.{{$fromCol}}
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
