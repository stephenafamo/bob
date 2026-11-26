{{- if and (has "batch_children" $.TableNames) (has "batch_parents" $.TableNames) -}}
{{$.Importer.Import "context"}}
{{$.Importer.Import "slices"}}
{{$.Importer.Import "testing"}}
{{$.Importer.Import "github.com/stephenafamo/bob"}}
{{$.Importer.Import "github.com/stephenafamo/scan"}}
{{$.Importer.Import (printf "github.com/stephenafamo/bob/dialect/%s" $.Dialect)}}
{{$.Importer.Import (printf "github.com/stephenafamo/bob/dialect/%s/sm" $.Dialect)}}
{{$.Importer.Import (printf "github.com/stephenafamo/bob/dialect/%s/dialect" $.Dialect)}}
{{$.Importer.Import "models" (index $.OutputPackages "models")}}

type batchReads struct { bob.Executor; n int }
func (r *batchReads) QueryContext(ctx context.Context, sql string, args ...any) (scan.Rows, error) {
	r.n++
	return r.Executor.QueryContext(ctx, sql, args...)
}

func TestBatchKeyOwnership(t *testing.T) {
	ctx := t.Context()
	tx, err := testDB.Begin(ctx)
	if err != nil { t.Fatal(err) }
	defer tx.Rollback(ctx)
	for _, sql := range []string{
		`INSERT INTO batch_parents VALUES (9007199254740992,'ABC'), (9007199254740993,'ABC')`,
		`INSERT INTO batch_children VALUES (1,9007199254740992,'ABC'), (2,9007199254740992,'ABC'), (3,9007199254740993,'ABC'), (4,9007199254740992,NULL)`,
	} {
		if _, err := tx.ExecContext(ctx, sql); err != nil { t.Fatal(err) }
	}

	db := &batchReads{Executor:tx}
	// These neighboring integers compare equal if converted to float64.
	const first, second = 9007199254740992, 9007199254740993
	parents := models.BatchParentSlice{
		{Tenant:first, Code:"ABC"}, nil, {Tenant:second, Code:"ABC"}, {Tenant:first, Code:"ABC"}, {Tenant:3, Code:"missing"},
	}
	id := models.BatchChildren.Columns.ID

	t.Run("group by child primary key", func(t *testing.T) {
		before := db.n
		if err := parents.LoadChildren(ctx, db, sm.GroupBy(id), sm.OrderBy(id)); err != nil { t.Fatal(err) }
		a, b, dup := parents[0].R.Children, parents[2].R.Children, parents[3].R.Children
		if len(a) != 2 || a[0].ID != 1 || a[0].Tenant != first || a[1].ID != 2 || len(b) != 1 || b[0].ID != 3 ||
			!slices.Equal(a, dup) || len(parents[4].R.Children) != 0 || db.n != before+1 {
			t.Fatalf("grouped load: first=%v second=%v duplicate=%v missing=%v queries=%d", a, b, dup, parents[4].R.Children, db.n-before)
		}
	})

	t.Run("partial projection, pagination, and after-select order", func(t *testing.T) {
		type reverse struct{}
		models.BatchChildren.AfterSelectHooks.AppendHooks(func(ctx context.Context, _ bob.Executor, rows models.BatchChildSlice) (context.Context, error) {
			if ctx.Value(reverse{}) == true { slices.Reverse(rows) }
			return ctx, nil
		})
		before := db.n
		if err := parents.LoadChildren(context.WithValue(ctx, reverse{}, true), db,
			sm.Columns(id), sm.OrderBy(id), sm.Limit(2), sm.Offset(1)); err != nil { t.Fatal(err) }
		a, b, dup := parents[0].R.Children, parents[2].R.Children, parents[3].R.Children
		if len(a) != 1 || a[0].ID != 2 || a[0].Tenant != 0 || len(b) != 1 || b[0].ID != 3 ||
			!slices.Equal(a, dup) || len(parents[4].R.Children) != 0 || db.n != before+1 {
			t.Fatalf("paged load: first=%v second=%v duplicate=%v queries=%d", a, b, dup, db.n-before)
		}
	})

	t.Run("count after query hooks", func(t *testing.T) {
		type countCalls struct{}
		var calls int
		countCtx := context.WithValue(ctx, countCalls{}, &calls)
		models.BatchChildren.SelectQueryHooks.AppendHooks(func(ctx context.Context, _ bob.Executor, q {{$.Dialect}}.SelectEditor) (context.Context, error) {
			if n, ok := ctx.Value(countCalls{}).(*int); ok {
				(*n)++
				sm.Where(id.GT({{$.Dialect}}.Arg(1))).Edit(q)
				sm.OrderBy(id).Edit(q)
			}
			return ctx, nil
		})
		mods := []bob.Mod[dialect.SelectEditor]{sm.Limit(1), sm.Offset(3), sm.Columns(id), sm.GroupBy(id)}
		before := db.n
		if err := parents.LoadCountChildren(countCtx, db, mods...); err != nil { t.Fatal(err) }
		if calls != 1 || db.n != before+1 { t.Fatalf("batch count: %d hook calls, %d queries", calls, db.n-before) }
		for i, want := range []int64{1,0,1,1,0} {
			p := parents[i]
			if p == nil { continue }
			if p.C.Children == nil || *p.C.Children != want { t.Fatalf("parent %d: count %v, want %d", i, p.C.Children, want) }
		}
	})
}
{{- end}}

{{- if has "batch_arrays" $.TableNames -}}
{{$.Importer.Import "testing"}}
{{$.Importer.Import "models" (index $.OutputPackages "models")}}
func TestBatchArrayOwnership(t *testing.T) {
	ctx := t.Context()
	tx, err := testDB.Begin(ctx)
	if err != nil { t.Fatal(err) }
	defer tx.Rollback(ctx)
	if _, err := tx.ExecContext(ctx, `INSERT INTO batch_arrays VALUES (1,'{1}'), (1,'{1,2}'), (2,'{1}');
		INSERT INTO batch_items VALUES (11,1,'{1}'), (12,1,'{1,2}'), (21,2,'{1}'), (22,2,NULL)`); err != nil { t.Fatal(err) }
	parents, err := models.BatchArrays.Query().All(ctx, tx)
	if err != nil || len(parents) != 3 { t.Fatalf("parents: %v, %v", parents, err) }
	parents = models.BatchArraySlice{nil, parents[0], nil, parents[1], parents[2]}
	db := &batchReads{Executor:tx}
	if err := parents.LoadItems(ctx, db); err != nil { t.Fatal(err) }
	if err := parents.LoadCountItems(ctx, db); err != nil { t.Fatal(err) }
	for _, p := range parents {
		if p == nil { continue }
		if len(p.R.Items) != 1 || int(p.R.Items[0].ID) != 10 * int(p.Tenant) + len(p.ID) || *p.C.Items != 1 {
			t.Fatalf("wrong matches for tenant %d, array key %v", p.Tenant, p.ID)
		}
	}
	if db.n != 2 { t.Fatalf("load and count used %d queries, want 2", db.n) }
}
{{- end}}
