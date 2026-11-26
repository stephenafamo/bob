{{$.Importer.Import "github.com/stephenafamo/bob/expr"}}
type columnName = expr.ColumnName

{{if $.Relationships -}}
{{$.Importer.Import "github.com/stephenafamo/bob"}}
{{$.Importer.Import "github.com/stephenafamo/bob/orm/genruntime"}}
{{$.Importer.Import (printf "github.com/stephenafamo/bob/dialect/%s" $.Dialect)}}
{{$.Importer.Import (printf "github.com/stephenafamo/bob/dialect/%s/sm" $.Dialect)}}
{{$.Importer.Import (printf "github.com/stephenafamo/bob/dialect/%s/dialect" $.Dialect)}}

func correlatedQuery(rel {{$.Dialect}}.PreloadRel, parent string, projection bob.Expression, filters []bob.Mod[dialect.SelectEditor]) bob.QueryBuilder[dialect.SelectEditor] {
	q := {{$.Dialect}}.Select(sm.Columns(projection), genruntime.Correlate(rel, parent, func(q dialect.SelectEditor, table bob.Expression) { sm.From(table).Edit(q) }))
	return q.WithMods(filters...)
}
{{end -}}
