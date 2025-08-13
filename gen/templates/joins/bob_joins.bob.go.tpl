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

type joinSet[Q interface{ aliasedAs(string) Q }] struct {
    InnerJoin Q
    LeftJoin Q
    RightJoin Q
}

func (j joinSet[Q]) AliasedAs(alias string) joinSet[Q] {
  return joinSet[Q]{
    InnerJoin: j.InnerJoin.aliasedAs(alias),
    LeftJoin: j.LeftJoin.aliasedAs(alias),
    RightJoin: j.RightJoin.aliasedAs(alias),
  }
}

type joins[Q dialect.Joinable] struct {
		{{range $table := .Tables -}}{{if $.Relationships.Get $table.Key -}}
		{{$tAlias := $.Aliases.Table $table.Key -}}
		{{$tAlias.UpPlural}} joinSet[{{$tAlias.DownSingular}}Joins[Q]]
		{{end}}{{end}}
}

func buildJoinSet[Q interface { aliasedAs(string) Q }, C any, F func(C, string) Q](c C, f F) joinSet[Q] {
	return joinSet[Q] {
	  InnerJoin: f(c, clause.InnerJoin),
	  LeftJoin: f(c, clause.LeftJoin),
	  RightJoin: f(c, clause.RightJoin),
	}
}

func getJoins[Q dialect.Joinable]() joins[Q] {
	return joins[Q]{
		{{range $table := .Tables -}}{{if $.Relationships.Get $table.Key -}}
		{{$tAlias := $.Aliases.Table $table.Key -}}
		{{$tAlias.UpPlural}}: buildJoinSet[{{$tAlias.DownSingular}}Joins[Q]]({{$tAlias.UpPlural}}.Columns, build{{$tAlias.UpSingular}}Joins),
		{{end}}{{end}}
	}
}

type modAs[Q any, C interface{ AliasedAs(string) C }] struct {
  c C
  f func(C) bob.Mod[Q]
}

func (m modAs[Q, C]) Apply(q Q) {
  m.f(m.c).Apply(q)
}

func (m modAs[Q, C]) AliasedAs(alias string) bob.Mod[Q] {
  m.c = m.c.AliasedAs(alias)
  return m
}

{{$.Importer.Import "hash/maphash"}}
func randInt() int64 {
	out := int64(new(maphash.Hash).Sum64())

	if out < 0 {
		return -out % 10000
	}

	return out % 10000
}
