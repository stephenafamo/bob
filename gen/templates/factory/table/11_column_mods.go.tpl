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
{{- $colTypBase := $.Types.Get $.CurrentPackage $.Importer $column.Type -}}
{{- $colTyp := $.Types.GetNullable $.CurrentPackage $.Importer $column.Type $column.Nullable -}}

// Set the model columns to this value
func (m {{$tAlias.DownSingular}}Mods) {{$colAlias}}(val {{$colTyp}}) {{$tAlias.UpSingular}}Mod {
	return {{$tAlias.UpSingular}}ModFunc(func(_ context.Context, o *{{$tAlias.UpSingular}}Template) {
		o.{{$colAlias}} = func() {{$colTyp}} { return val }
	})
}

// Set the Column from the function
func (m {{$tAlias.DownSingular}}Mods) {{$colAlias}}Func(f func() {{$colTyp}}) {{$tAlias.UpSingular}}Mod {
	return {{$tAlias.UpSingular}}ModFunc(func(_ context.Context, o *{{$tAlias.UpSingular}}Template) {
			o.{{$colAlias}} = f
	})
}

// Clear any values for the column
func (m {{$tAlias.DownSingular}}Mods) Unset{{$colAlias}}() {{$tAlias.UpSingular}}Mod {
	return {{$tAlias.UpSingular}}ModFunc(func(_ context.Context, o *{{$tAlias.UpSingular}}Template) {
		o.{{$colAlias}} = nil
	})
}

// Generates a random value for the column using the given faker
// if faker is nil, a default faker is used
{{if not $column.Nullable -}}
  func (m {{$tAlias.DownSingular}}Mods) Random{{$colAlias}}(f *faker.Faker) {{$tAlias.UpSingular}}Mod {
    return {{$tAlias.UpSingular}}ModFunc(func(_ context.Context, o *{{$tAlias.UpSingular}}Template) {
      o.{{$colAlias}} = func() {{$colTyp}} {
        return random_{{normalizeType $column.Type}}(f, {{$column.LimitsString}})
      }
    })
  }
{{- else -}}
  // The generated value is sometimes null
  func (m {{$tAlias.DownSingular}}Mods) Random{{$colAlias}}(f *faker.Faker) {{$tAlias.UpSingular}}Mod {
    return {{$tAlias.UpSingular}}ModFunc(func(_ context.Context, o *{{$tAlias.UpSingular}}Template) {
      o.{{$colAlias}} = func() {{$colTyp}} {
          if f == nil {
            f = &defaultFaker
          }

          val := random_{{normalizeType $column.Type}}(f, {{$column.LimitsString}})
          return {{$.Tables.ColumnSetter $.CurrentPackage $.Importer $.Types $table.Key $column.Name "val" "f.Bool()"}}
      }
    })
  }

  // Generates a random value for the column using the given faker
  // if faker is nil, a default faker is used
  // The generated value is never null
  func (m {{$tAlias.DownSingular}}Mods) Random{{$colAlias}}NotNull(f *faker.Faker) {{$tAlias.UpSingular}}Mod {
    return {{$tAlias.UpSingular}}ModFunc(func(_ context.Context, o *{{$tAlias.UpSingular}}Template) {
      o.{{$colAlias}} = func() {{$colTyp}} {
          if f == nil {
            f = &defaultFaker
          }

          val := random_{{normalizeType $column.Type}}(f, {{$column.LimitsString}})
          return {{$.Tables.ColumnSetter $.CurrentPackage $.Importer $.Types $table.Key $column.Name "val" "true"}}
      }
    })
  }
{{end}}

{{end}}
