type Factory struct {
    {{range $table := .Tables}}
    {{ $tAlias := $.Aliases.Table $table.Name -}}
		base{{$tAlias.UpSingular}}Mods {{$tAlias.UpSingular}}Mods
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

{{$.Importer.Import "reflect"}}
func isZero(value interface{}) bool {
	val := reflect.Indirect(reflect.ValueOf(value))
	typ := val.Type()

	zero := reflect.Zero(typ)
	return reflect.DeepEqual(zero.Interface(), val.Interface())
}


type contextKey string

{{$.Importer.Import "context"}}
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

{{$.Importer.Import "fmt"}}
func stringifyVal(val ...interface{}) string {
  strVal := ""

  for _, v := range val {
      strVal += fmt.Sprintf("%v", v)
  }

  return strVal
}
