{{- define "helpers/where_variables"}}
{{$.Importer.Import (printf "github.com/stephenafamo/bob/dialect/%s/dialect" $.Dialect)}}
var (
	SelectWhere = Where[*dialect.SelectQuery]()
	UpdateWhere = Where[*dialect.UpdateQuery]()
	DeleteWhere = Where[*dialect.DeleteQuery]()
)
{{- end -}}

{{- define "helpers/then_load_variables"}}
var (
	SelectThenLoad = getThenLoaders[*dialect.SelectQuery]()
)
{{end -}}
