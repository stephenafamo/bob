{{$.Importer.Import "strings"}}
{{$.Importer.Import "github.com/jaswdr/faker"}}
{{$.Importer.Import "github.com/aarondl/opt/null"}}

var defaultFaker = faker.New()

// Random returns a random value for the given type, using the faker
// * If the given faker is nil, the default faker is used
// * The zero value is returned if the type cannot be handled
func Random[T any](f *faker.Faker) T {
    if f == nil {
      f = &defaultFaker
    }

    var val T
    switch any(val).(type) {
    default:
      return val
    case string:
      return any(string(strings.Join(f.Lorem().Words(5), " "))).(T)

    case bool:
      return any(bool(f.BoolWithChance(50))).(T)

    case int:
      return any(int(f.Int())).(T)

    {{$doneTypes := dict "string" nil "bool" nil "int" nil }}
    {{- range $table := .Tables}}
    {{- $tAlias := $.Aliases.Table $table.Key}}
      {{range $column := $table.Columns -}}
        {{- $colTyp := $column.Type -}}
        {{- if hasKey $doneTypes $colTyp}}{{continue}}{{end -}}
        {{- $.Importer.ImportList $column.Imports -}}
        {{- $_ :=  set $doneTypes $colTyp nil -}}
        {{- $colAlias := $tAlias.Column $column.Name -}}
        case {{$colTyp}}:
          return val

      {{end -}}
    {{- end}}
    }
}

// RandomNull is like [Random], but for null types
func RandomNull[T any](f *faker.Faker) null.Val[T] {
  return null.From(Random[T](f))
}
