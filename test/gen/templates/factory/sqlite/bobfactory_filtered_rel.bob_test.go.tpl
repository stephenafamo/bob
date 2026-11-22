{{- if and (has "locations" $.TableNames) (has "location_versions" $.TableNames) -}}
{{$.Importer.Import "testing"}}
{{$.Importer.Import "models" (index $.OutputPackages "models")}}
func TestInsertPrimaryVersion(t *testing.T) {
	ctx := t.Context()
	tx, err := testDB.Begin(ctx)
	if err != nil { t.Fatal(err) }
	defer tx.Rollback(ctx)
	loc, err := models.Locations.Insert(&models.LocationSetter{}).One(ctx, tx)
	if err != nil { t.Fatal(err) }
	if _, err := models.LocationVersions.Insert(&models.LocationVersionSetter{LocationID: &loc.ID}).One(ctx, tx); err != nil { t.Fatal(err) }
	if err := loc.InsertPrimaryVersion(ctx, tx, &models.LocationVersionSetter{LocationID: &loc.ID}); err != nil { t.Fatal(err) }
	if loc.R.PrimaryVersion == nil || !loc.R.PrimaryVersion.IsPrimaryVersion { t.Fatal("primary version not set") }
	if err := loc.LoadLocationVersions(ctx, tx); err != nil { t.Fatal(err) }
	if len(loc.R.LocationVersions) != 2 { t.Fatalf("got %d versions, want both", len(loc.R.LocationVersions)) }
	primary, err := loc.PrimaryVersion().One(ctx, tx)
	if err != nil { t.Fatal(err) }
	if primary.ID != loc.R.PrimaryVersion.ID { t.Fatalf("stored primary ID %d, want %d", primary.ID, loc.R.PrimaryVersion.ID) }
	parent, err := primary.PrimaryVersionFor().One(ctx, tx)
	if err != nil || parent.ID != loc.ID { t.Fatalf("inverse: %v, %v", parent, err) }
	if err := loc.R.LocationVersions.LoadPrimaryVersionFor(ctx, tx); err != nil { t.Fatal(err) }
	for _, version := range loc.R.LocationVersions {
		if (version.R.PrimaryVersionFor != nil) != version.IsPrimaryVersion { t.Fatalf("wrong inverse on version %d", version.ID) }
	}
	loaded, err := models.Locations.Query(models.Preload.Location.PrimaryVersion()).One(ctx, tx)
	if err != nil { t.Fatal(err) }
	if loaded.R.PrimaryVersion == nil || loaded.R.PrimaryVersion.ID != primary.ID { t.Fatal("preload did not return the primary version") }

	other, err := models.Locations.Insert(&models.LocationSetter{}).One(ctx, tx)
	if err != nil { t.Fatal(err) }
	if _, err := models.LocationVersions.Insert(&models.LocationVersionSetter{LocationID: &other.ID}).One(ctx, tx); err != nil { t.Fatal(err) }
	// Count and EXISTS must apply the same filter as the relationship loader.
	selected, err := models.Locations.Query(models.SelectWhere.Locations.R.HasPrimaryVersions(),
		models.PreloadCount.Location.PrimaryVersions()).All(ctx, tx)
	if err != nil { t.Fatal(err) }
	if len(selected) != 1 || selected[0].ID != loc.ID || selected[0].C.PrimaryVersions == nil || *selected[0].C.PrimaryVersions != 1 {
		t.Fatalf("filtered EXISTS or preloaded count: %+v", selected)
	}
	batch := models.LocationSlice{loc, nil, other}
	if err := batch.LoadCountPrimaryVersions(ctx, tx); err != nil { t.Fatal(err) }
	if *loc.C.PrimaryVersions != 1 || *other.C.PrimaryVersions != 0 { t.Fatal("wrong filtered batch counts") }

	// BLOB keys use SQL byte equality, not Go pointer equality. NULL never
	// matches NULL, and duplicate input parents both get the matching count.
	if _, err := tx.ExecContext(ctx, "INSERT INTO collections(id, code) VALUES (1, x'00'), (2, NULL)"); err != nil { t.Fatal(err) }
	if _, err := tx.ExecContext(ctx, "INSERT INTO entries(id, code) VALUES (1, x'00'), (2, NULL)"); err != nil { t.Fatal(err) }
	all, err := models.Collections.Query().All(ctx, tx)
	if err != nil { t.Fatal(err) }
	duplicate := *all[0]
	all = append(all, &duplicate)
	if err := all.LoadEntries(ctx, tx); err != nil { t.Fatal(err) }
	if err := all.LoadCountEntries(ctx, tx); err != nil { t.Fatal(err) }
	for _, c := range all {
		want := int64(0)
		if c.ID == 1 { want = 1 }
		if int64(len(c.R.Entries)) != want || c.C.Entries == nil || *c.C.Entries != want { t.Fatalf("collection %d: rows=%d count=%v", c.ID, len(c.R.Entries), c.C.Entries) }
	}
}

