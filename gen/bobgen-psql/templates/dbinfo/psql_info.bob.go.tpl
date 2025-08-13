{{- define "index_extra_fields" -}}
		NullsFirst    []bool   
		NullsDistinct bool     
		Where         string   
		Include       []string 
{{- end -}}
