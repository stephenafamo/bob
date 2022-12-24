type factory struct {
    {{range $table := .Tables}}
    {{ $tAlias := $.Aliases.Table $table.Name -}}
		base{{$tAlias.UpSingular}}Mods {{$tAlias.UpSingular}}ModSlice
    {{- end}}
}

func NewFactory() *factory {
  return &factory{
    {{range $table := .Tables -}}
    {{ $tAlias := $.Aliases.Table $table.Name -}}
    base{{$tAlias.UpSingular}}Mods: {{$tAlias.UpSingular}}ModSlice{
      {{range $column := $table.Columns -}}
        {{if .Default}}{{continue}}{{end -}}
        {{$.Importer.ImportList $column.Imports -}}
        {{$colAlias := $tAlias.Column $column.Name -}}
        {{$tAlias.UpSingular}}Mods.{{$colAlias}}Func(func() {{$column.Type}} {
          var zero {{$column.Type}}
          return zero
        }),
      {{- end}}
    },
    {{- end}}
  }
}

var defaultFactory = NewFactory()

{{range $table := .Tables}}{{if not $table.IsJoinTable -}}
{{ $tAlias := $.Aliases.Table $table.Name -}}
func ClearBase{{$tAlias.UpSingular}}Mods() {
    defaultFactory.ClearBase{{$tAlias.UpSingular}}Mods()
}

{{ $tAlias := $.Aliases.Table $table.Name -}}
func (f *factory) ClearBase{{$tAlias.UpSingular}}Mods() {
    f.base{{$tAlias.UpSingular}}Mods = nil
}

{{ $tAlias := $.Aliases.Table $table.Name -}}
func AddBase{{$tAlias.UpSingular}}Mods(mods ...{{$tAlias.UpSingular}}Mod) {
    defaultFactory.AddBase{{$tAlias.UpSingular}}Mod(mods...)
}

{{ $tAlias := $.Aliases.Table $table.Name -}}
func (f *factory) AddBase{{$tAlias.UpSingular}}Mod(mods ...{{$tAlias.UpSingular}}Mod) {
f.base{{$tAlias.UpSingular}}Mods = append(f.base{{$tAlias.UpSingular}}Mods, mods...)
}

{{end}}{{- end}}

{{$.Importer.Import "context"}}
{{$.Importer.Import "models" $.ModelsPackage}}
type contextKey string
var (
    {{range $table := .Tables}}
    {{ $tAlias := $.Aliases.Table $table.Name -}}
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

