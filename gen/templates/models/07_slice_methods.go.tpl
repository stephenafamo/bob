{{if .Table.Constraints.Primary -}}
{{$.Importer.Import (printf "github.com/stephenafamo/bob/dialect/%s/sm" $.Dialect)}}
{{$table := .Table}}
{{$tAlias := .Aliases.Table $table.Key -}}

func (o {{$tAlias.UpSingular}}Slice) UpdateAll(ctx context.Context, exec bob.Executor, vals {{$tAlias.UpSingular}}Setter) error {
	return {{$tAlias.UpPlural}}.Update(ctx, exec, &vals, o...)
}

func (o {{$tAlias.UpSingular}}Slice) DeleteAll(ctx context.Context, exec bob.Executor) error {
	return {{$tAlias.UpPlural}}.Delete(ctx, exec, o...)
}


func (o {{$tAlias.UpSingular}}Slice) ReloadAll(ctx context.Context, exec bob.Executor) error {
  var mods []bob.Mod[*dialect.SelectQuery]

	{{range $colName := $table.Constraints.Primary.Columns -}}
		{{$column := $table.GetColumn $colName -}}
		{{$colAlias := $tAlias.Column $colName -}}
		{{$colAlias}}PK := make([]{{$column.Type}}, len(o))
	{{end}}

	for i, o := range o {
		{{range $column := $table.Constraints.Primary.Columns -}}
		{{$colAlias := $tAlias.Column $column -}}
			{{$colAlias}}PK[i] = o.{{$colAlias}}
		{{end -}}
	}

	mods = append(mods, 
	{{range $column := $table.Constraints.Primary.Columns -}}
		{{- $colAlias := $tAlias.Column $column -}}
		SelectWhere.{{$tAlias.UpPlural}}.{{$colAlias}}.In({{$colAlias}}PK...),
	{{end}}
	)

	o2, err := {{$tAlias.UpPlural}}.Query(ctx, exec, mods...).All()
	if err != nil {
		return err
	}

	for _, old := range o {
		for _, new := range o2 {
			{{range $column := $table.Constraints.Primary.Columns -}}
			{{- $colAlias := $tAlias.Column $column -}}
			if new.{{$colAlias}} != old.{{$colAlias}} {
				continue
			}
			{{end -}}
			{{if $.Relationships.Get $table.Key}}new.R = old.R{{end}}
			*old = *new
			break
		}
	}

	return nil
}

{{- end}}

