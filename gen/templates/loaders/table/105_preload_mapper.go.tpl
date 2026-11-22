{{- $table := .Table}}
{{- $tAlias := .Aliases.Table $table.Key -}}
{{- /* Only generate for tables that are the target (foreign side) of at least
       one to-one relationship: those are the only tables Preload can load,
       and generating unreferenced mappers would trip the `unused` linter. */ -}}
{{- $isPreloadTarget := false -}}
{{- range $t := $.Tables -}}
{{- range $rel := $.Relationships.Get $t.Key -}}
{{- if and (not $rel.IsToMany) (eq $rel.Foreign $table.Key) -}}{{- $isPreloadTarget = true -}}{{- end -}}
{{- end -}}
{{- end -}}
{{- if $isPreloadTarget -}}
{{$.Importer.Import "github.com/stephenafamo/bob/orm" -}}
{{$.Importer.Import "github.com/stephenafamo/scan"}}

// NULL-tolerant scan buffer for {{$tAlias.DownSingular}}ScanMapperNullable:
// on a LEFT JOIN miss every column comes back NULL, so each field uses the
// nullable version of the column type even when the column itself is NOT NULL.
type {{$tAlias.DownSingular}}PreloadBuf struct {
	{{range $column := $table.Columns -}}
	{{$tAlias.Column $column.Name}} {{$.Types.GetNullable $.CurrentPackage $.Importer $column.Type true}}
	{{end -}}
}

// {{$tAlias.DownSingular}}ScanMapperNullable maps the preloaded {{$tAlias.DownSingular}}
// columns (prefixed with the runtime join alias) without reflection, while
// keeping the LEFT JOIN semantics of the reflection-based mapper: a row whose
// prefixed columns are all NULL yields nil (no child), and NULL values never
// error, they just leave the zero value in the field.
func {{$tAlias.DownSingular}}ScanMapperNullable(prefix string) scan.Mapper[*{{$tAlias.UpSingular}}] {
	return orm.JoinedModelMapper(prefix, func(name string) func(*{{$tAlias.DownSingular}}PreloadBuf) any {
			switch name {
			{{range $column := $table.Columns -}}
			{{- $colAlias := $tAlias.Column $column.Name -}}
			case {{quote $column.Name}}:
				return func(b *{{$tAlias.DownSingular}}PreloadBuf) any { return &b.{{$colAlias}} }
			{{end -}}
			default: return nil
			}
		}, func(buf *{{$tAlias.DownSingular}}PreloadBuf) *{{$tAlias.UpSingular}} {

			// Same rule as the reflection mapper's row validator: the child
			// exists only if at least one of its columns is not NULL. Columns
			// not selected by the query are never scanned and stay invalid,
			// so this check also matches when only a subset is selected.
			if {{range $i, $column := $table.Columns}}{{if $i}} &&
				{{end}}!({{$.Types.GetNullTypeValid $.CurrentPackage $column.Type (printf "buf.%s" ($tAlias.Column $column.Name))}}){{end}} {
				return nil
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
			return o
		})
}
{{end -}}
