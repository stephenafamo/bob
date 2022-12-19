{{range $enum := $.ExtraInfo.Enums}}
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
{{end}}


{{if $.ExtraInfo.Enums}}
{{$.Importer.Import "bytes"}}
{{$.Importer.Import "github.com/lib/pq"}}
{{$.Importer.Import "database/sql/driver"}}
type EnumArray[T ~string] []T

// Scan implements the sql.Scanner interface.
func (e *EnumArray[T]) Scan(src any) error {
	var arr pq.StringArray
	if err := arr.Scan(src); err != nil {
		return err
	}

	var slice = make([]T, len(arr))
	for i, s := range arr {
		slice[i] = T(s)
	}

	*e = slice
	return nil
}

// Value implements the driver.Valuer interface.
func (e EnumArray[T]) Value() (driver.Value, error) {
	if e == nil {
		return nil, nil
	}

	if n := len(e); n > 0 {
		// There will be at least two curly brackets, 2*N bytes of quotes,
		// and N-1 bytes of delimiters.
		b := make([]byte, 1, 1+3*n)
		b[0] = '{'

		b = appendArrayQuotedBytes(b, []byte(e[0]))
		for i := 1; i < n; i++ {
			b = append(b, ',')
			b = appendArrayQuotedBytes(b, []byte(e[i]))
		}

		return string(append(b, '}')), nil
	}

	return "{}", nil
}

func appendArrayQuotedBytes(b, v []byte) []byte {
	b = append(b, '"')
	for {
		i := bytes.IndexAny(v, `"\`)
		if i < 0 {
			b = append(b, v...)
			break
		}
		if i > 0 {
			b = append(b, v[:i]...)
		}
		b = append(b, '\\', v[i])
		v = v[i+1:]
	}
	return append(b, '"')
}
{{end}}
