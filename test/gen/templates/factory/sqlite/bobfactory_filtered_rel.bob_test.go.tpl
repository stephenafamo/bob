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
}
{{- end}}
