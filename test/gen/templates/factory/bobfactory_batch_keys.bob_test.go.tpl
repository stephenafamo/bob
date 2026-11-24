{{- if and (has "batch_children" $.TableNames) (has "batch_parents" $.TableNames) -}}
{{$.Importer.Import "context"}}
{{$.Importer.Import "slices"}}
{{$.Importer.Import "testing"}}
{{$.Importer.Import "github.com/stephenafamo/bob"}}
{{$.Importer.Import "github.com/stephenafamo/scan"}}
{{$.Importer.Import (printf "github.com/stephenafamo/bob/dialect/%s/sm" $.Dialect)}}
{{$.Importer.Import (printf "github.com/stephenafamo/bob/dialect/%s/dialect" $.Dialect)}}
{{$.Importer.Import "models" (index $.OutputPackages "models")}}

type batchReads struct { bob.Executor; n, argCount int; sql string }
func (r *batchReads) QueryContext(ctx context.Context, sql string, args ...any) (scan.Rows, error) {
	r.n++
	r.sql = sql
	r.argCount = len(args)
	return r.Executor.QueryContext(ctx, sql, args...)
}

func TestBatchKeyOwnership(t *testing.T) {
	ctx := t.Context()
	tx, err := testDB.Begin(ctx)
	if err != nil { t.Fatal(err) }
	defer tx.Rollback(ctx)
	for _, sql := range []string{
		`INSERT INTO batch_parents VALUES (9007199254740992,'ABC'), (9007199254740993,'ABC')`,
		`INSERT INTO batch_children VALUES (1,9007199254740992,'abc'), (2,9007199254740992,'AbC'), (3,9007199254740993,'abc'), (4,9007199254740992,NULL)`,
	} {
		if _, err := tx.ExecContext(ctx, sql); err != nil { t.Fatal(err) }
	}

	db := &batchReads{Executor:tx}
	// These neighboring integers compare equal if converted to float64.
	const first, second = 9007199254740992, 9007199254740993
	parents := models.BatchParentSlice{
		{Tenant:first, Code:"abc"}, nil, {Tenant:second, Code:"ABC"}, {Tenant:first, Code:"ABC"}, {Tenant:3, Code:"missing"},
	}
	if err := parents.LoadCountChildren(ctx, db); err != nil { t.Fatal(err) }
	for i, want := range []int64{2,0,1,2,0} {
		if p := parents[i]; p != nil && (p.C.Children == nil || *p.C.Children != want) { t.Fatalf("parent %d: count %v, want %d", i, p.C.Children, want) }
	}
	if db.n != 1 { t.Fatalf("batch count used %d queries, want 1", db.n) }
	{{if eq $.Dialect "mysql" -}}
	{{$.Importer.Import "strings"}}
	if !strings.Contains(db.sql, "VALUES ROW(") {
		t.Fatalf("expected typed VALUES in batch count: %s", db.sql)
	}
	{{else if eq $.Dialect "psql" -}}
	{{$.Importer.Import "strings"}}
	if db.argCount != 2 || !strings.Contains(db.sql, "WITH ORDINALITY") {
		t.Fatalf("expected one array per key column and WITH ORDINALITY; got %d arguments: %s", db.argCount, db.sql)
	}
	{{end -}}

	// LIMIT is on children, not on matches. Both spellings of the first
	// parent's key must get the same two child instances, even with no FK
	// in the requested projection.
	if err := parents.LoadChildren(ctx, db, sm.Columns(models.BatchChildren.Columns.ID),
		sm.OrderBy(models.BatchChildren.Columns.ID), sm.Limit(2)); err != nil { t.Fatal(err) }
	for i, want := range []int{2,0,0,2,0} {
		if p := parents[i]; p != nil && len(p.R.Children) != want { t.Fatalf("parent %d: got %d children, want %d", i,len(p.R.Children),want) }
	}
	for i, child := range parents[0].R.Children {
		if int(child.ID) != i+1 || parents[3].R.Children[i] != child { t.Fatal("duplicate owners changed child order or identity") }
	}
	if db.n != 2 { t.Fatalf("batch load used %d queries, want 1", db.n-1) }

	children, err := models.BatchChildren.Query(sm.OrderBy(models.BatchChildren.Columns.ID)).All(ctx, tx)
	if err != nil { t.Fatal(err) }
	if err := children.LoadParent(ctx, db); err != nil { t.Fatal(err) }
	if len(children) != 4 || children[0].R.Parent == nil || children[0].R.Parent.Tenant != first ||
		children[1].R.Parent != children[0].R.Parent || children[2].R.Parent == nil ||
		children[2].R.Parent.Tenant != second || children[3].R.Parent != nil {
		t.Fatal("incorrect inverse ownership for case variants, composite keys or NULL")
	}
	if db.n != 3 { t.Fatalf("inverse batch load used %d queries, want 1", db.n-2) }
	t.Run("after-select hook reorders children", func(t *testing.T) {
		type reverse struct{}
		models.BatchChildren.AfterSelectHooks.AppendHooks(func(ctx context.Context, _ bob.Executor, rows models.BatchChildSlice) (context.Context, error) {
			if ctx.Value(reverse{}) == true { slices.Reverse(rows) }
			return ctx, nil
		})
		if err := parents.LoadChildren(context.WithValue(ctx, reverse{}, true), db,
			sm.Columns(models.BatchChildren.Columns.ID), sm.OrderBy(models.BatchChildren.Columns.ID)); err != nil { t.Fatal(err) }
		a, b, dup := parents[0].R.Children, parents[2].R.Children, parents[3].R.Children
		if len(a) != 2 || a[0].ID != 2 || a[1].ID != 1 || len(b) != 1 || b[0].ID != 3 || !slices.Equal(a, dup) {
			t.Fatalf("reordered children lost ownership: %v, %v, duplicate: %v", a, b, dup)
		}
	})
	t.Run("count after query hooks", func(t *testing.T) {
		type countCalls struct{}
		var calls int
		countCtx := context.WithValue(ctx, countCalls{}, &calls)
		models.BatchChildren.SelectQueryHooks.AppendHooks(func(ctx context.Context, _ bob.Executor, q *dialect.SelectQuery) (context.Context, error) {
			if n, ok := ctx.Value(countCalls{}).(*int); ok {
				(*n)++
				q.AppendWhere("id > 1")
				sm.OrderBy(models.BatchChildren.Columns.ID).Apply(q)
			}
			return ctx, nil
		})
		mods := []bob.Mod[*dialect.SelectQuery]{sm.Limit(1), sm.Offset(3),
			sm.Columns(models.BatchChildren.Columns.ID), sm.GroupBy(models.BatchChildren.Columns.ID)}
		before := db.n
		if err := parents.LoadCountChildren(countCtx, db, mods...); err != nil { t.Fatal(err) }
		if calls != 1 || db.n != before+1 { t.Fatalf("batch count: %d hook calls, %d queries", calls, db.n-before) }
		for i, want := range []int64{1,0,1,1,0} {
			p := parents[i]
			if p == nil { continue }
			got := *p.C.Children
			if err := p.LoadCountChildren(countCtx, db, mods...); err != nil { t.Fatal(err) }
			if got != want || *p.C.Children != want { t.Fatalf("parent %d: batch %d, single %d, want %d", i, got, *p.C.Children, want) }
		}
	})
	// A table select hook must filter counts just as it filters single loads.
	type hideChildren struct{}
	models.BatchChildren.SelectQueryHooks.AppendHooks(func(ctx context.Context, _ bob.Executor, q *dialect.SelectQuery) (context.Context, error) {
		if ctx.Value(hideChildren{}) == true { q.AppendWhere("1 = 0") }
		return ctx, nil
	})
	hidden := context.WithValue(ctx, hideChildren{}, true)
	if err := parents.LoadCountChildren(hidden, db); err != nil { t.Fatal(err) }
	for _, p := range parents {
		if p != nil && *p.C.Children != 0 { t.Fatal("batch count ignored the target's select hook") }
	}
	{{if eq $.Dialect "psql" -}}
	// CHAR without an explicit width truncates 'ABC' to 'A'. A slice
	// query must select the full key and must not duplicate rows for 'abc'.
	if _, err := tx.ExecContext(ctx, `INSERT INTO batch_parents VALUES (9007199254740992,'A');
		INSERT INTO batch_children VALUES (6,9007199254740992,'A')`); err != nil { t.Fatal(err) }
	rows, err := parents.Children(sm.OrderBy(models.BatchChildren.Columns.ID)).All(ctx, tx)
	if err != nil || len(rows) != 3 || rows[0].ID != 1 || rows[1].ID != 2 || rows[2].ID != 3 {
		t.Fatalf("slice filter returned the wrong children: %v, %v", rows, err)
	}
	if err := parents.LoadChildren(ctx, db, sm.ForUpdate()); err != nil { t.Fatal(err) }
	if len(parents[0].R.Children) != 2 { t.Fatal("locked batch load lost matches") }
	{{end -}}
	{{if eq $.Dialect "mysql" -}}
	if _, err := tx.ExecContext(ctx, `INSERT INTO batch_parents VALUES (18446744073709551615,'ABC')`); err != nil { t.Fatal(err) }
	if _, err := tx.ExecContext(ctx, `INSERT INTO batch_children VALUES (5,18446744073709551615,'abc')`); err != nil { t.Fatal(err) }
	maxKey := models.BatchParentSlice{ {Tenant:18446744073709551615, Code:"ABC"} }
	if err := maxKey.LoadChildren(ctx, db); err != nil { t.Fatal(err) }
	if len(maxKey[0].R.Children) != 1 || maxKey[0].R.Children[0].ID != 5 { t.Fatal("lost the maximum BIGINT UNSIGNED key") }
	{{end -}}
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
	// The array is the second key component: the whole tuple must use VALUES.
	if _, err := tx.ExecContext(ctx, `INSERT INTO batch_arrays VALUES (1,'{1}'), (1,'{1,2}'), (2,'{1}');
		INSERT INTO batch_items VALUES (11,1,'{1}'), (12,1,'{1,2}'), (21,2,'{1}'), (22,2,NULL)`); err != nil { t.Fatal(err) }
	parents, err := models.BatchArrays.Query().All(ctx, tx)
	if err != nil || len(parents) != 3 { t.Fatalf("parents: %v, %v", parents, err) }
	parents = models.BatchArraySlice{nil, parents[0], nil, parents[1], parents[2]}
	db := &batchReads{Executor:tx}
	rows, err := parents.Items().All(ctx, tx)
	if err != nil || len(rows) != 3 { t.Fatalf("array slice filter: %v, %v", rows, err) }
	if err := parents.LoadItems(ctx, db); err != nil { t.Fatal(err) }
	if err := parents.LoadCountItems(ctx, db); err != nil { t.Fatal(err) }
	for _, p := range parents {
		if p == nil { continue }
		if len(p.R.Items) != 1 || int(p.R.Items[0].ID) != 10 * int(p.Tenant) + len(p.ID) || *p.C.Items != 1 { t.Fatalf("wrong owners for tenant %d, array key %v", p.Tenant, p.ID) }
	}
	if db.n != 2 { t.Fatalf("load and count used %d queries, want 2", db.n) }
}
{{- end}}

