{{$tAlias := .Aliases.Table .Table.Name -}}

func New{{$tAlias.UpSingular}}(mods ...{{$tAlias.UpSingular}}Mod) *{{$tAlias.UpSingular}}Template {
	return defaultFactory.New{{$tAlias.UpSingular}}(mods...)
}

func (f Factory) New{{$tAlias.UpSingular}}(mods ...{{$tAlias.UpSingular}}Mod) *{{$tAlias.UpSingular}}Template {
	o := &{{$tAlias.UpSingular}}Template{}

	f.base{{$tAlias.UpSingular}}Mods.Apply(o)
 {{$tAlias.UpSingular}}Mods(mods).Apply(o)

	return o
}

func New{{$tAlias.UpPlural}}(number int, mods ...{{$tAlias.UpSingular}}Mod) {{$tAlias.UpSingular}}TemplateSlice {
	return defaultFactory.New{{$tAlias.UpPlural}}(number, mods...)
}

func (f Factory) New{{$tAlias.UpPlural}}(number int, mods ...{{$tAlias.UpSingular}}Mod) {{$tAlias.UpSingular}}TemplateSlice {
  var templates = make({{$tAlias.UpSingular}}TemplateSlice, number)

  for i := 0; i < number; i++ {
		templates[i] = f.New{{$tAlias.UpSingular}}(mods...)
	}

	return templates
}

