{{/*
Note: if a relationship's alias collides with another relationship's
<Rel>Count (e.g. one rel aliased "Foo" and another aliased "FooCount"),
the two generated methods double-define and the generated package fails
to compile. That is a loud, build-time failure, not a silent one.
*/}}
{{$table := .Table}}
{{- $tAlias := .Aliases.Table $table.Key}}
{{- $rels := $.Relationships.Get $table.Key}}
{{- range $rel := $rels}}
{{- if not $rel.IsToMany}}{{continue}}{{end}}
{{- $relAlias := $tAlias.Relationship $rel.Name}}
{{- $.Importer.Import "models" modelsPkg}}
{{- $.Importer.Import "sectionload" sectionloadPkg}}
{{- $firstSide := index $rel.Sides 0}}
{{- $lastSide := index $rel.Sides (sub (len $rel.Sides) 1)}}
{{- $isMultiHop := gt (len $rel.Sides) 1}}
{{- $fTable := $.Tables.Get $rel.Foreign}}
{{- $fAlias := $.Aliases.Table $rel.Foreign}}

// {{$relAlias}}Count loads ONLY the number of {{$relAlias}} rows per
// {{$table.Name}}, never the rows themselves: it sets C.{{$relAlias}}
// (*int64) on every parent, 0 included, exactly like bob's
// SelectThenLoadCount / LoadCount{{$relAlias}}. A nil C.{{$relAlias}} means
// "not loaded"; a pointer to 0 means "loaded, no {{$relAlias}}".
//
// It takes no children on purpose: a count node loads no models, so there
// is nothing to nest under it, and leaving the parameter out makes that a
// compile error (issue P-04 §10.4).
{{- if $isMultiHop}}
//
// R-24: this relationship is multi-hop ({{len $rel.Sides}} sides) — the
// grouped key is a column of the FIRST intermediate table, which has no
// TableParts of its own to introspect at runtime, so KeyCols/NewCarriedBuf
// below carry the DBType/scan-buffer information the generator already
// knows at generation time (sectionload.CountEdge's own doc, P-34-multihop-
// count.md §7.1).
{{- end}}
//
// options apply to the count's own aggregate SELECT:
//   - WHERE narrows which {{$fTable.Name}} rows are counted.
//   - LIMIT/OFFSET apply to the GROUPED rows — "the first N distinct
//     parent keys", not "N children" — so a parent outside that window
//     counts 0 (still non-nil). This is what bob's own LoadCount{{$relAlias}}
//     does with the same mods.
//   - ORDER BY passes through verbatim. Alone it cannot change any count;
//     combined with LIMIT/OFFSET it decides WHICH parent keys are counted
//     (a deterministic window), again matching bob's LoadCount{{$relAlias}}.
//
// sectionload.Only/Except (P-33) cannot be used here: a count's SELECT is
// always exactly the join key columns plus count(*), so there is no model
// column to restrict. Passing Only/Except in options is rejected outright
// with sectionload.CountColumnRestrictionUnsupportedError.
//
// Pass sectionload.SelectOptions{} for no options at all.
func ({{$tAlias.DownSingular}}Sections) {{$relAlias}}Count(options sectionload.SelectOptions) sectionload.Node {
	return sectionload.NewCountNode(sectionload.CountEdge[*models.{{$tAlias.UpSingular}}, *models.{{$fAlias.UpSingular}}]{
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
		Options: options,
		{{- if $isMultiHop}}
		// R-24: this edge is multi-hop ({{len $rel.Sides}} sides) — see
		// sectionload.CountEdge.KeyCols's doc for why DBType is copied from
		// the PARENT's own column instead of the (nonexistent) intermediate
		// table's TableParts.
		KeyCols: []sectionload.Column{
			{{range $i, $col := $firstSide.ToColumns -}}
			{{$pCol := index $firstSide.FromColumns $i -}}
			{{$c := $table.GetColumn $pCol -}}
			{Name: {{quote $col}}, DBType: {{quote $c.DBType}}},
			{{end -}}
		},
		NewCarriedBuf: {{$tAlias.UpSingular}}Parts.NewBuf,
		{{- end}}
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
		// ParentKeyFromBuf (P-33): generated so a count node nested under a
		// ColumnPlan-restricted parent row edge can read that parent's
		// partially-scanned buffer (validatePlan requires this,
		// validate.go:111-114). Reads the count's own Sides[0].FromColumns
		// off the PARENT table ({{$tAlias.UpSingular}}), identical to
		// ParentKey above.
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
		// ChildKeyFromBuf (I5, Opus review 2026-08-18): generated so a
		// count node's own scan buffer can be read without falling back
		// to BuildFor(nil). This is NOT required by validatePlan (a
		// CountEdge has no ColumnPlan field at all, so it is never
		// itself "restricted" — §6.7), and is therefore not part of
		// P-33's original design goal. It is added here as a
		// piggyback fix on the SAME struct literal P-33 already edits
		// (§7.4 note below explains why): without it,
		// childKeyFromBufAny's fallback (node.go:421-426) calls
		// b.BuildFor(nil) — "every declared field" — against a buffer
		// that a single-hop count section only ever scans keyCols into
		// (exec.go, `targets = append(targets, b.targetsFor(c.keyCols)...)`).
		// Generated Buf.BuildFor(nil) unconditionally calls .MustGet()
		// on every NOT NULL column of the table (e.g.
		// pkg/dbsections/servers.bob.go:243-255), which is a candidate
		// for a panic on any NOT NULL column outside keyCols — see the
		// caveat immediately below. Reads Sides[len-1]'s own ToColumns
		// off the CHILD table ({{$fAlias.UpSingular}}), identical in
		// shape to 01_edges.go.tpl's ChildKeyFromBuf and to ChildKey
		// above. R-24: this is unreachable for a multi-hop count
		// (childKeyFrom, node.go, routes multi-hop through
		// NewCarriedBuf/ParentKeyFromBuf instead), but is still generated
		// so ChildKeyFromBuf's shape and contract match a single-hop
		// count exactly.
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
		SetCount: func(o *models.{{$tAlias.UpSingular}}, count int64) error {
			if o == nil {
				return nil
			}
			if err := o.PreloadCount({{quote $relAlias}}, count); err != nil {
				return err
			}
			// A-04-4: bob's PreloadCount is a switch with no default case,
			// so an unknown name is a silent no-op. Reading the field back
			// is a typed access, not reflection — and if the relationship
			// ever disappears from pkg/dbmodels, this line stops compiling,
			// which is a stronger guarantee than any runtime check.
			if o.C.{{$relAlias}} == nil {
				return &sectionload.StaleCountContractError{RelName: {{quote $relAlias}}}
			}
			return nil
		},
	})
}
{{- end}}
