{{- define "helpers/where_variables"}}
{{$.Importer.Import (printf "github.com/stephenafamo/bob/dialect/%s/dialect" $.Dialect)}}
var (
	SelectWhere = Where[dialect.SelectEditor]()
	UpdateWhere = Where[dialect.UpdateEditor]()
	DeleteWhere = Where[dialect.DeleteEditor]()
)
{{- end -}}
