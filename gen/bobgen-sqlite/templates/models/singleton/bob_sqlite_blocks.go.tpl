{{define "helpers/join_variables" -}}
var (
	SelectJoins = getJoins[*dialect.SelectQuery]
	UpdateJoins = getJoins[*dialect.UpdateQuery]
)
{{- end}}

