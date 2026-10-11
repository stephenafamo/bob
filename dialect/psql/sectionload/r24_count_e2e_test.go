package sectionload

// R-24 (P-34-multihop-count.md) §8 T-8/T-9/T-12: DB-backed execution
// tests for a multi-hop (through) COUNT edge, driven through the public
// All() API. Reuses p05_e2e_test.go's shared p05e2e_* fixture (users ->
// user_roles -> roles), the same fixture T-8/T-9/T-12 were designed
// against (issue §8.1-L2, corrected by Opus review I-3: the fixture-schema
// file testdata/p05_fixture_schema.sql is for L3 generation only, not for
// this DB-execution test).
//
// Fixture data recap (p05_e2e_test.go's own comment, copied here for
// convenience — the actual CREATE TABLE/INSERT lives there):
//
//	(a) alice(1)  -> {admin(10), editor(11), editor(11)}  duplicate pair
//	(b) bob(2)    -> no p05e2e_user_roles row at all       (child-less)
//	(c) carol(3)  -> role_id NULL (INNER JOIN drops it)    (NULL FK)
//	(d) user_id 999 in the join table matches no user      (orphan)

import (
	"context"
	"database/sql"
	"testing"
)

// p05eRolesCountEdge is the CountEdge counterpart of p05eRolesEdge
// (p05_e2e_test.go): the SAME 2-hop chain (users -> user_roles -> roles),
// but counting rows instead of loading them. calls/counts are recorded by
// the caller-supplied setCount callback.
func p05eRolesCountEdge(setCount func(p *p05eUser, n int64) error) CountEdge[*p05eUser, *p05eRole] {
	return CountEdge[*p05eUser, *p05eRole]{
		RelName: "RolesCount",
		Sides: []Side{
			{FromTable: "p05e2e_users", FromCols: []string{"id"}, ToTable: "p05e2e_user_roles", ToCols: []string{"user_id"}},
			{FromTable: "p05e2e_user_roles", FromCols: []string{"role_id"}, ToTable: "p05e2e_roles", ToCols: []string{"id"}},
		},
		Child:         p05eRoleParts(),
		KeyCols:       []Column{{Name: "user_id", DBType: "int8"}},
		NewCarriedBuf: func() Buf[*p05eUser] { return &p05eUserBuf{} },
		ParentKey:     func(p *p05eUser) []any { return []any{p.ID} },
		ChildKey:      func(c *p05eRole) []any { return []any{c.ID} },
		SetCount:      setCount,
	}
}

// TestR24E2E_TwoHopCount_TopLevel (T-8): a top-level multi-hop count's DB
// execution result must match a manual GROUP BY of the same fixture data,
// including the two contract points I-3/I-4 flag explicitly:
//
//   - I-3: bob (child-less parent) gets SetCount called with 0, NOT
//     skipped — the raw CountEdge API always calls SetCount for every
//     parent (parts.go's own doc comment: "EVERY parent gets a non-nil
//     *int64, zero included"); at this level that means "SetCount is
//     invoked", which the generated dbsections layer then turns into a
//     non-nil *int64 by always assigning &count.
//   - I-4: multi-hop count sums JOIN-ed ROWS, not DISTINCT children — so
//     alice, who has the duplicate (1,11) pair, must count 3 (admin once +
//     editor twice), not 2.
func TestR24E2E_TwoHopCount_TopLevel(t *testing.T) {
	tx, cleanup := p05eFixture(t)
	defer cleanup()
	ctx := context.Background()

	calls := map[int64]int{}
	counts := map[int64]int64{}
	edge := p05eRolesCountEdge(func(p *p05eUser, n int64) error {
		calls[p.ID]++
		counts[p.ID] = n
		return nil
	})

	users, err := All(ctx, tx, p05eUsersQuery(), p05eUserParts(), NewCountNode(edge))
	if err != nil {
		t.Fatalf("All: %v", err)
	}
	if len(users) != 3 {
		t.Fatalf("got %d users, want 3", len(users))
	}

	// I-3: every one of the 3 users — including bob, who has zero role
	// rows — must get SetCount called exactly once.
	for _, id := range []int64{1, 2, 3} {
		if calls[id] != 1 {
			t.Fatalf("SetCount called %d times for user %d, want 1 (I-3: every parent gets a count, zero included)", calls[id], id)
		}
	}
	// I-4: alice has admin once + editor twice (duplicate join-table
	// pair) = 3 JOIN-ed rows, not 2 distinct role ids.
	if got, want := counts[1], int64(3); got != want {
		t.Fatalf("alice's RolesCount = %d, want %d (I-4: JOIN row count, not DISTINCT)", got, want)
	}
	// I-3: bob has no p05e2e_user_roles row at all — count 0, not skipped.
	if got, want := counts[2], int64(0); got != want {
		t.Fatalf("bob's RolesCount = %d, want %d (&0, child-less parent)", got, want)
	}
	// carol's join-table row has role_id NULL — the INNER JOIN to roles
	// drops it, so carol also ends up at 0, same as bob but via a
	// different fixture shape (NULL intermediate FK, not "no row at all").
	if got, want := counts[3], int64(0); got != want {
		t.Fatalf("carol's RolesCount = %d, want %d (NULL intermediate FK)", got, want)
	}
}

