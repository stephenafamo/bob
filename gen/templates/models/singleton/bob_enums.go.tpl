{{if .Enums}}
{{$.Importer.Import "fmt"}}
{{$.Importer.Import "database/sql/driver"}}
{{end}}

{{- range $enum := $.Enums}}
	{{$allvals := list }}

	// Enum values for {{$enum.Type}}
	const (
	{{range $val := $enum.Values -}}
		{{- $enumValue := enumVal $val -}}
		{{$enum.Type}}{{$enumValue}} {{$enum.Type}} = {{quote $val}}
		{{$allvals = append $allvals (printf "%s%s" $enum.Type $enumValue) -}}
	{{end -}}
	)

	func All{{$enum.Type}}() []{{$enum.Type}} {
		return []{{$enum.Type}}{
      {{join ",\n" $allvals}},
    }
	}

	type {{$enum.Type}} string

  func (e {{$enum.Type}}) String() string {
    return string(e)
  }

  func (e {{$enum.Type}}) Valid() bool {
    switch e {
    case {{join ",\n" $allvals}}:
      return true
    default:
      return false
    } 
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
    case []byte:
      *e = {{$enum.Type}}(x)
    case nil:
      return fmt.Errorf("cannot nil into {{$enum.Type}}")
    default:
      return fmt.Errorf("cannot scan type %T: %v", value, value)
    }

    if !e.Valid() {
      return fmt.Errorf("invalid {{$enum.Type}} value: %s", *e)
    }

    return nil
  }

{{end -}}

