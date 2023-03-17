{{if $.Enums}}
{{$.Importer.Import "models" $.ModelsPackage}}

type (
	{{range $enum := $.Enums -}}
		{{$enum.Type}} = models.{{$enum.Type}}
	{{end}}
)
{{end}}
