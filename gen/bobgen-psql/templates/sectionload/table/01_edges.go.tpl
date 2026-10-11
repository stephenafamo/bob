{{$table := .Table}}
{{- $tAlias := .Aliases.Table $table.Key}}
{{- $rels := $.Relationships.Get $table.Key}}
{{- if $rels}}
{{- $.Importer.Import "models" modelsPkg}}
{{- $.Importer.Import "sectionload" sectionloadPkg}}

// {{$tAlias.DownSingular}}Sections holds the tree-builder methods for edges
// starting at {{$table.Name}}.
type {{$tAlias.DownSingular}}Sections struct{}

// {{$tAlias.UpSingular}} is the entry point for building loading trees
// rooted at {{$table.Name}}: e.g. {{$tAlias.UpSingular}}.SomeRel(...).
var {{$tAlias.UpSingular}} {{$tAlias.DownSingular}}Sections

{{range $rel := $rels}}
{{- $relAlias := $tAlias.Relationship $rel.Name}}
{{- $firstSide := index $rel.Sides 0}}
{{- $lastSide := index $rel.Sides (sub (len $rel.Sides) 1)}}
{{- $isMultiHop := gt (len $rel.Sides) 1}}
{{- $fTable := $.Tables.Get $rel.Foreign}}
{{- $fAlias := $.Aliases.Table $rel.Foreign}}

// {{$relAlias}}: {{$table.Name}} ({{range $i, $c := $firstSide.FromColumns}}{{if $i}}, {{end}}{{$c}}{{end}}){{range $rel.Sides}} → {{($.Tables.Get .To).Name}}{{end}}{{if not $isMultiHop}} ({{range $i, $c := $lastSide.ToColumns}}{{if $i}}, {{end}}{{$c}}{{end}}){{end}}
func ({{$tAlias.DownSingular}}Sections) {{$relAlias}}(children ...sectionload.Node) sectionload.Node {
	return {{$tAlias.DownSingular}}Sections{}.{{$relAlias}}With(sectionload.SelectOptions{}, children...)
}

