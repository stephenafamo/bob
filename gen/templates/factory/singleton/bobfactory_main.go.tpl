{{$.Importer.Import "strings"}}
{{$.Importer.Import "github.com/jaswdr/faker"}}
{{$.Importer.Import "github.com/aarondl/opt/null"}}

type Factory struct {
    {{range $table := .Tables}}
    {{ $tAlias := $.Aliases.Table $table.Key -}}
		base{{$tAlias.UpSingular}}Mods {{$tAlias.UpSingular}}ModSlice
    {{- end}}
}

func New() *Factory {
  return &Factory{}
}

{{range $table := .Tables}}
{{ $tAlias := $.Aliases.Table $table.Key -}}
func (f *Factory) New{{$tAlias.UpSingular}}(mods ...{{$tAlias.UpSingular}}Mod) *{{$tAlias.UpSingular}}Template {
	o := &{{$tAlias.UpSingular}}Template{f: f}

  if f != nil {
    f.base{{$tAlias.UpSingular}}Mods.Apply(o)
  }

  {{$tAlias.UpSingular}}ModSlice(mods).Apply(o)

	return o
}

{{end}}

{{range $table := .Tables}}
{{ $tAlias := $.Aliases.Table $table.Key -}}
func (f *Factory) ClearBase{{$tAlias.UpSingular}}Mods() {
    f.base{{$tAlias.UpSingular}}Mods = nil
}

func (f *Factory) AddBase{{$tAlias.UpSingular}}Mod(mods ...{{$tAlias.UpSingular}}Mod) {
f.base{{$tAlias.UpSingular}}Mods = append(f.base{{$tAlias.UpSingular}}Mods, mods...)
}

{{end}}

{{$.Importer.Import "context"}}
{{$.Importer.Import "models" $.ModelsPackage}}
type contextKey string
var (
    {{range $table := .Tables}}
    {{ $tAlias := $.Aliases.Table $table.Key -}}
    {{$tAlias.DownSingular}}Ctx = newContextual[*models.{{$tAlias.UpSingular}}]("{{$tAlias.DownSingular}}")
    {{- end}}
)

type contextual[V any] struct {
  key contextKey
}

// This could be weird because of type inference not handling `K` due to `V` having to be manual.
func newContextual[V any](key string) contextual[V] {
  return contextual[V]{key: contextKey(key)}
}

func (k contextual[V]) WithValue(ctx context.Context, val V) context.Context {
  return context.WithValue(ctx, k.key, val)
}

func (k contextual[V]) Value(ctx context.Context) (V, bool) {
  v, ok := ctx.Value(k.key).(V)
  return v, ok
}

var defaultFaker = faker.New()

// random returns a random value for the given type, using the faker
// * If the given faker is nil, the default faker is used
// * The zero value is returned if the type cannot be handled
func random[T any](f *faker.Faker) T {
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

// randomNull is like [Random], but for null types
func randomNull[T any](f *faker.Faker) null.Val[T] {
  return null.From(random[T](f))
}
