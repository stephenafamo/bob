{{$.Importer.Import "context"}}
{{$.Importer.Import "models" (index $.OutputPackages "models") }}
{{$.Importer.Import "sync"}}
{{range $table := .Tables}}{{if $.Relationships.Get $table.Key}}{{$.Importer.Import "unsafe"}}{{end}}{{end}}

type Factory struct {
    // visited tracks model pointers during FromExisting calls to prevent circular reference stack overflow
    visited sync.Map
    {{range $table := .Tables}}
    {{ $tAlias := $.Aliases.Table $table.Key -}}
		base{{$tAlias.UpSingular}}Mods {{$tAlias.UpSingular}}ModSlice
    {{- end}}
}

func New() *Factory {
  return &Factory{}
}

{{range $table := .Tables}}
{{ $tAlias := $.Aliases.Table $table.Key -}}
func (f *Factory) New{{$tAlias.UpSingular}}(mods ...{{$tAlias.UpSingular}}Mod) *{{$tAlias.UpSingular}}Template {
	return f.New{{$tAlias.UpSingular}}WithContext(context.Background(), mods...)
}

func (f *Factory) New{{$tAlias.UpSingular}}WithContext(ctx context.Context, mods ...{{$tAlias.UpSingular}}Mod) *{{$tAlias.UpSingular}}Template {
	o := &{{$tAlias.UpSingular}}Template{f: f}

  if f != nil {
    f.base{{$tAlias.UpSingular}}Mods.Apply(ctx, o)
  }

  {{$tAlias.UpSingular}}ModSlice(mods).Apply(ctx, o)

	return o
}

func (f *Factory) FromExisting{{$tAlias.UpSingular}}(m *models.{{$tAlias.UpSingular}}) *{{$tAlias.UpSingular}}Template {
	o := &{{$tAlias.UpSingular}}Template{f: f, alreadyPersisted: true}

  {{range $column := $table.Columns -}}
  {{$colAlias := $tAlias.Column $column.Name -}}
  {{- $colTyp := $.Types.GetNullable $.CurrentPackage $.Importer $column.Type $column.Nullable -}}
        o.{{$colAlias}} = func() {{$colTyp}} { return m.{{$colAlias}} }
  {{end}}

  {{if $.Relationships.Get $table.Key -}}
  // Check for circular references using Factory-level visited map to prevent stack overflow
  // See https://github.com/stephenafamo/bob/issues/584
  ptr := uintptr(unsafe.Pointer(m))
  if _, loaded := f.visited.LoadOrStore(ptr, struct{}{}); loaded {
    return o // Already processing this model, skip to prevent infinite recursion
  }
  defer f.visited.Delete(ptr) // Clean up after processing
  {{- end}}
  {{range $.Relationships.Get $table.Key -}}
    {{$relAlias := $tAlias.Relationship .Name -}}
    {{if .IsToMany -}}
      if len(m.R.{{$relAlias}}) > 0 {
      {{$tAlias.UpSingular}}Mods.AddExisting{{$relAlias}}(m.R.{{$relAlias}}...).Apply(context.Background(), o)
      }
    {{- else -}}
      if m.R.{{$relAlias}} != nil {
      {{$tAlias.UpSingular}}Mods.WithExisting{{$relAlias}}(m.R.{{$relAlias}}).Apply(context.Background(), o)
      }
    {{- end}}
  {{end}}

  return o
}

{{end}}

{{range $table := .Tables}}
{{ $tAlias := $.Aliases.Table $table.Key -}}
func (f *Factory) ClearBase{{$tAlias.UpSingular}}Mods() {
    f.base{{$tAlias.UpSingular}}Mods = nil
}

func (f *Factory) AddBase{{$tAlias.UpSingular}}Mod(mods ...{{$tAlias.UpSingular}}Mod) {
f.base{{$tAlias.UpSingular}}Mods = append(f.base{{$tAlias.UpSingular}}Mods, mods...)
}

{{end}}

