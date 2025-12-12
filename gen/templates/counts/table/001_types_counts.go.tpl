{{- define "model/fields/counts" -}}
{{- $table := .Table -}}
{{- $tAlias := .Aliases.Table $table.Key -}}
{{- $rels := $.Relationships.Get $table.Key -}}
{{- $hasToMany := false -}}
{{- range $rel := $rels -}}
	{{- if $rel.IsToMany -}}{{- $hasToMany = true -}}{{- end -}}
{{- end -}}
{{- if $hasToMany}}

	C {{$tAlias.DownSingular}}C `db:"-" {{generateTags $.Tags $.RelationTag | trim}}`
{{- end -}}
{{- end -}}
