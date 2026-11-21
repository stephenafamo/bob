{{if or .Table.Constraints.Uniques .Table.Constraints.Primary .Table.Constraints.Checks }}
{{- $table := .Table -}}
{{- $tAlias := .Aliases.Table $table.Key -}}

var {{$tAlias.UpSingular}}Errors = &{{$tAlias.DownSingular}}Errors{
  {{if $table.Constraints.Primary}}
  {{$pk := $table.Constraints.Primary}}
	ErrUnique{{$pk.Name | camelcase}}: &UniqueConstraintError{
    schema: {{printf "%q" $table.Schema}},
    table: {{printf "%q" $table.Name}},
    columns: {{printf "%#v" $pk.Columns}},
    s: {{printf "%q" $pk.Name}},
  },
  {{end}}
	{{range $index := $table.Constraints.Uniques}}
	ErrUnique{{$index.Name | camelcase}}: &UniqueConstraintError{
    schema: {{printf "%q" $table.Schema}},
    table: {{printf "%q" $table.Name}},
    columns: {{printf "%#v" $index.Columns}},
    s: "{{$index.Name}}",
  },
	{{end}}
	{{range $check := $table.Constraints.Checks}}
	ErrCheck{{$check.Name | camelcase}}: &CheckConstraintError{
    schema: {{printf "%q" $table.Schema}},
    table: {{printf "%q" $table.Name}},
    columns: {{printf "%#v" $check.Columns}},
    s: {{printf "%q" $check.Name}},
  },
	{{end}}
}

type {{$tAlias.DownSingular}}Errors struct {
  {{if $table.Constraints.Primary}}
  {{$pk := $table.Constraints.Primary}}
	ErrUnique{{$pk.Name | camelcase}} *UniqueConstraintError
  {{end}}
	{{range $index := $table.Constraints.Uniques}}
	ErrUnique{{$index.Name | camelcase}} *UniqueConstraintError
	{{end}}
	{{range $check := $table.Constraints.Checks}}
	ErrCheck{{$check.Name | camelcase}} *CheckConstraintError
	{{end}}
}
{{ end -}}
