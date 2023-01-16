{{- range $enum := $.Enums}}
	{{$allvals := "\n"}}
	type {{$enum.Type}} string

	// Enum values for {{$enum.Type}}
	const (
	{{range $val := $enum.Values -}}
		{{- $enumValue := titleCase $val -}}
		{{$enum.Type}}{{$enumValue}} {{$enum.Type}} = {{quote $val}}
		{{$allvals = printf "%s%s%s,\n" $allvals $enum.Type $enumValue -}}
	{{end -}}
	)

	func All{{$enum.Type}}() []{{$enum.Type}} {
		return []{{$enum.Type}}{ {{$allvals}} }
	}

{{end -}}

