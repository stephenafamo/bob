{{block "join_helpers" . -}}
var (
	SelectJoins = joins[*dialect.SelectQuery]
	UpdateJoins = joins[*dialect.UpdateQuery]
)
{{- end}}