func TestRelationshipAssignment(t *testing.T) {
	ctx := t.Context()
	tx, err := testDB.Begin(ctx)
	if err != nil { t.Fatal(err) }
	defer tx.Rollback(ctx)
	loc, err := models.Locations.Insert(&models.LocationSetter{}).One(ctx, tx)
	if err != nil { t.Fatal(err) }
	old := &models.LocationVersion{}
	loc.R.LocationVersions = models.LocationVersionSlice{old}
	if err := loc.InsertLocationVersions(ctx, tx, &models.LocationVersionSetter{LocationID: &loc.ID}); err != nil { t.Fatal(err) }
	if len(loc.R.LocationVersions) != 2 || loc.R.LocationVersions[0] != old || loc.R.{{$.RelationLoadedName}}.LocationVersions {
		t.Fatal("insert must append without marking a partial result loaded")
	}
	if err := loc.LoadLocationVersions(ctx, tx); err != nil { t.Fatal(err) }
	if len(loc.R.LocationVersions) != 1 || loc.R.LocationVersions[0] == old || !loc.R.{{$.RelationLoadedName}}.LocationVersions {
		t.Fatal("load must replace the partial result")
	}
	child := loc.R.LocationVersions[0]
	{{if $.NoBackReferencing}}
	if child.R.Location != nil || child.R.{{$.RelationLoadedName}}.Location { t.Fatal("inverse populated with back-referencing disabled") }
	{{else}}
	if child.R.Location != loc || !child.R.{{$.RelationLoadedName}}.Location { t.Fatal("missing loaded inverse") }
	{{end}}
	if err := loc.Preload("LocationVersions", models.LocationVersionSlice{}); err != nil { t.Fatal(err) }
	if loc.R.LocationVersions == nil || len(loc.R.LocationVersions) != 0 || !loc.R.{{$.RelationLoadedName}}.LocationVersions {
		t.Fatal("preload must preserve a known empty result")
	}
	if err := loc.AttachLocationVersions(ctx, tx, child); err != nil { t.Fatal(err) }
	if len(loc.R.LocationVersions) != 1 || loc.R.LocationVersions[0] != child || !loc.R.{{$.RelationLoadedName}}.LocationVersions {
		t.Fatal("attach must append and preserve an existing loaded flag")
	}
}
{{- end}}

{{- if and (has "locations" $.TableNames) (has "teams" $.TableNames) -}}
{{$.Importer.Import "testing"}}
{{$.Importer.Import "models" (index $.OutputPackages "models")}}
func TestRepeatedTableCorrelation(t *testing.T) {
	ctx := t.Context()
	tx, err := testDB.Begin(ctx)
	if err != nil { t.Fatal(err) }
	defer tx.Rollback(ctx)
	if _, err := tx.ExecContext(ctx, "INSERT INTO locations (id) VALUES (1), (2)"); err != nil { t.Fatal(err) }
	if _, err := tx.ExecContext(ctx, `INSERT INTO teams (id, location_id, parent_id, active) VALUES
		(10, 1, NULL, false), (11, 1, 10, true), (12, 1, 10, true), (13, 1, 10, false),
		(20, 2, NULL, true), (21, 2, 20, false)`); err != nil { t.Fatal(err) }

	// Both subqueries join teams twice. The filter on teams must apply to the child.
	activeChild := models.SelectWhere.Teams.Active.EQ(true)
	locs, err := models.Locations.Query(models.SelectWhere.Locations.R.HasTeams(activeChild),
		models.PreloadCount.Location.Teams(activeChild)).All(ctx, tx)
	if err != nil { t.Fatal(err) }
	if len(locs) != 1 || locs[0].ID != 1 || locs[0].C.Teams == nil || *locs[0].C.Teams != 2 {
		t.Fatalf("expected only location 1 with 2 active child teams, got %+v", locs)
	}
	loc := locs[0]
	if err := loc.LoadTeams(ctx, tx, activeChild); err != nil { t.Fatal(err) }
	if len(loc.R.Teams) != 2 { t.Fatalf("direct load: got %d active child teams, want 2", len(loc.R.Teams)) }
	other := &models.Location{ID: 2}
	batch := models.LocationSlice{other, nil, loc}
	if err := batch.LoadTeams(ctx, tx, activeChild); err != nil { t.Fatal(err) }
	if len(loc.R.Teams) != 2 || len(other.R.Teams) != 0 { t.Fatal("batch attached child teams to the wrong location") }
	if err := batch.LoadCountTeams(ctx, tx, activeChild); err != nil { t.Fatal(err) }
	if *loc.C.Teams != 2 || *other.C.Teams != 0 { t.Fatal("wrong grouped counts for the repeated table") }

	// The bridge stores the integer parent key as nullable TEXT. The loader
	// must scan it as the parent's type before stitching the results.
	if _, err := tx.ExecContext(ctx, `INSERT INTO links VALUES ('1', 11), ('2', 21), (NULL, 12)`); err != nil { t.Fatal(err) }
	if err := batch.LoadLinkedTeams(ctx, tx); err != nil { t.Fatal(err) }
	if len(loc.R.LinkedTeams) != 1 || loc.R.LinkedTeams[0].ID != 11 ||
		len(other.R.LinkedTeams) != 1 || other.R.LinkedTeams[0].ID != 21 { t.Fatal("wrong teams for mixed-type bridge keys") }
}
{{- end}}

