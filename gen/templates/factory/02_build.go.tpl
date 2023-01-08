{{$.Importer.Import "models" $.ModelsPackage}}
{{ $table := .Table}}
{{ $tAlias := .Aliases.Table .Table.Key -}}

// setModelRels creates and sets the relationships on *models.{{$tAlias.UpSingular}}
// according to the relationships in the template. Nothing is inserted into the db
func (t {{$tAlias.UpSingular}}Template) setModelRels(o *models.{{$tAlias.UpSingular}}) {
    {{- range $index, $rel := .Table.Relationships -}}
        {{- $relAlias := $tAlias.Relationship .Name -}}
        {{- $invRel := $table.GetRelationshipInverse $.Tables . -}}
        {{- $ftable := $.Aliases.Table $rel.Foreign -}}
        {{- $invAlias := "" -}}
    {{- if and (not $.NoBackReferencing) $invRel.Name -}}
            {{- $invAlias = $ftable.Relationship $invRel.Name}}
        {{- end -}}

        if t.r.{{$relAlias}} != nil {
            {{- if not .IsToMany}}
                rel := t.r.{{$relAlias}}.o.toModel()
                {{- if and (not $.NoBackReferencing) $invRel.Name}}
                    {{- if not $invRel.IsToMany}}
                        rel.R.{{$invAlias}} = o
                    {{- else}}
                        rel.R.{{$invAlias}} = append(rel.R.{{$invAlias}}, o)
                    {{- end}}
                {{- end}}
                {{setFactoryDeps $.Importer $.Tables $.Aliases . false}}
            {{- else -}}
                rel := models.{{$ftable.UpSingular}}Slice{}
                for _, r := range t.r.{{$relAlias}} {
                  related := r.o.toModels(r.number)
                  for _, rel := range related {
                    {{- setFactoryDeps $.Importer $.Tables $.Aliases . false}}
                    {{- if and (not $.NoBackReferencing) $invRel.Name}}
                        {{- if not $invRel.IsToMany}}
                            rel.R.{{$invAlias}} = o
                        {{- else}}
                            rel.R.{{$invAlias}} = append(rel.R.{{$invAlias}}, o)
                        {{- end}}
                    {{- end}}
                  }
                  rel = append(rel, related...)
                }
            {{- end}}
            o.R.{{$relAlias}} = rel
        }

    {{end -}}
}

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
	o.setModelRels(m)

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
