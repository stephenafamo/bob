{{$.Importer.Import "models" $.ModelsPackage}}
{{$table := .Table}}
{{$tAlias := .Aliases.Table .Table.Name -}}

type {{$tAlias.UpSingular}}Mod interface {
	Apply(*{{$tAlias.UpSingular}}Template) error
}

type {{$tAlias.UpSingular}}ModFunc func(*{{$tAlias.UpSingular}}Template) error

func (f {{$tAlias.UpSingular}}ModFunc) Apply(n *{{$tAlias.UpSingular}}Template) error {
	return f(n)
}

type {{$tAlias.UpSingular}}Mods []{{$tAlias.UpSingular}}Mod

func (mods {{$tAlias.UpSingular}}Mods) Apply(n *{{$tAlias.UpSingular}}Template) error {
	for _, f := range mods {
		err := f.Apply(n)
		if err != nil {
			return err
		}
	}

	return nil
}

// {{$tAlias.UpSingular}}TemplateSlice is an alias for a slice of pointers to {{$tAlias.UpSingular}}.
// This should almost always be used instead of []{{$tAlias.UpSingular}}Template.
type {{$tAlias.UpSingular}}TemplateSlice []*{{$tAlias.UpSingular}}Template

// {{$tAlias.UpSingular}}Template is an object representing the database table.
// all columns are optional and should be set by mods
type {{$tAlias.UpSingular}}Template struct {
	{{- range $column := .Table.Columns -}}
		{{- $.Importer.ImportList $column.Imports -}}
		{{- $colAlias := $tAlias.Column $column.Name -}}
		{{- $colTyp := "" -}}
		{{- if $column.Nullable -}}
			{{- $.Importer.Import "github.com/aarondl/opt/omitnull" -}}
			{{- $colTyp = printf "omitnull.Val[%s]" $column.Type -}}
		{{- else -}}
			{{- $.Importer.Import "github.com/aarondl/opt/omit" -}}
			{{- $colTyp = printf "omit.Val[%s]" $column.Type -}}
		{{- end -}}
		{{$colAlias}} {{$colTyp}}
	{{end -}}

	{{if .Table.Relationships}}
		r {{$tAlias.DownSingular}}R
	{{end -}}
}

{{if .Table.Relationships}}
{{$.Importer.Import "github.com/aarondl/opt/omit" -}}
type {{$tAlias.DownSingular}}R struct {
	{{range .Table.Relationships -}}
	{{- $ftable := $.Aliases.Table .Foreign -}}
	{{- $relAlias := $tAlias.Relationship .Name -}}
	{{if .IsToMany -}}
		{{$relAlias}} omit.Val[{{$ftable.UpSingular}}TemplateSlice]
	{{else -}}
		{{$relAlias}} omit.Val[*{{$ftable.UpSingular}}Template]
	{{end}}{{end -}}
}
{{end -}}

func (o *{{$tAlias.UpSingular}}Template) Apply(mods ...{{$tAlias.UpSingular}}Mod) error {
  for _, mod := range mods {
		if err := mod.Apply(o); err != nil {
			return err
		}
	}

	return nil
}

func (o {{$tAlias.UpSingular}}Template) ToModel() (*models.{{$tAlias.UpSingular}}) {
	m := &models.{{$tAlias.UpSingular}}{}

	{{range $column := .Table.Columns -}}
	{{$colAlias := $tAlias.Column $column.Name -}}
		if !o.{{$colAlias}}.IsUnset() {
			{{if $column.Nullable -}}
			m.{{$colAlias}} = o.{{$colAlias}}.MustGetNull()
			{{else -}}
			m.{{$colAlias}} = o.{{$colAlias}}.MustGet()
			{{end -}}
		}
	{{end}}

	return m
}

{{if .Table.PKey -}}
func (o {{$tAlias.UpSingular}}Template) ToOptional() (*models.Optional{{$tAlias.UpSingular}}) {
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

type {{$tAlias.UpSingular}} struct {}
