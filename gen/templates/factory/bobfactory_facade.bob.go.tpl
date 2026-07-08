{{- $isSplit := and $.ModelSplit $.ModelSplit.Enabled -}}
{{- $isFacade := and $isSplit (eq $.ModelSplit.Generation "facade") -}}
{{- if $isFacade -}}
{{$.Importer.Import "context"}}
{{$.Importer.Import "models" (index $.OutputPackages "models") }}

type Factory struct {
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
type {{$tAlias.UpSingular}}Template = {{$.FactoryTemplateType $table.Key}}
type {{$tAlias.UpSingular}}Mod = {{$.FactoryModType $table.Key}}
type {{$tAlias.UpSingular}}ModFunc = {{$.FactoryModFuncType $table.Key}}
type {{$tAlias.UpSingular}}ModSlice = {{$.FactoryModSliceType $table.Key}}

var {{$tAlias.UpSingular}}Mods = {{$.FactoryModsVar $table.Key}}

func (f *Factory) New{{$tAlias.UpSingular}}(mods ...{{$tAlias.UpSingular}}Mod) *{{$tAlias.UpSingular}}Template {
	return f.New{{$tAlias.UpSingular}}WithContext(context.Background(), mods...)
}

func (f *Factory) New{{$tAlias.UpSingular}}WithContext(ctx context.Context, mods ...{{$tAlias.UpSingular}}Mod) *{{$tAlias.UpSingular}}Template {
	var baseMods {{$tAlias.UpSingular}}ModSlice
	if f != nil {
		baseMods = f.base{{$tAlias.UpSingular}}Mods
	}

	return {{$.FactoryNewWithContextFunc $table.Key}}(ctx, f, baseMods, mods...)
}

func (f *Factory) FromExisting{{$tAlias.UpSingular}}(ctx context.Context, m *models.{{$tAlias.UpSingular}}) *{{$tAlias.UpSingular}}Template {
	return {{$.FactoryFromExistingFunc $table.Key}}(ctx, f, m)
}

func (f *Factory) ClearBase{{$tAlias.UpSingular}}Mods() {
	f.base{{$tAlias.UpSingular}}Mods = nil
}

func (f *Factory) AddBase{{$tAlias.UpSingular}}Mod(mods ...{{$tAlias.UpSingular}}Mod) {
	f.base{{$tAlias.UpSingular}}Mods = append(f.base{{$tAlias.UpSingular}}Mods, mods...)
}

{{end}}
{{end}}
