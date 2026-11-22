{{- $t := $.Aliases.Table $.Table.Key -}}
{{- $rels := $.Relationships.Get $.Table.Key -}}
{{if $rels -}}
{{$.Importer.Import "github.com/stephenafamo/bob/orm"}}

// {{$t.UpSingular}}RelAssignments assigns relationships in memory for loaders, attach methods, and factories.
var {{$t.UpSingular}}RelAssignments {{$t.DownSingular}}RelAssignments
type {{$t.DownSingular}}RelAssignments struct{}

{{range $rel := $rels -}}
{{$name := $t.Relationship $rel.Name -}}
{{$f := $.Aliases.Table $rel.Foreign -}}
{{$inv := $.Relationships.GetInverse $rel -}}
{{$assignment := printf "orm.OneRelAssignment[%s, %s]" $t.UpSingular $f.UpSingular -}}
{{if $rel.IsToMany}}{{$assignment = printf "orm.ManyRelAssignment[%s, %s, %sSlice]" $t.UpSingular $f.UpSingular $f.UpSingular}}{{end -}}
func ({{$t.DownSingular}}RelAssignments) {{$name}}() {{$assignment}} {
	return {{$assignment}}{
		Slot: func(o *{{$t.UpSingular}}) ({{if $rel.IsToMany}}*{{$f.UpSingular}}Slice{{else}}**{{$f.UpSingular}}{{end}}, *bool) {
			return &o.R.{{$name}}, &o.R.{{$.RelationLoadedName}}.{{$name}}
		},
		{{if and (not $.NoBackReferencing) $inv.Name -}}
		Backlink: func(rel *{{$f.UpSingular}}, o *{{$t.UpSingular}}) {
			{{$f.UpSingular}}RelAssignments.{{$f.Relationship $inv.Name}}().{{if $inv.IsToMany}}AppendLocal{{else}}SetLocal{{end}}(rel, o)
		},
		{{end -}}
	}
}
{{end -}}
{{end -}}
