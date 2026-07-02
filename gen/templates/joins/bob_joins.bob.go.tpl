{{$.Importer.Import "github.com/stephenafamo/bob"}}
{{$.Importer.Import "github.com/stephenafamo/bob/clause"}}
{{$.Importer.Import (printf "github.com/stephenafamo/bob/dialect/%s/dialect" $.Dialect)}}

{{block "helpers/join_variables" . -}}
var (
	SelectJoins = getJoins[*dialect.SelectQuery]()
	UpdateJoins = getJoins[*dialect.UpdateQuery]()
	DeleteJoins = getJoins[*dialect.DeleteQuery]()
)
{{- end}}

type JoinSet[Q interface{ AliasedAs(string) Q }] struct {
    InnerJoin Q
    LeftJoin Q
    RightJoin Q
}

func (j JoinSet[Q]) AliasedAs(alias string) JoinSet[Q] {
  return JoinSet[Q]{
    InnerJoin: j.InnerJoin.AliasedAs(alias),
    LeftJoin: j.LeftJoin.AliasedAs(alias),
    RightJoin: j.RightJoin.AliasedAs(alias),
  }
}

type joins[Q dialect.Joinable] struct {
		{{range $table := .Tables -}}{{if $.Relationships.Get $table.Key -}}
		{{$tAlias := $.Aliases.Table $table.Key -}}
		{{$tAlias.UpPlural}} JoinSet[{{$.JoinType $table.Key}}[Q]]
		{{end}}{{end}}
}

func BuildJoinSet[Q interface { AliasedAs(string) Q }, C any, F func(C, string) Q](c C, f F) JoinSet[Q] {
	return JoinSet[Q] {
	  InnerJoin: f(c, clause.InnerJoin),
	  LeftJoin: f(c, clause.LeftJoin),
	  RightJoin: f(c, clause.RightJoin),
	}
}

func getJoins[Q dialect.Joinable]() joins[Q] {
	return joins[Q]{
		{{range $table := .Tables -}}{{if $.Relationships.Get $table.Key -}}
		{{$tAlias := $.Aliases.Table $table.Key -}}
		{{$tAlias.UpPlural}}: BuildJoinSet[{{$.JoinType $table.Key}}[Q]]({{$.TableVar $table.Key}}.Columns, {{$.BuildJoinFunc $table.Key}}),
		{{end}}{{end}}
	}
}

type ModAs[Q any, C interface{ AliasedAs(string) C }] struct {
  c C
  f func(C) bob.Mod[Q]
}

func (m ModAs[Q, C]) Apply(q Q) {
  m.f(m.c).Apply(q)
}

func (m ModAs[Q, C]) AliasedAs(alias string) bob.Mod[Q] {
  m.c = m.c.AliasedAs(alias)
  return m
}

