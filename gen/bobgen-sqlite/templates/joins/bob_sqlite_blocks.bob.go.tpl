{{- define "helpers/join_variables"}}
var (
	SelectJoins = getJoins[dialect.SelectEditor]
	UpdateJoins = getJoins[dialect.UpdateEditor]
)
{{end -}}
