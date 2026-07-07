{{$table := .Table}}
{{$tAlias := .Aliases.Table $table.Key -}}
{{$.Importer.Import "context"}}
{{$.Importer.Import "github.com/stephenafamo/scan"}}

// scan.Mapper[*T] without reflection: maps columns to fields by index.
func {{$tAlias.DownSingular}}ScanMapper(ctx context.Context, cols []string) (scan.BeforeFunc, func(any) (*{{$tAlias.UpSingular}}, error)) {
	return func(row *scan.Row) (any, error) {
		o := new({{$tAlias.UpSingular}})
		for i, col := range cols {
			switch col {
			{{range $column := $table.Columns -}}
			{{- $colAlias := $tAlias.Column $column.Name -}}
			case {{quote $column.Name}}:
				row.ScheduleScanByIndex(i, &o.{{$colAlias}})
			{{end -}}
			}
		}
		return o, nil
	}, func(v any) (*{{$tAlias.UpSingular}}, error) {
		return v.(*{{$tAlias.UpSingular}}), nil
	}
}
