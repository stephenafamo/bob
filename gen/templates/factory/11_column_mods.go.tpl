{{$.Importer.Import "github.com/jaswdr/faker/v2"}}
{{ $table := .Table }}
{{ $tAlias := .Aliases.Table .Table.Key -}}

// {{$tAlias.UpSingular}} has methods that act as mods for the {{$tAlias.UpSingular}}Template
var {{$tAlias.UpSingular}}Mods {{$tAlias.DownSingular}}Mods
type {{$tAlias.DownSingular}}Mods struct {}

func (m {{$tAlias.DownSingular}}Mods) RandomizeAllColumns(f *faker.Faker) {{$tAlias.UpSingular}}Mod {
	return {{$tAlias.UpSingular}}ModSlice{
		{{range $column := $table.Columns -}}
			{{$colAlias := $tAlias.Column $column.Name -}}
			{{$tAlias.UpSingular}}Mods.Random{{$colAlias}}(f),
		{{end -}}
	}
}

{{range $column := .Table.Columns}}
{{$colAlias := $tAlias.Column $column.Name -}}
{{- $typDef :=  index $.Types $column.Type -}}
{{- $colTyp := getType $column.Type $typDef -}}
{{- if $column.Nullable -}}
	{{- $.Importer.Import "github.com/aarondl/opt/null" -}}
	{{- $colTyp = printf "null.Val[%s]" $colTyp -}}
{{- end -}}

// Set the model columns to this value
func (m {{$tAlias.DownSingular}}Mods) {{$colAlias}}(val {{$colTyp}}) {{$tAlias.UpSingular}}Mod {
	return {{$tAlias.UpSingular}}ModFunc(func(o *{{$tAlias.UpSingular}}Template) {
		o.{{$colAlias}} = func() {{$colTyp}} { return val }
	})
}

// Set the Column from the function
func (m {{$tAlias.DownSingular}}Mods) {{$colAlias}}Func(f func() {{$colTyp}}) {{$tAlias.UpSingular}}Mod {
	return {{$tAlias.UpSingular}}ModFunc(func(o *{{$tAlias.UpSingular}}Template) {
			o.{{$colAlias}} = f
	})
}

// Clear any values for the column
func (m {{$tAlias.DownSingular}}Mods) Unset{{$colAlias}}() {{$tAlias.UpSingular}}Mod {
	return {{$tAlias.UpSingular}}ModFunc(func(o *{{$tAlias.UpSingular}}Template) {
		o.{{$colAlias}} = nil
	})
}

// Generates a random value for the column using the given faker
// if faker is nil, a default faker is used
func (m {{$tAlias.DownSingular}}Mods) Random{{$colAlias}}(f *faker.Faker) {{$tAlias.UpSingular}}Mod {
	return {{$tAlias.UpSingular}}ModFunc(func(o *{{$tAlias.UpSingular}}Template) {
		o.{{$colAlias}} = func() {{$colTyp}} {
			{{if $column.Nullable -}}
      	if f == nil {
          f = &defaultFaker
        }

        if f.Bool() {
          return null.FromPtr[{{getType $column.Type $typDef}}](nil)
        }

        return null.From(random_{{normalizeType $column.Type}}(f))
			{{- else -}}
				return random_{{normalizeType $column.Type}}(f)
			{{- end}}
		}
	})
}

{{end}}