{{- if has "batch_exact_parents" $.TableNames -}}
{{$.Importer.Import "testing"}}
{{$.Importer.Import "strings"}}
{{$.Importer.Import "github.com/shopspring/decimal"}}
{{$.Importer.Import "models" (index $.OutputPackages "models")}}
func TestBatchExactOwnership(t *testing.T) {
	ctx := t.Context()
	tx, err := testDB.Begin(ctx)
	if err != nil { t.Fatal(err) }
	defer tx.Rollback(ctx)
	// Each key differs from the first in just one component. The binary FK
	// has a different declared length from the referenced column.
	const insert = `INSERT INTO batch_exact_parents VALUES
		(1, REPEAT('9',65), CONCAT('0.',REPEAT('9',30)), X'00FF', '2020-01-01 01:02:03.123456'),
		(2, CONCAT(REPEAT('9',64),'8'), CONCAT('0.',REPEAT('9',30)), X'00FF', '2020-01-01 01:02:03.123456'),
		(3, REPEAT('9',65), CONCAT('0.',REPEAT('9',29),'8'), X'00FF', '2020-01-01 01:02:03.123456'),
		(4, REPEAT('9',65), CONCAT('0.',REPEAT('9',30)), X'00FE', '2020-01-01 01:02:03.123456'),
		(5, REPEAT('9',65), CONCAT('0.',REPEAT('9',30)), X'00FF', '2020-01-01 01:02:03.123455')`
	if _, err := tx.ExecContext(ctx, insert); err != nil { t.Fatal(err) }
	if _, err := tx.ExecContext(ctx, `INSERT INTO batch_exact_children SELECT * FROM batch_exact_parents`); err != nil { t.Fatal(err) }
	parents, err := models.BatchExactParents.Query().All(ctx, tx)
	if err != nil || len(parents) != 5 { t.Fatalf("parents: %v, %v", parents, err) }
	db := &batchReads{Executor:tx}
	if err := parents.LoadChildren(ctx, db); err != nil { t.Fatal(err) }
	if err := parents.LoadCountChildren(ctx, db); err != nil { t.Fatal(err) }
	if db.n != 2 || !strings.Contains(db.sql, "UNION ALL") { t.Fatalf("expected direct key probes; %d queries; last SQL: %s", db.n, db.sql) }
	for _, p := range parents {
		if len(p.R.Children) != 1 || p.R.Children[0].ID != p.ID || p.C.Children == nil || *p.C.Children != 1 {
			t.Fatalf("wrong matches for parent %d: children=%v count=%v", p.ID, p.R.Children, p.C.Children)
		}
	}
	// Predicates compare the bound values, not the values an INSERT would
	// store. Casting these inputs to DECIMAL(65,0) first can create matches.
	reference := *parents[0]
	fraction, overflow := reference, reference
	fraction.Amount = reference.Amount.Add(decimal.New(1, -1))
	overflow.Amount = decimal.RequireFromString("1" + strings.Repeat("0", 65))
	probes := models.BatchExactParentSlice{&reference, &fraction, &overflow}
	want := make([]models.BatchExactChildSlice, len(probes))
	for i, p := range probes {
		want[i], err = p.Children().All(ctx, tx)
		if err != nil { t.Fatal(err) }
	}
	if err := probes.LoadChildren(ctx, db); err != nil { t.Fatal(err) }
	if err := probes.LoadCountChildren(ctx, db); err != nil { t.Fatal(err) }
	for i, p := range probes {
		if len(p.R.Children) != len(want[i]) || *p.C.Children != int64(len(want[i])) {
			t.Fatalf("amount %s: direct rows %d, batch rows %d, batch count %d", p.Amount, len(want[i]), len(p.R.Children), *p.C.Children)
		}
		for j, row := range want[i] {
			if p.R.Children[j].ID != row.ID { t.Fatalf("amount %s: direct child %d, batch child %d", p.Amount, row.ID, p.R.Children[j].ID) }
		}
	}
}
{{- end}}