// {{$relAlias}}With is {{$relAlias}} with caller-provided options applied to
// this edge's own rows: query mods (WHERE/ORDER BY/LIMIT/OFFSET, via
// sectionload.Mods) and/or a column restriction (sectionload.Only/Except)
// on the {{$fTable.Name}} columns this edge fetches.
// {{$relAlias}}(children...) is {{$relAlias}}With with no options.
//
// LIMIT/OFFSET apply across every matched parent combined — "N rows
// total", the same meaning bob's own ThenLoad gives it — never "N rows
// per parent".
func ({{$tAlias.DownSingular}}Sections) {{$relAlias}}With(options sectionload.SelectOptions, children ...sectionload.Node) sectionload.Node {
	return sectionload.NewNode(sectionload.Edge[*models.{{$tAlias.UpSingular}}, *models.{{$fAlias.UpSingular}}]{
		RelName: {{quote $relAlias}},
		Sides: []sectionload.Side{
			{{range $s := $rel.Sides -}}
			{{$sFrom := $.Tables.Get $s.From -}}
			{{$sTo := $.Tables.Get $s.To -}}
			{
				FromSchema: {{quote $sFrom.Schema}},
				FromTable:  {{quote $sFrom.Name}},
				FromCols:   []string{ {{range $s.FromColumns}}{{quote .}}, {{end}} },
				ToSchema:   {{quote $sTo.Schema}},
				ToTable:    {{quote $sTo.Name}},
				ToCols:     []string{ {{range $s.ToColumns}}{{quote .}}, {{end}} },
				{{- $sHasWhere := or $s.FromWhere $s.ToWhere}}
				{{- if $sHasWhere}}
				Where: []sectionload.EdgeWhere{
					{{range $w := $s.FromWhere -}}
					{On: "from", Column: {{quote $w.Column}}, SQLValue: {{quote $w.SQLValue}}},
					{{end -}}
					{{range $w := $s.ToWhere -}}
					{On: "to", Column: {{quote $w.Column}}, SQLValue: {{quote $w.SQLValue}}},
					{{end -}}
				},
				{{- end}}
			},
			{{end -}}
		},
		ToMany:  {{$rel.IsToMany}},
		Options: options,
		{{- if $isMultiHop}}
		// This edge is multi-hop ({{len $rel.Sides}} sides) — the final
		// table ({{$fTable.Name}}) has no column holding {{$table.Name}}'s own
		// join key, so it is carried alongside each row as an extra SELECT
		// column instead (Carried/NewCarriedBuf below).
		Carried: []sectionload.Column{
			{{range $i, $col := $firstSide.FromColumns -}}
			{{$c := $table.GetColumn $col -}}
			{Name: sectionload.CarriedKeyName({{$i}}), DBType: {{quote $c.DBType}}},
			{{end -}}
		},
		NewCarriedBuf: {{$tAlias.UpSingular}}Parts.NewBuf,
		{{- end}}
		// ParentKeyFromBuf: generated for EVERY edge, single-hop
		// included, because validatePlan (validate.go) requires it on ANY
		// edge that might be nested as a child under a ColumnPlan-restricted
		// parent edge — not only on a multi-hop edge's own carried-key read
		// (which this same field also serves). The body
		// reads sides()[0]'s own FromColumns off the PARENT table
		// ({{$tAlias.UpSingular}}), identical to ParentKey below; for a
		// single-hop edge these are the same columns ParentKey already
		// reads.
		ParentKeyFromBuf: func(b sectionload.Buf[*models.{{$tAlias.UpSingular}}]) []any {
			o := b.BuildFor([]sectionload.Column{
				{{range $col := $firstSide.FromColumns -}}
				{{$c := $table.GetColumn $col -}}
				{Name: {{quote $col}}, DBType: {{quote $c.DBType}}},
				{{end -}}
			})
			{{- range $col := $firstSide.FromColumns}}
			{{- $c := $table.GetColumn $col}}
			{{- if $c.Nullable}}
			if !({{$.Types.GetNullTypeValid $.CurrentPackage $c.Type (printf "o.%s" ($tAlias.Column $col))}}) {
				return nil
			}
			{{- end}}
			{{- end}}
			return []any{
				{{range $col := $firstSide.FromColumns -}}
				{{$c := $table.GetColumn $col -}}
				{{$.Types.UnwrapNullExpr $.CurrentPackage $.Importer $c.Type (printf "o.%s" ($tAlias.Column $col)) $c.Nullable}},
				{{end -}}
			}
		},
		Child: {{$fAlias.UpSingular}}Parts,
		ParentKey: func(o *models.{{$tAlias.UpSingular}}) []any {
			{{- range $col := $firstSide.FromColumns}}
			{{- $c := $table.GetColumn $col}}
			{{- if $c.Nullable}}
			if !({{$.Types.GetNullTypeValid $.CurrentPackage $c.Type (printf "o.%s" ($tAlias.Column $col))}}) {
				return nil
			}
			{{- end}}
			{{- end}}
			return []any{
				{{range $col := $firstSide.FromColumns -}}
				{{$c := $table.GetColumn $col -}}
				{{$.Types.UnwrapNullExpr $.CurrentPackage $.Importer $c.Type (printf "o.%s" ($tAlias.Column $col)) $c.Nullable}},
				{{end -}}
			}
		},
		// ChildKey reads Sides[len-1].ToColumns — the FINAL table's own
		// columns. For a multi-hop edge these are never what
		// stitching actually uses (childKeyFrom reads the carried buffer
		// instead, node.go), but it is still generated so ChildKey's
		// shape and contract match a single-hop edge's exactly.
		ChildKey: func(o *models.{{$fAlias.UpSingular}}) []any {
			{{- range $col := $lastSide.ToColumns}}
			{{- $c := $fTable.GetColumn $col}}
			{{- if $c.Nullable}}
			if !({{$.Types.GetNullTypeValid $.CurrentPackage $c.Type (printf "o.%s" ($fAlias.Column $col))}}) {
				return nil
			}
			{{- end}}
			{{- end}}
			return []any{
				{{range $col := $lastSide.ToColumns -}}
				{{$c := $fTable.GetColumn $col -}}
				{{$.Types.UnwrapNullExpr $.CurrentPackage $.Importer $c.Type (printf "o.%s" ($fAlias.Column $col)) $c.Nullable}},
				{{end -}}
			}
		},
		// ChildKeyFromBuf: generated for EVERY edge, so any edge can
		// have its ColumnPlan restrict its own columns (validatePlan
		// requires this for the single-hop case, see validate.go). Reads
		// sides()[len-1]'s own ToColumns off the CHILD table
		// ({{$fAlias.UpSingular}}), identical to ChildKey above.
		ChildKeyFromBuf: func(b sectionload.Buf[*models.{{$fAlias.UpSingular}}]) []any {
			o := b.BuildFor([]sectionload.Column{
				{{range $col := $lastSide.ToColumns -}}
				{{$c := $fTable.GetColumn $col -}}
				{Name: {{quote $col}}, DBType: {{quote $c.DBType}}},
				{{end -}}
			})
			{{- range $col := $lastSide.ToColumns}}
			{{- $c := $fTable.GetColumn $col}}
			{{- if $c.Nullable}}
			if !({{$.Types.GetNullTypeValid $.CurrentPackage $c.Type (printf "o.%s" ($fAlias.Column $col))}}) {
				return nil
			}
			{{- end}}
			{{- end}}
			return []any{
				{{range $col := $lastSide.ToColumns -}}
				{{$c := $fTable.GetColumn $col -}}
				{{$.Types.UnwrapNullExpr $.CurrentPackage $.Importer $c.Type (printf "o.%s" ($fAlias.Column $col)) $c.Nullable}},
				{{end -}}
			}
		},
		Stitch: func(p *models.{{$tAlias.UpSingular}}, children []*models.{{$fAlias.UpSingular}}) error {
			{{- if $rel.IsToMany}}
			return p.Preload({{quote $relAlias}}, models.{{$fAlias.UpSingular}}Slice(children))
			{{- else}}
			// Always call Preload, even with a typed nil, so that
			// R.Loaded.{{$relAlias}} becomes true when no child was found
			// (bob Preload parity — nil/true, not nil/false).
			var related *models.{{$fAlias.UpSingular}}
			if len(children) > 0 {
				related = children[0]
			}
			return p.Preload({{quote $relAlias}}, related)
			{{- end}}
		},
	}, children...)
}
{{end}}
{{- end}}
