{{if .Enums}}
{{$.Importer.Import "fmt"}}
{{$.Importer.Import "database/sql/driver"}}
{{end}}

{{- range $enum := $.Enums}}
	{{$allvals := list }}

	// Enum values for {{$enum.Type}}
	const (
	{{range $i, $val := $enum.Values -}}
		{{- $enumValue := enumVal $val -}}
		{{$enum.Type}}{{$enumValue}} {{$enum.Type}} = {{ $i }}
		{{$allvals = append $allvals (printf "%s%s" $enum.Type $enumValue) -}}
	{{end -}}
	)

	func All{{$enum.Type}}() []{{$enum.Type}} {
		return []{{$enum.Type}}{
      {{join ",\n" $allvals}},
    }
	}

	type {{$enum.Type}} int32

  func (e {{$enum.Type}}) String() string {
		switch e {
		{{range $val := $enum.Values -}}
		{{- $enumValue := enumVal $val -}}
		case {{$enum.Type}}{{$enumValue}}:
			return {{printf "%q" $val}}
		{{end}}
		default:
      panic(fmt.Errorf("enum value %d invalid for {{printf "%s" $enum.Type}}", e))
		}
  }

  func Parse{{$enum.Type}}(s string) ({{$enum.Type}}, error) {
  	switch s {
  	{{range $val := $enum.Values -}}
    {{- $enumValue := enumVal $val -}}
	  case {{printf "%q" $val}}:
		  return {{$enum.Type}}{{$enumValue}}, nil
	  {{end}}
  	default:
  	    return {{$enum.Type}}(0), fmt.Errorf("unable to parse %s for {{printf "%s" $enum.Type}}", s)
  	}
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
    if !e.Valid() {
			return []byte{}, nil
		}
		return []byte(e.String()), nil
  }

  func (e *{{$enum.Type}}) UnmarshalText(text []byte) error {
    return e.Scan(text)
  }

  func (e {{$enum.Type}}) MarshalBinary() ([]byte, error) {
    return []byte(e.String()), nil
  }

  func (e *{{$enum.Type}}) UnmarshalBinary(data []byte) error {
    return e.Scan(data)
  }

  func (e {{$enum.Type}}) Value() (driver.Value, error) {
    return e.String(), nil
  }

  func (e *{{$enum.Type}}) Scan(value any) error {
    switch x := value.(type) {
    case string:
      ee, err := Parse{{$enum.Type}}(x)
      if err != nil {
        return err
      }
      *e = ee
    case []byte:
      ee, err := Parse{{$enum.Type}}(string(x))
      if err != nil {
        return err
      }
      *e = ee
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

