{{- define "helpers/then_load_variables"}}
var (
  SelectThenLoad = getThenLoaders[*dialect.SelectQuery]()
  )
{{end -}}
