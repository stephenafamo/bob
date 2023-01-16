{{- $.Importer.Import "models" $.ModelsPackage -}}

{{range $enum := $.Enums}}
	type {{$enum.Type}} = models.{{$enum.Type}}
{{end -}}
