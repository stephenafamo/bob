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

