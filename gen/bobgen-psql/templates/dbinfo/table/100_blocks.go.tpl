{{- define "index_extra_values" -}}
		NullsFirst:    {{printf "%#v" .NullsFirst}},
		NullsDistinct: {{.NullsDistinct}},
    Where:         {{quote .Where}},
		Include:       {{printf "%#v" .Include}},
{{- end -}}
