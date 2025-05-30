{{- if $.Enums}}
{{$.Importer.Import "models" (index $.OutputPackages "models") }}

type (
	{{range $enum := $.Enums -}}
		{{$enum.Type}} = models.{{$enum.Type}}
	{{end}}
)

{{range $enum := $.Enums -}}
	func all{{$enum.Type}}() []{{$enum.Type}} {
		return models.All{{$enum.Type}}()
	}
{{end}}

{{end -}}
