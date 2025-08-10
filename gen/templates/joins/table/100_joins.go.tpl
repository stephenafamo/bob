{{if $.Relationships.Get $.Table.Key -}}
  {{$table := .Table}}
  {{$tAlias := .Aliases.Table $table.Key -}}
  {{$.Importer.Import "github.com/stephenafamo/bob"}}
  {{$.Importer.Import "github.com/stephenafamo/bob/mods"}}
  {{$.Importer.Import (printf "github.com/stephenafamo/bob/dialect/%s/dialect" $.Dialect)}}

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
{{end -}}
