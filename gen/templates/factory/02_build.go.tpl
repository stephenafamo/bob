{{$.Importer.Import "models" $.ModelsPackage}}
{{ $tAlias := .Aliases.Table .Table.Name -}}


{{if .Table.PKey -}}
// BuildOptional returns an *models.Optional{{$tAlias.UpSingular}}
// this does nothing with the relationship templates
func (o {{$tAlias.UpSingular}}Template) BuildOptional() (*models.Optional{{$tAlias.UpSingular}}) {
	m := &models.Optional{{$tAlias.UpSingular}}{}

	{{range $column := .Table.Columns -}}
	{{- if $column.Generated}}{{continue}}{{end -}}
	{{$colAlias := $tAlias.Column $column.Name -}}
		if !o.{{$colAlias}}.IsUnset() {
			{{if $column.Nullable -}}
			m.{{$colAlias}} = o.{{$colAlias}}
			{{else -}}
			m.{{$colAlias}} = o.{{$colAlias}}
			{{end -}}
		}
	{{end}}

	return m
}
{{- end}}


// Build returns an *models.{{$tAlias.UpSingular}}
// Related objects are also created and placed in the .R field
// NOTE: Objects are not inserted into the database. Use {{$tAlias.UpSingular}}.Insert
func (o {{$tAlias.UpSingular}}Template) Build() (*models.{{$tAlias.UpSingular}}) {
	m := o.toModel()
	o.setModelRelationships(m)

	return m
}


func Build{{$tAlias.UpSingular}}(mods ...{{$tAlias.UpSingular}}Mod) (*models.{{$tAlias.UpSingular}}, error) {
	return defaultFactory.Build{{$tAlias.UpSingular}}(mods...)
}

func (f Factory) Build{{$tAlias.UpSingular}}(mods ...{{$tAlias.UpSingular}}Mod) (*models.{{$tAlias.UpSingular}}, error) {
	o, err := f.Get{{$tAlias.UpSingular}}Template(mods...)
	if err != nil {
	  return nil, err
	}

	return o.Build(), err
}

func Build{{$tAlias.UpPlural}}(number int, mods ...{{$tAlias.UpSingular}}Mod) (models.{{$tAlias.UpSingular}}Slice, error) {
	return defaultFactory.Build{{$tAlias.UpPlural}}(number, mods...)
}

func (f Factory) Build{{$tAlias.UpPlural}}(number int, mods ...{{$tAlias.UpSingular}}Mod) (models.{{$tAlias.UpSingular}}Slice, error) {
	var err error
  var built = make(models.{{$tAlias.UpSingular}}Slice, number)

  for i := 0; i < number; i++ {
		built[i], err = f.Build{{$tAlias.UpSingular}}(mods...)
		if err != nil {
			return nil, err
		}
	}

	return built, nil
}

