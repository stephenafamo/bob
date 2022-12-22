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
// NOTE: Objects are not inserted into the database. Use {{$tAlias.UpSingular}}Template.Insert
func (o {{$tAlias.UpSingular}}Template) Build() (*models.{{$tAlias.UpSingular}}) {
	m := o.toModel()
	o.setModelRelationships(m)

	return m
}

// Build returns an models.{{$tAlias.UpSingular}}Slice
// Related objects are also created and placed in the .R field
// NOTE: Objects are not inserted into the database. Use {{$tAlias.UpSingular}}TemplateSlice.Insert
func (o {{$tAlias.UpSingular}}TemplateSlice) Build() (models.{{$tAlias.UpSingular}}Slice) {
	m := make(models.{{$tAlias.UpSingular}}Slice, len(o))

	for i, o := range o {
	  m[i] = o.Build()
	}

	return m
}
