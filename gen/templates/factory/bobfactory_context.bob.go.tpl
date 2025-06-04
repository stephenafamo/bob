{{$.Importer.Import "context"}}

type contextKey string
var (
    {{range $table := .Tables -}}
      {{ $tAlias := $.Aliases.Table $table.Key -}}
      // Relationship Contexts for {{$table.Key}}
      {{$tAlias.DownSingular}}WithParentsCascadingCtx = newContextual[bool]("{{$tAlias.DownSingular}}WithParentsCascading")
      {{range $rel := $.Relationships.Get $table.Key -}}
        {{ $relAlias := $tAlias.Relationship .Name -}}
        {{$tAlias.DownSingular}}Rel{{$relAlias}}Ctx = newContextual[bool]("{{$.Relationships.GlobalKey $rel}}");
      {{- end}}

    {{end}}
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

