{{- $.Importer.Import "models" $.ModelsPackage -}}

{{range $enum := $.ExtraInfo.Enums}}
	type {{$enum.Type}} = models.{{$enum.Type}}
{{end -}}
