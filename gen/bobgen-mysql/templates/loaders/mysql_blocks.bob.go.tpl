{{- define "helpers/then_load_variables"}}
var (
  SelectThenLoad = getThenLoaders[*dialect.SelectQuery]()
  InsertThenLoad = getThenLoaders[*dialect.InsertQuery]()
  )
{{end -}}
