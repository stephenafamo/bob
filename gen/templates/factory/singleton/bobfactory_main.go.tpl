type Factory struct {
    {{range $table := .Tables}}
    {{ $tAlias := $.Aliases.Table $table.Name -}}
		base{{$tAlias.UpSingular}}Mods {{$tAlias.UpSingular}}ModSlice
    {{- end}}
}

var defaultFactory = new(Factory)

{{range $table := .Tables}}{{if not $table.IsJoinTable -}}
{{ $tAlias := $.Aliases.Table $table.Name -}}
func ClearBase{{$tAlias.UpSingular}}Mods() {
    defaultFactory.ClearBase{{$tAlias.UpSingular}}Mods()
}

{{ $tAlias := $.Aliases.Table $table.Name -}}
func (f *Factory) ClearBase{{$tAlias.UpSingular}}Mods() {
    f.base{{$tAlias.UpSingular}}Mods = nil
}

{{ $tAlias := $.Aliases.Table $table.Name -}}
func AddBase{{$tAlias.UpSingular}}Mods(mods ...{{$tAlias.UpSingular}}Mod) {
    defaultFactory.AddBase{{$tAlias.UpSingular}}Mod(mods...)
}

{{ $tAlias := $.Aliases.Table $table.Name -}}
func (f *Factory) AddBase{{$tAlias.UpSingular}}Mod(mods ...{{$tAlias.UpSingular}}Mod) {
f.base{{$tAlias.UpSingular}}Mods = append(f.base{{$tAlias.UpSingular}}Mods, mods...)
}

{{end}}{{- end}}

{{$.Importer.Import "context"}}
{{$.Importer.Import "models" $.ModelsPackage}}
type contextKey string
var (
    {{range $table := .Tables}}
    {{ $tAlias := $.Aliases.Table $table.Name -}}
    {{$tAlias.UpSingular}}Ctx = NewKey[contextKey, *models.{{$tAlias.UpSingular}}]("{{$tAlias.DownSingular}}")
    {{- end}}
)

type Key[K comparable, V any] struct {
  key K
}

// This could be weird because of type inference not handling `K` due to `V` having to be manual.
func NewKey[K comparable, V any](key K) Key[K, V] {
  return Key[K, V]{key: key}
}

func (k Key[K, V]) WithValue(ctx context.Context, val V) context.Context {
  return context.WithValue(ctx, k.key, val)
}

func (k Key[K, V]) Value(ctx context.Context) (V, bool) {
  v, ok := ctx.Value(k.key).(V)
  return v, ok
}

func inContextKey(ctx context.Context, key contextKey, val string) bool {
  vals, _ := ctx.Value(key).(map[string]struct{})
  if vals == nil {
      return false
  }

  _, ok := vals[val]
  return ok
}

func addToContextKey(ctx context.Context, key contextKey, val string) context.Context {
  vals, _ := ctx.Value(key).(map[string]struct{})
  if vals == nil {
      vals = map[string]struct{}{
          val: {},
      }
  } else {
      vals[val] = struct{}{}
  }

  return context.WithValue(ctx, key, vals)
}

