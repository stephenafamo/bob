{{$.Importer.Import "models" modelsPkg}}

// HookProbe reports, for ONE table, whether SELECT-time hooks are
// registered on its bob model at runtime, and whether sectionload parts
// were generated for it.
//
// Background: sections and Then(...) islands are
// composed as raw SQL and never pass through <Table>.Query(), so hooks
// registered on a child table would silently stop firing. Such hooks
// typically implement row-level filtering (e.g. tenancy), so "silently
// stop firing" can mean rows are returned that the hooks would have
// filtered out. Layer 1 of the defense is
// to not generate parts for such tables (making the unsafe call a compile
// error); this table is layer 2: it lets a test assert that the exclusion
// list still covers every table that actually has hooks, so a newly
// registered hook cannot slip through unnoticed.
type HookProbe struct {
	// Table is the database table name.
	Table string
	// Excluded is true when this table has NO generated parts.
	Excluded bool
	// HasHooks reports whether SELECT-time hooks are registered right now.
	// Call it after the application's hook registrations have run (they
	// live in package init functions).
	HasHooks func() bool
}

// HookProbes covers every table in the schema, excluded ones included.
var HookProbes = []HookProbe{
	{{range $p := hookProbes}}
	{Table: {{quote $p.Table}}, Excluded: {{$p.Excluded}}, HasHooks: func() bool {
		return len(models.{{$p.ViewVar}}.SelectQueryHooks.GetHooks()) > 0 ||
			len(models.{{$p.ViewVar}}.AfterSelectHooks.GetHooks()) > 0
	}},
	{{end -}}
}
