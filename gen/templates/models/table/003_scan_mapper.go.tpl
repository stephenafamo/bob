{{$table := .Table}}
{{$tAlias := .Aliases.Table $table.Key -}}
{{$.Importer.Import "context"}}
{{$.Importer.Import "github.com/stephenafamo/scan"}}

// scan.Mapper[*T] without reflection: maps columns to fields by index.
func {{$tAlias.DownSingular}}ScanMapper(ctx context.Context, cols []string) (scan.BeforeFunc, func(any) (*{{$tAlias.UpSingular}}, error)) {
	// resolve the column positions once per query, not per row
	type target struct {
		idx int
		dst func(o *{{$tAlias.UpSingular}}) any
	}
	targets := make([]target, 0, {{len $table.Columns}})
	for i, col := range cols {
		switch col {
		{{range $column := $table.Columns -}}
		{{- $colAlias := $tAlias.Column $column.Name -}}
		case {{quote $column.Name}}:
			targets = append(targets, target{i, func(o *{{$tAlias.UpSingular}}) any { return &o.{{$colAlias}} }})
		{{end -}}
		}
	}

	return func(row *scan.Row) (any, error) {
		o := new({{$tAlias.UpSingular}})
		for _, t := range targets {
			row.ScheduleScanByIndex(t.idx, t.dst(o))
		}
		return o, nil
	}, func(v any) (*{{$tAlias.UpSingular}}, error) {
		return v.(*{{$tAlias.UpSingular}}), nil
	}
}
