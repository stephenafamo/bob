{{if $.Relationships.Get $.Table.Key -}}
  {{$table := .Table}}
  {{$tAlias := .Aliases.Table $table.Key -}}
  {{$.Importer.Import "github.com/stephenafamo/bob"}}
  {{$.Importer.Import "github.com/stephenafamo/bob/mods"}}
  {{$.Importer.Import (printf "github.com/stephenafamo/bob/dialect/%s/dialect" $.Dialect)}}

  type {{$tAlias.DownSingular}}Joins[Q queryAliasable] struct {
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

  func build{{$tAlias.UpSingular}}Joins[Q queryAliasable](cols {{$tAlias.DownSingular}}Columns, typ string) {{$tAlias.DownSingular}}Joins[Q] {
    return {{$tAlias.DownSingular}}Joins[Q]{
      typ: typ,
      {{range $rel := $.Relationships.Get $table.Key -}}
        {{- $fAlias := $.Aliases.Table $rel.Foreign -}}
        {{- $relAlias := $tAlias.Relationship $rel.Name -}}
        {{$relAlias}}: modAs[Q, {{$fAlias.DownSingular}}Columns] {
          c: {{$fAlias.UpPlural}}.Columns,
          f: func(q Q, to {{$fAlias.DownSingular}}Columns) bob.Mod[Q] {
            mods := make(mods.QueryMods[Q], 0, {{len $rel.Sides}})
            parentAlias := cols.Alias()

            {{range $index, $side := $rel.Sides -}}
            {{- $from := $.Aliases.Table $side.From -}}
            {{- $fromCols := printf "%s.Columns" $from.UpPlural -}}
            {{- $to := $.Aliases.Table $side.To -}}
            {{- $toCols := printf "%s.Columns" $to.UpPlural -}}
            {
              from := {{$fromCols}}.AliasedAs(parentAlias)
              {{if ne $index (sub (len $rel.Sides) 1) -}}
              joined := {{$toCols}}.AliasedAs(q.NextQueryAlias())
              {{else -}}
              joined := to
              {{end -}}
              mods = append(mods, dialect.Join[Q](typ, {{$to.UpPlural}}.NameExpr().As(joined.Alias())).On(
                  {{range $i, $local := $side.FromColumns -}}
                    {{- $fromCol := index $from.Columns $local -}}
                    {{- $toCol := index $to.Columns (index $side.ToColumns $i) -}}
                    joined.{{$toCol}}.EQ(from.{{$fromCol}}),
                  {{- end}}
                  {{- range $where := $side.FromWhere}}
                    {{- $fromCol := index $from.Columns $where.Column}}
                    from.{{$fromCol}}.EQ({{$.Dialect}}.Arg({{quote $where.SQLValue}})),
                  {{- end}}
                  {{- range $where := $side.ToWhere}}
                    {{- $toCol := index $to.Columns $where.Column}}
                    joined.{{$toCol}}.EQ({{$.Dialect}}.Arg({{quote $where.SQLValue}})),
                  {{- end}}
              ))
              parentAlias = joined.Alias()
            }
            {{end}}

            return mods
          },
        },
      {{end}}
    }
  }
{{end -}}