{{- if and (has "revisions" $.TableNames) (has "revision_labels" $.TableNames) -}}
{{$.Importer.Import "slices"}}
{{$.Importer.Import "testing"}}
{{$.Importer.Import "context"}}
{{$.Importer.Import "errors"}}
{{$.Importer.Import "github.com/stephenafamo/bob"}}
{{$.Importer.Import "models" (index $.OutputPackages "models")}}
func TestCompositeRowRefresh(t *testing.T) {
	ctx := t.Context()
	tx, err := testDB.Begin(ctx)
	if err != nil { t.Fatal(err) }
	defer tx.Rollback(ctx)
	pk := models.Revisions.PrimaryKey()
	if !slices.Equal(pk.Names(), []string{"code", "seq"}) || !slices.Equal(pk.Parent(), []string{"revisions"}) {
		t.Fatalf("primary key: got %v on %v, want (code, seq) on revisions", pk.Names(), pk.Parent())
	}
	if _, err := tx.ExecContext(ctx, "INSERT INTO revisions (code, seq, label) VALUES (x'01', 2, 'first'), (x'02', 2, 'second'), (x'02', 3, 'third')"); err != nil { t.Fatal(err) }
	label, err := models.FindRevision(ctx, tx, []byte{2}, 2, "label")
	if err != nil || label.Label != "second" || label.Code != nil { t.Fatalf("find with projection: %+v, %v", label, err) }
	exists, err := models.RevisionExists(ctx, tx, []byte{1}, 3)
	if err != nil || exists { t.Fatalf("partial key matched: %v, %v", exists, err) }
	a := &models.Revision{Code: []byte{1}, Seq: 2}
	b := &models.Revision{Code: []byte{2}, Seq: 2}
	missing := &models.Revision{Code: []byte{1}, Seq: 3, Label: "untouched"}
	duplicate := *a
	rows := models.RevisionSlice{b, a, missing, &duplicate}
	if err := rows.ReloadAll(ctx, tx); err != nil { t.Fatal(err) }
	if rows[0] != b || b.Label != "second" || a.Label != "first" || duplicate.Label != "first" || missing.Label != "untouched" {
		t.Fatalf("composite-key refresh: %q, %q, %q, %q", b.Label, a.Label, duplicate.Label, missing.Label)
	}
	b.Seq = 3
	if err := b.Reload(ctx, tx); err != nil || b.Label != "third" { t.Fatalf("reload: %+v, %v", b, err) }
}

func TestViewHookDispatch(t *testing.T) {
	type marker struct{}
	ctx := context.WithValue(t.Context(), marker{}, true)
	stop := errors.New("select hook")
	models.RevisionLabels.AfterSelectHooks.AppendHooks(func(ctx context.Context, _ bob.Executor, rows models.RevisionLabelSlice) (context.Context, error) {
		if ctx.Value(marker{}) == true && len(rows) == 1 { return ctx, stop }
		return ctx, nil
	})
	view := &models.RevisionLabel{}
	if err := view.AfterQueryHook(ctx, nil, bob.QueryTypeSelect); !errors.Is(err, stop) { t.Fatalf("select hook: %v", err) }
	for _, typ := range []bob.QueryType{bob.QueryTypeInsert, bob.QueryTypeUpdate, bob.QueryTypeDelete, bob.QueryTypeMerge, bob.QueryTypeUnknown} {
		if err := view.AfterQueryHook(ctx, nil, typ); err != nil { t.Fatalf("unexpected view hook for %v: %v", typ, err) }
	}
	if err := view.AfterQueryHook(bob.SkipModelHooks(ctx), nil, bob.QueryTypeSelect); err != nil { t.Fatalf("skipped select hook: %v", err) }
}
{{- end}}
