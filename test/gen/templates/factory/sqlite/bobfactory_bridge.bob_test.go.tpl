{{- if and (has "links" $.TableNames) (has "exporters" $.TableNames) (has "tasks" $.TableNames) (has "owners" $.TableNames) -}}
{{$.Importer.Import "testing"}}
func TestCreateExportersThroughLinks(t *testing.T) {
	ctx := t.Context()
	tx, err := testDB.Begin(ctx)
	if err != nil { t.Fatal(err) }
	defer tx.Rollback(ctx)
	f := New()
	owner := f.NewOwner().CreateOrFail(ctx, t, tx)
	backup := f.NewExporter().CreateOrFail(ctx, t, tx)
	f.AddBaseLinkMod(LinkMods.Note("custom bridge"), LinkMods.WithExistingOwner(owner), LinkMods.WithExistingBackupExporter(backup), LinkMods.WithNewExporter())
	task := f.NewTask(TaskMods.WithNewExporters(2)).CreateOrFail(ctx, t, tx)
	if len(task.R.Exporters) != 2 { t.Fatalf("expected 2 related exporters, got %d", len(task.R.Exporters)) }
	var tasks, exporters, links int
	if err := tx.QueryRowContext(ctx, `SELECT (SELECT count(*) FROM tasks), (SELECT count(*) FROM exporters), (SELECT count(*) FROM links)`).Scan(&tasks, &exporters, &links); err != nil {
		t.Fatal(err)
	}
	// Two requested exporters plus the separately configured backup.
	if tasks != 1 || exporters != 3 || links != 2 { t.Fatalf("tasks=%d exporters=%d links=%d; want 1, 3, 2", tasks, exporters, links) }
	for _, exporter := range task.R.Exporters {
		var note string
		if err := tx.QueryRowContext(ctx, "SELECT note FROM links WHERE task_id = ? AND exporter_id = ? AND backup_id = ? AND owner_id = ?", task.ID, exporter.ID, backup.ID, owner.ID).Scan(&note); err != nil { t.Fatal(err) }
		if note != "custom bridge" { t.Fatalf("lost bridge setting: %q", note) }
		if len(exporter.R.Tasks) != 1 || exporter.R.Tasks[0] != task { t.Fatal("missing inverse relationship") }
	}
}
{{- end}}
