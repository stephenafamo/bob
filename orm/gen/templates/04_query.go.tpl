{{$tAlias := .Aliases.Table .Table.Name -}}

{{$.Importer.Import "github.com/stephenafamo/bob"}}
{{if not .Table.PKey -}}
func {{$tAlias.UpPlural}}(mods ...bob.Mod[*{{$.Dialect}}.SelectQuery]) *model.ViewQuery[*{{$tAlias.UpSingular}}, {{$tAlias.UpSingular}}Slice] {
	return {{$tAlias.UpPlural}}View.Query(mods...)
}
{{- else -}}
func {{$tAlias.UpPlural}}(mods ...bob.Mod[*{{$.Dialect}}.SelectQuery]) *model.TableQuery[*{{$tAlias.UpSingular}}, {{$tAlias.UpSingular}}Slice, *Optional{{$tAlias.UpSingular}}] {
	return {{$tAlias.UpPlural}}Table.Query(mods...)
}
{{- end}}

