{{$.Importer.Import "models" $.ModelsPackage}}
{{ $tAlias := .Aliases.Table .Table.Name -}}

func (f Factory) Get{{$tAlias.UpSingular}}Template(mods ...{{$tAlias.UpSingular}}Mod) (*{{$tAlias.UpSingular}}Template, error) {
	o := &{{$tAlias.UpSingular}}Template{}

	if err := f.base{{$tAlias.UpSingular}}Mods.Apply(o); err != nil {
		return nil, err
	}

	if err := {{$tAlias.UpSingular}}Mods(mods).Apply(o); err != nil {
	  return nil, err
	}

	return o, nil
}

func (f Factory) Get{{$tAlias.UpSingular}}TemplateSlice(length int, mods ...{{$tAlias.UpSingular}}Mod) ({{$tAlias.UpSingular}}TemplateSlice, error) {
	var err error
  var templates = make({{$tAlias.UpSingular}}TemplateSlice, length)

  for i := 0; i < length; i++ {
		templates[i], err = f.Get{{$tAlias.UpSingular}}Template(mods...)
		if err != nil {
			return nil, err
		}
	}

	return templates, nil
}

func Create{{$tAlias.UpSingular}}(mods ...{{$tAlias.UpSingular}}Mod) (*models.{{$tAlias.UpSingular}}, error) {
	return defaultFactory.Create{{$tAlias.UpSingular}}(mods...)
}

func (f Factory) Create{{$tAlias.UpSingular}}(mods ...{{$tAlias.UpSingular}}Mod) (*models.{{$tAlias.UpSingular}}, error) {
	o, err := f.Get{{$tAlias.UpSingular}}Template(mods...)
	if err != nil {
	  return nil, err
	}

	return o.ToModel(), err
}

func Create{{$tAlias.UpPlural}}(number int, mods ...{{$tAlias.UpSingular}}Mod) (models.{{$tAlias.UpSingular}}Slice, error) {
	return defaultFactory.Create{{$tAlias.UpPlural}}(number, mods...)
}

func (f Factory) Create{{$tAlias.UpPlural}}(number int, mods ...{{$tAlias.UpSingular}}Mod) (models.{{$tAlias.UpSingular}}Slice, error) {
	var err error
  var created = make(models.{{$tAlias.UpSingular}}Slice, number)

  for i := 0; i < number; i++ {
		created[i], err = f.Create{{$tAlias.UpSingular}}(mods...)
		if err != nil {
			return nil, err
		}
	}

	return created, nil
}

