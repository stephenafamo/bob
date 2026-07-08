{{- $table := .Table}}
{{- $tAlias := .Aliases.Table $table.Key -}}
{{- /* Only generate for tables that are the target (foreign side) of at least
       one to-one relationship: those are the only tables Preload can load,
       and generating unreferenced mappers would trip the `unused` linter. */ -}}
{{- $isPreloadTarget := false -}}
{{- range $t := $.AllTables -}}
{{- range $rel := $.Relationships.Get $t.Key -}}
{{- if and (not $rel.IsToMany) (eq $rel.Foreign $table.Key) -}}{{- $isPreloadTarget = true -}}{{- end -}}
{{- end -}}
{{- end -}}
{{- if $isPreloadTarget -}}
{{$.Importer.Import "context" -}}
{{$.Importer.Import "strings" -}}
{{$.Importer.Import "github.com/stephenafamo/scan"}}

// NULL-tolerant scan buffer for {{$tAlias.DownSingular}}ScanMapperNullable:
// on a LEFT JOIN miss every column comes back NULL, so each field uses the
// nullable version of the column type even when the column itself is NOT NULL.
type {{$tAlias.DownSingular}}PreloadBuf struct {
	{{range $column := $table.Columns -}}
	{{$tAlias.Column $column.Name}} {{$.Types.GetNullable $.CurrentPackage $.Importer $column.Type true}}
	{{end -}}
}

// {{$tAlias.UpSingular}}ScanMapperNullable maps the preloaded {{$tAlias.DownSingular}}
// columns (prefixed with the runtime join alias) without reflection, while
// keeping the LEFT JOIN semantics of the reflection-based mapper: a row whose
// prefixed columns are all NULL yields nil (no child), and NULL values never
// error, they just leave the zero value in the field.
func {{$tAlias.UpSingular}}ScanMapperNullable(prefix string) scan.Mapper[*{{$tAlias.UpSingular}}] {
	return func(ctx context.Context, cols []string) (scan.BeforeFunc, func(any) (*{{$tAlias.UpSingular}}, error)) {
		// resolve the column names once per query, not per row
		type target struct {
			idx int
			dst func(b *{{$tAlias.DownSingular}}PreloadBuf) any
		}
		targets := make([]target, 0, {{len $table.Columns}})
		for i, col := range cols {
			name, ok := strings.CutPrefix(col, prefix)
			if !ok {
				continue
			}
			switch name {
			{{range $column := $table.Columns -}}
			{{- $colAlias := $tAlias.Column $column.Name -}}
			case {{quote $column.Name}}:
				targets = append(targets, target{i, func(b *{{$tAlias.DownSingular}}PreloadBuf) any { return &b.{{$colAlias}} }})
			{{end -}}
			}
		}

		return func(row *scan.Row) (any, error) {
			buf := new({{$tAlias.DownSingular}}PreloadBuf)
			for _, t := range targets {
				row.ScheduleScanByIndex(t.idx, t.dst(buf))
			}
			return buf, nil
		}, func(link any) (*{{$tAlias.UpSingular}}, error) {
			buf := link.(*{{$tAlias.DownSingular}}PreloadBuf)

			// Same rule as the reflection mapper's row validator: the child
			// exists only if at least one of its columns is not NULL. Columns
			// not selected by the query are never scanned and stay invalid,
			// so this check also matches when only a subset is selected.
			if {{range $i, $column := $table.Columns}}{{if $i}} &&
				{{end}}!({{$.Types.GetNullTypeValid $.CurrentPackage $column.Type (printf "buf.%s" ($tAlias.Column $column.Name))}}){{end}} {
				return nil, nil
			}

			o := new({{$tAlias.UpSingular}})
			{{range $column := $table.Columns -}}
			{{- $colAlias := $tAlias.Column $column.Name -}}
			{{- if $column.Nullable -}}
			o.{{$colAlias}} = buf.{{$colAlias}}
			{{else -}}
			if {{$.Types.GetNullTypeValid $.CurrentPackage $column.Type (printf "buf.%s" $colAlias)}} {
				o.{{$colAlias}} = {{$.Types.UnwrapNullExpr $.CurrentPackage $.Importer $column.Type (printf "buf.%s" $colAlias) true}}
			}
			{{end -}}
			{{end -}}
			return o, nil
		}
	}
}
{{end -}}
