{{- define "helpers/then_load_variables"}}
var (
  SelectThenLoad = getThenLoaders[dialect.SelectEditor]()
  InsertThenLoad = getThenLoaders[dialect.InsertEditor]()
  )
{{end -}}
