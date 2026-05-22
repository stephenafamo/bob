{{if $.Relationships.Get $.Table.Key -}}
  {{$table := .Table}}
  {{$tAlias := .Aliases.Table $table.Key -}}
  {{$.Importer.Import "github.com/stephenafamo/bob"}}
  {{$.Importer.Import "github.com/stephenafamo/bob/mods"}}
  {{$.Importer.Import (printf "github.com/stephenafamo/bob/dialect/%s/dialect" $.Dialect)}}

  type {{$tAlias.UpSingular}}Joins[Q dialect.Joinable] struct {
    typ string
    {{range $.Relationships.Get $table.Key -}}
    {{- $relAlias := $tAlias.Relationship .Name -}}
    {{- $fAlias := $.Aliases.Table .Foreign -}}
    {{$relAlias}} ModAs[Q, {{$.ColumnsType .Foreign}}]
    {{end -}}
  }

  func (j {{$tAlias.UpSingular}}Joins[Q]) AliasedAs(alias string) {{$tAlias.UpSingular}}Joins[Q] {
    return Build{{$tAlias.UpSingular}}Joins[Q](Build{{$tAlias.UpSingular}}Columns(alias), j.typ)
  }

  func Build{{$tAlias.UpSingular}}Joins[Q dialect.Joinable](cols {{$tAlias.UpSingular}}Columns, typ string) {{$tAlias.UpSingular}}Joins[Q] {
    return {{$tAlias.UpSingular}}Joins[Q]{
      typ: typ,
      {{range $rel := $.Relationships.Get $table.Key -}}
        {{- $fAlias := $.Aliases.Table $rel.Foreign -}}
        {{- $relAlias := $tAlias.Relationship $rel.Name -}}
        {{$relAlias}}: ModAs[Q, {{$.ColumnsType $rel.Foreign}}] {
          c: {{$.TableVar $rel.Foreign}}.Columns,
          f: func(to {{$.ColumnsType $rel.Foreign}}) bob.Mod[Q] {
            {{if gt (len $rel.Sides) 1 -}}{{$.Importer.Import "strconv" -}}
              uniqueSuffix := strconv.FormatUint(bob.NextUniqueInt(), 10)
            {{- end}}
            mods := make(mods.QueryMods[Q], 0, {{len $rel.Sides}})

            {{range $index, $side := $rel.Sides -}}
            {{- $from := $.Aliases.Table $side.From -}}
            {{- $fromCols := printf "%s.Columns" ($.TableVar $side.From) -}}
            {{- $to := $.Aliases.Table $side.To -}}
            {{- $toCols := printf "%s.Columns" ($.TableVar $side.To) -}}
            {{- $toTable := $.AllTables.Get $side.To -}}
            {
              {{if ne $index 0 -}}
              cols := {{$fromCols}}.AliasedAs({{$fromCols}}.Alias() + uniqueSuffix)
              {{end -}}
              {{if ne $index (sub (len $rel.Sides) 1) -}}
              to := {{$toCols}}.AliasedAs({{$toCols}}.Alias() + uniqueSuffix)
              {{end -}}
              mods = append(mods, dialect.Join[Q](typ, {{$.TableVar $side.To}}.NameExpr().As(to.Alias())).On(
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
