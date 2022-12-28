{{$tAlias := .Aliases.Table .Table.Name -}}

func New{{$tAlias.UpSingular}}(mods ...{{$tAlias.UpSingular}}Mod) *{{$tAlias.UpSingular}}Template {
	return defaultFactory.New{{$tAlias.UpSingular}}(mods...)
}

func (f *factory) New{{$tAlias.UpSingular}}(mods ...{{$tAlias.UpSingular}}Mod) *{{$tAlias.UpSingular}}Template {
	o := &{{$tAlias.UpSingular}}Template{f: f}

	f.base{{$tAlias.UpSingular}}Mods.Apply(o)
 {{$tAlias.UpSingular}}ModSlice(mods).Apply(o)

	return o
}

