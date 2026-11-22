{{- if and (has "videos" $.TableNames) (has "users" $.TableNames) -}}
{{$.Importer.Import "context"}}
{{$.Importer.Import "errors"}}
{{$.Importer.Import "slices"}}
{{$.Importer.Import "testing"}}
{{$.Importer.Import "github.com/aarondl/opt/omit"}}
{{$.Importer.Import "github.com/stephenafamo/bob"}}
{{$.Importer.Import "models" (index $.OutputPackages "models")}}

func TestSingleModelWriteHooks(t *testing.T) {
	ctx := t.Context()
	tx, err := testDB.Begin(ctx)
	if err != nil { t.Fatal(err) }
	defer tx.Rollback(ctx)
	v := New().NewVideo().CreateOrFail(ctx, t, tx)
	user := New().NewUser().CreateOrFail(ctx, t, tx)
	set := &models.VideoSetter{UserID: omit.From(user.ID)}
	type key struct{ id int }
	k := &key{}
	type trace struct {
		calls []string
		veto error
	}
	hook := func(name string, before bool) bob.Hook[models.VideoSlice] {
		return func(ctx context.Context, _ bob.Executor, rows models.VideoSlice) (context.Context, error) {
			if s, ok := ctx.Value(k).(*trace); ok {
				s.calls = append(s.calls, name)
				if len(rows) != 1 { return ctx, errors.New("expected one video") }
				if before { return ctx, s.veto }
				if name == "after update" {
					if rows[0].UserID != user.ID { return ctx, errors.New("after update saw stale columns") }
					{{if eq $.Dialect "mysql"}}
					if rows[0] != v { return ctx, errors.New("MySQL hook must receive the original model") }
					{{else}}
					if rows[0] == v { return ctx, errors.New("RETURNING hook must receive the returned model") }
					{{end}}
				}
			}
			return ctx, nil
		}
	}
	models.Videos.BeforeUpdateHooks.AppendHooks(hook("before update", true))
	models.Videos.AfterUpdateHooks.AppendHooks(hook("after update", false))
	models.Videos.BeforeDeleteHooks.AppendHooks(hook("before delete", true))
	models.Videos.AfterDeleteHooks.AppendHooks(hook("after delete", false))

	s := &trace{veto: errors.New("stop")}
	ctx = context.WithValue(ctx, k, s)
	if err := v.Update(ctx, tx, set); !errors.Is(err, s.veto) { t.Fatalf("update: %v", err) }
	if err := v.Delete(ctx, tx); !errors.Is(err, s.veto) { t.Fatalf("delete: %v", err) }
	if err := v.Reload(ctx, tx); err != nil || v.UserID == user.ID { t.Fatalf("a vetoed write changed the row: %v", err) }
	s.veto = nil
	if err := v.Update(ctx, tx, set); err != nil { t.Fatal(err) }
	if err := v.Delete(ctx, tx); err != nil { t.Fatal(err) }
	want := []string{"before update", "before delete", "before update", "after update", "before delete", "after delete"}
	if !slices.Equal(s.calls, want) { t.Fatalf("hooks: %v; want %v", s.calls, want) }
	{{if ne $.Dialect "mysql"}}
	{{$.Importer.Import "database/sql"}}
	if err := v.Update(ctx, tx, set); !errors.Is(err, sql.ErrNoRows) { t.Fatalf("update after delete: %v", err) }
	{{end}}
}
{{- end}}
