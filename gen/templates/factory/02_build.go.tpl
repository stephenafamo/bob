{{$.Importer.Import "models" $.ModelsPackage}}
{{ $tAlias := .Aliases.Table .Table.Name -}}

{{if .Table.PKey -}}
// BuildOptional returns an *models.Optional{{$tAlias.UpSingular}}
// this does nothing with the relationship templates
func (o {{$tAlias.UpSingular}}Template) BuildOptional() *models.Optional{{$tAlias.UpSingular}} {
	m := &models.Optional{{$tAlias.UpSingular}}{}

	{{range $column := .Table.Columns -}}
	{{- if $column.Generated}}{{continue}}{{end -}}
	{{$colAlias := $tAlias.Column $column.Name -}}
		if o.{{$colAlias}} != nil {
			{{if $column.Nullable -}}
			{{- $.Importer.Import "github.com/aarondl/opt/omitnull" -}}
			m.{{$colAlias}} = omitnull.FromNull(o.{{$colAlias}}())
			{{else -}}
			{{- $.Importer.Import "github.com/aarondl/opt/omit" -}}
			m.{{$colAlias}} = omit.From(o.{{$colAlias}}())
			{{end -}}
		}
	{{end}}

	return m
}

// BuildManyOptional returns an []*models.Optional{{$tAlias.UpSingular}}
// this does nothing with the relationship templates
func (o {{$tAlias.UpSingular}}Template) BuildManyOptional(number int) []*models.Optional{{$tAlias.UpSingular}} {
	m := make([]*models.Optional{{$tAlias.UpSingular}}, number)

	for i := range m {
	  m[i] = o.BuildOptional()
	}

	return m
}
{{- end}}

// Build returns an *models.{{$tAlias.UpSingular}}
// Related objects are also created and placed in the .R field
// NOTE: Objects are not inserted into the database. Use {{$tAlias.UpSingular}}Template.Create
func (o {{$tAlias.UpSingular}}Template) Build() *models.{{$tAlias.UpSingular}} {
	m := o.toModel()
	o.setModelRelationships(m)

	return m
}

// BuildMany returns an models.{{$tAlias.UpSingular}}Slice
// Related objects are also created and placed in the .R field
// NOTE: Objects are not inserted into the database. Use {{$tAlias.UpSingular}}Template.CreateMany
func (o {{$tAlias.UpSingular}}Template) BuildMany(number int) models.{{$tAlias.UpSingular}}Slice {
	m := make(models.{{$tAlias.UpSingular}}Slice, number)

	for i := range m {
	  m[i] = o.Build()
	}

	return m
}