// TestR24E2E_TwoHopCount_MatchesRowLoadLen (T-9): a multi-hop count and a
// multi-hop ROW load (p05eRolesEdge, already exercised by
// TestP05E2E_TwoHopToMany) must agree: for every parent, RolesCount() ==
// len(RolesRowLoad()). This is the same style of cross-check P-05's own
// L2 task used for row-vs-row parity, applied here to count-vs-row.
func TestR24E2E_TwoHopCount_MatchesRowLoadLen(t *testing.T) {
	tx, cleanup := p05eFixture(t)
	defer cleanup()
	ctx := context.Background()

	// Row load: full multi-hop Edge, len() per parent.
	rowLens := map[int64]int{}
	rowEdge := p05eRolesEdge(func(p *p05eUser, cs []*p05eRole) error {
		rowLens[p.ID] = len(cs)
		return nil
	})
	if _, err := All(ctx, tx, p05eUsersQuery(), p05eUserParts(), NewNode(rowEdge)); err != nil {
		t.Fatalf("All (row load): %v", err)
	}

	// Count: same 2-hop chain, CountEdge.
	counts := map[int64]int64{}
	countEdge := p05eRolesCountEdge(func(p *p05eUser, n int64) error {
		counts[p.ID] = n
		return nil
	})
	if _, err := All(ctx, tx, p05eUsersQuery(), p05eUserParts(), NewCountNode(countEdge)); err != nil {
		t.Fatalf("All (count): %v", err)
	}

	for _, id := range []int64{1, 2, 3} {
		if got, want := counts[id], int64(rowLens[id]); got != want {
			t.Fatalf("user %d: RolesCount() = %d, len(row load) = %d — must match", id, got, want)
		}
	}
}

// --- T-12: nested multi-hop count -----------------------------------

// p05eRoleTagRatingsCountEdge is a SECOND multi-hop chain, rooted at
// *p05eRole (NOT the island's true root *p05eUser): roles -> role_tags ->
// tag_ratings. Used to nest a multi-hop count UNDER a row section
// (p05eRolesEdge's own children), so its parent key must be read from
// that row SECTION's own scan buffer via CountEdge.parentKeyFromBufAny
// (exec.go's section branch, ~line 418) — a different code path than the
// island-root branch T-8 exercises (~line 377).
type p05eTagRating struct {
	Tag    string
	Rating int64
}
type p05eTagRatingBuf struct {
	tag    sql.NullString
	rating sql.NullInt64
}

func (b *p05eTagRatingBuf) TargetsFor(cols []Column) []any { return []any{&b.tag, &b.rating} }
func (b *p05eTagRatingBuf) BuildFor(modelCols []Column) *p05eTagRating {
	return &p05eTagRating{Tag: b.tag.String, Rating: b.rating.Int64}
}

// p05eRoleCarriedBuf is a DEDICATED carried-key buffer for
// p05eRoleTagRatingsCountEdge's NewCarriedBuf, distinct from p05eRoleBuf
// (p05_e2e_test.go). p05eRoleBuf.TargetsFor is deliberately
// cols-ignoring (always 2 targets: id, label — it is only ever used for a
// full-row scan of p05e2e_roles elsewhere in this package), so reusing it
// here as a 1-column carried-key buffer would silently mismatch the
// single-target carriedScanCols() call, exactly the failure mode
// p05eUserBuf's own doc comment (p05_e2e_test.go) already warns about and
// was in fact hit once while writing this file ("expected 9 destination
// arguments in Scan, not 10").
type p05eRoleCarriedBuf struct {
	id sql.NullInt64
}

