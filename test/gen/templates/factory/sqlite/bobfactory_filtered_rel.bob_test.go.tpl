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
{{- end}}
