{{$.Importer.Import "context"}}
{{$.Importer.Import "models" $.ModelsPackage}}

type contextKey string
var (
    {{range $table := .Tables}}
    {{ $tAlias := $.Aliases.Table $table.Key -}}
    {{$tAlias.DownSingular}}Ctx = newContextual[*models.{{$tAlias.UpSingular}}]("{{$tAlias.DownSingular}}")
    {{- end}}
)

// Contextual is a convienience wrapper around context.WithValue and context.Value
type contextual[V any] struct {
  key contextKey
}

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

