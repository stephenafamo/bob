{{if .Enums}}
{{$.Importer.Import "fmt"}}
{{$.Importer.Import "database/sql/driver"}}
{{end}}

{{- range $enum := $.Enums}}
	{{$allvals := "\n"}}

	// Enum values for {{$enum.Type}}
	const (
	{{range $val := $enum.Values -}}
		{{- $enumValue := enumVal $val -}}
		{{$enum.Type}}{{$enumValue}} {{$enum.Type}} = {{quote $val}}
		{{$allvals = printf "%s%s%s,\n" $allvals $enum.Type $enumValue -}}
	{{end -}}
	)

	func All{{$enum.Type}}() []{{$enum.Type}} {
		return []{{$enum.Type}}{ {{$allvals}} }
	}

	type {{$enum.Type}} string

  func (e {{$enum.Type}}) String() string {
    return string(e)
  }

  func (e {{$enum.Type}}) MarshalText() ([]byte, error) {
    return []byte(e), nil
  }

  func (e *{{$enum.Type}}) UnmarshalText(text []byte) error {
    return e.Scan(text)
  }

  func (e {{$enum.Type}}) MarshalBinary() ([]byte, error) {
    return []byte(e), nil
  }

  func (e *{{$enum.Type}}) UnmarshalBinary(data []byte) error {
    return e.Scan(data)
  }

  func (e {{$enum.Type}}) Value() (driver.Value, error) {
    return string(e), nil
  }

  func (e *{{$enum.Type}}) Scan(value any) error {
    switch x := value.(type) {
    case string:
      *e = {{$enum.Type}}(x)
      return nil
    case []byte:
      *e = {{$enum.Type}}(x)
      return nil
    case nil:
      return nil
    default:
      return fmt.Errorf("cannot scan type %T: %v", value, value)
    }
  }

{{end -}}

