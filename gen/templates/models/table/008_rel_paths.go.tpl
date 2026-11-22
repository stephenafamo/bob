{{$table := .Table -}}
{{$t := $.Aliases.Table $table.Key -}}
{{range $rel := $.Relationships.Get $table.Key -}}
{{$name := $t.Relationship $rel.Name -}}
// Shared by relationship filters, joins and preloaders.
var {{$t.DownSingular}}Rel{{$name}} = {{$.Dialect}}.PreloadRel{
	Name: {{quote $name}},
	Sides: []{{$.Dialect}}.PreloadSide{
		{{range $side := $rel.Sides -}}
		{
			From: {{($.Aliases.Table $side.From).UpPlural}}, To: {{($.Aliases.Table $side.To).UpPlural}},
			FromColumns: []string{ {{range $side.FromColumns}}{{quote .}},{{end}} },
			ToColumns: []string{ {{range $side.ToColumns}}{{quote .}},{{end}} },
			{{range $kind, $wheres := dict "FromWhere" $side.FromWhere "ToWhere" $side.ToWhere -}}
			{{if $wheres -}}
			{{$.Importer.Import "github.com/stephenafamo/bob/orm"}}
			{{$kind}}: []orm.RelWhere{
				{{range $wheres -}}
				{Column: {{quote .Column}}, SQLValue: {{quote .SQLValue}}, GoValue: {{quote .GoValue}}},
				{{end -}}
			},
			{{end -}}
			{{end -}}
		},
		{{end -}}
	},
}
{{end -}}