func (b *p05eRoleCarriedBuf) TargetsFor(cols []Column) []any { return []any{&b.id} }
func (b *p05eRoleCarriedBuf) BuildFor(modelCols []Column) *p05eRole {
	return &p05eRole{ID: b.id.Int64}
}

func p05eRoleTagRatingsCountEdge(setCount func(p *p05eRole, n int64) error) CountEdge[*p05eRole, *p05eTagRating] {
	return CountEdge[*p05eRole, *p05eTagRating]{
		RelName: "TagRatingsCount",
		Sides: []Side{
			{FromTable: "p05e2e_roles", FromCols: []string{"id"}, ToTable: "p05e2e_role_tags", ToCols: []string{"role_id"}},
			{FromTable: "p05e2e_role_tags", FromCols: []string{"tag"}, ToTable: "p05e2e_tag_ratings", ToCols: []string{"tag"}},
		},
		Child: TableParts[*p05eTagRating]{
			Table:   "p05e2e_tag_ratings",
			Columns: []Column{{Name: "tag", DBType: "text"}, {Name: "rating", DBType: "int8"}},
			NewBuf:  func() Buf[*p05eTagRating] { return &p05eTagRatingBuf{} },
		},
		KeyCols:       []Column{{Name: "role_id", DBType: "int8"}},
		NewCarriedBuf: func() Buf[*p05eRole] { return &p05eRoleCarriedBuf{} },
		ParentKey:     func(p *p05eRole) []any { return []any{p.ID} },
		ChildKey:      func(c *p05eTagRating) []any { return []any{c.Rating} },
		SetCount:      setCount,
	}
}

// TestR24E2E_NestedMultiHopCount_MatchesManualGroupBy (T-12): a multi-hop
// count nested under a row section (p05eRolesEdge's role children) — the
// exec.go section branch (parentKeyFromBufAny reading the SECTION's own
// scan buffer, not the root's) — must match a manual GROUP BY of the same
// data. Every role ROW alice's row section produces (admin, editor,
// editor — the duplicate join-table pair yields two separate editor
// section rows, P-05 §13.5-6) gets its own TagRatingsCount invocation,
// once per row, not deduplicated by role id.
func TestR24E2E_NestedMultiHopCount_MatchesManualGroupBy(t *testing.T) {
	tx, cleanup := p05eFixture(t)
	defer cleanup()
	ctx := context.Background()

	extra := []string{
		`CREATE TABLE p05e2e_role_tags (role_id bigint, tag text NOT NULL)`,
		`CREATE TABLE p05e2e_tag_ratings (tag text NOT NULL, rating bigint NOT NULL)`,
		`INSERT INTO p05e2e_role_tags VALUES (10, 'built-in'), (11, 'built-in')`,
		// Two ratings for "built-in" -> every role tagged "built-in" gets
		// TagRatingsCount == 2.
		`INSERT INTO p05e2e_tag_ratings VALUES ('built-in', 5), ('built-in', 4)`,
	}
	for _, stmt := range extra {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			t.Fatalf("fixture statement failed:\n%s\nerror: %v", stmt, err)
		}
	}

	type call struct {
		roleID int64
		count  int64
	}
	var calls []call
	countEdge := p05eRoleTagRatingsCountEdge(func(p *p05eRole, n int64) error {
		calls = append(calls, call{roleID: p.ID, count: n})
		return nil
	})

	rowEdge := p05eRolesEdge(func(p *p05eUser, cs []*p05eRole) error { return nil })

	_, err := All(ctx, tx, p05eUsersQuery(), p05eUserParts(), NewNode(rowEdge, NewCountNode(countEdge)))
	if err != nil {
		t.Fatalf("All (nested count): %v", err)
	}

	// alice's row section produces 3 role rows: admin(10), editor(11),
	// editor(11) again (the duplicate join-table pair) — each is its own
	// section row, so the nested count fires 3 times, all with count==2
	// (both admin and editor are tagged "built-in", which has 2 ratings).
	if len(calls) != 3 {
		t.Fatalf("got %d nested count invocations, want 3 (one per alice role ROW, duplicates included): %+v", len(calls), calls)
	}
	for _, c := range calls {
		if c.count != 2 {
			t.Fatalf("role %d: TagRatingsCount = %d, want 2 (manual GROUP BY: 'built-in' has 2 ratings)", c.roleID, c.count)
		}
		if c.roleID != 10 && c.roleID != 11 {
			t.Fatalf("unexpected role id %d in nested count calls: %+v", c.roleID, calls)
		}
	}
}
