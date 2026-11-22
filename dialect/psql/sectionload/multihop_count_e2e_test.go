package sectionload

// DB-backed execution tests for a multi-hop (through) COUNT edge, driven
// through the public All() API. Reuses multihop_e2e_test.go's shared mh_*
// fixture (users -> user_roles -> roles), i.e. the schema that fixture creates.
//
// Fixture data recap (multihop_e2e_test.go's own comment, copied here for
// convenience — the actual CREATE TABLE/INSERT lives there):
//
//	(a) alice(1)  -> {admin(10), editor(11), editor(11)}  duplicate pair
//	(b) bob(2)    -> no mh_user_roles row at all       (child-less)
//	(c) carol(3)  -> role_id NULL (INNER JOIN drops it)    (NULL FK)
//	(d) user_id 999 in the join table matches no user      (orphan)

import (
	"context"
	"database/sql"
	"testing"
)

// e2eRolesCountEdge is the CountEdge counterpart of e2eRolesEdge
// (multihop_e2e_test.go): the SAME 2-hop chain (users -> user_roles -> roles),
// but counting rows instead of loading them. calls/counts are recorded by
// the caller-supplied setCount callback.
func e2eRolesCountEdge(setCount func(p *e2eUser, n int64) error) CountEdge[*e2eUser, *e2eRole] {
	return CountEdge[*e2eUser, *e2eRole]{
		RelName: "RolesCount",
		Sides: []Side{
			{FromTable: "mh_users", FromCols: []string{"id"}, ToTable: "mh_user_roles", ToCols: []string{"user_id"}},
			{FromTable: "mh_user_roles", FromCols: []string{"role_id"}, ToTable: "mh_roles", ToCols: []string{"id"}},
		},
		Child:         e2eRoleParts(),
		KeyCols:       []Column{{Name: "user_id", DBType: "int8"}},
		NewCarriedBuf: func() Buf[*e2eUser] { return &e2eUserBuf{} },
		ParentKey:     func(p *e2eUser) []any { return []any{p.ID} },
		ChildKey:      func(c *e2eRole) []any { return []any{c.ID} },
		SetCount:      setCount,
	}
}

// TestMultiHopCountE2E_TwoHopCount_TopLevel: a top-level multi-hop count's DB
// execution result must match a manual GROUP BY of the same fixture data,
// including two explicit contract points:
//
//   - bob (child-less parent) gets SetCount called with 0, NOT
//     skipped — the raw CountEdge API always calls SetCount for every
//     parent (parts.go's own doc comment: "EVERY parent gets a non-nil
//     *int64, zero included"); at this level that means "SetCount is
//     invoked", which generated code then turns into a non-nil *int64 by
//     always assigning &count.
//   - multi-hop count sums JOIN-ed ROWS, not DISTINCT children — so
//     alice, who has the duplicate (1,11) pair, must count 3 (admin once +
//     editor twice), not 2.
func TestMultiHopCountE2E_TwoHopCount_TopLevel(t *testing.T) {
	tx, cleanup := e2eFixture(t)
	defer cleanup()
	ctx := context.Background()

	calls := map[int64]int{}
	counts := map[int64]int64{}
	edge := e2eRolesCountEdge(func(p *e2eUser, n int64) error {
		calls[p.ID]++
		counts[p.ID] = n
		return nil
	})

	users, err := All(ctx, tx, e2eUsersQuery(), e2eUserParts(), NewCountNode(edge))
	if err != nil {
		t.Fatalf("All: %v", err)
	}
	if len(users) != 3 {
		t.Fatalf("got %d users, want 3", len(users))
	}

	// Every one of the 3 users — including bob, who has zero role
	// rows — must get SetCount called exactly once.
	for _, id := range []int64{1, 2, 3} {
		if calls[id] != 1 {
			t.Fatalf("SetCount called %d times for user %d, want 1 (every parent gets a count, zero included)", calls[id], id)
		}
	}
	// alice has admin once + editor twice (duplicate join-table
	// pair) = 3 JOIN-ed rows, not 2 distinct role ids.
	if got, want := counts[1], int64(3); got != want {
		t.Fatalf("alice's RolesCount = %d, want %d (JOIN row count, not DISTINCT)", got, want)
	}
	// bob has no mh_user_roles row at all — count 0, not skipped.
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

// TestMultiHopCountE2E_TwoHopCount_MatchesRowLoadLen: a multi-hop count and a
// multi-hop ROW load (e2eRolesEdge, already exercised by
// TestMultiHopE2E_TwoHopToMany) must agree: for every parent, RolesCount() ==
// len(RolesRowLoad()). This is the same style of cross-check the other multi-hop
// row-load tests use for row-vs-row parity, applied here to count-vs-row.
func TestMultiHopCountE2E_TwoHopCount_MatchesRowLoadLen(t *testing.T) {
	tx, cleanup := e2eFixture(t)
	defer cleanup()
	ctx := context.Background()

	// Row load: full multi-hop Edge, len() per parent.
	rowLens := map[int64]int{}
	rowEdge := e2eRolesEdge(func(p *e2eUser, cs []*e2eRole) error {
		rowLens[p.ID] = len(cs)
		return nil
	})
	if _, err := All(ctx, tx, e2eUsersQuery(), e2eUserParts(), NewNode(rowEdge)); err != nil {
		t.Fatalf("All (row load): %v", err)
	}

	// Count: same 2-hop chain, CountEdge.
	counts := map[int64]int64{}
	countEdge := e2eRolesCountEdge(func(p *e2eUser, n int64) error {
		counts[p.ID] = n
		return nil
	})
	if _, err := All(ctx, tx, e2eUsersQuery(), e2eUserParts(), NewCountNode(countEdge)); err != nil {
		t.Fatalf("All (count): %v", err)
	}

	for _, id := range []int64{1, 2, 3} {
		if got, want := counts[id], int64(rowLens[id]); got != want {
			t.Fatalf("user %d: RolesCount() = %d, len(row load) = %d — must match", id, got, want)
		}
	}
}

// --- nested multi-hop count ---------------------------------------

// e2eRoleTagRatingsCountEdge is a SECOND multi-hop chain, rooted at
// *e2eRole (NOT the island's true root *e2eUser): roles -> role_tags ->
// tag_ratings. Used to nest a multi-hop count UNDER a row section
// (e2eRolesEdge's own children), so its parent key must be read from
// that row SECTION's own scan buffer via CountEdge.parentKeyFromBufAny
// (exec.go's section branch) — a different code path than the
// island-root branch the top-level test exercises.
type e2eTagRating struct {
	Tag    string
	Rating int64
}
type e2eTagRatingBuf struct {
	tag    sql.NullString
	rating sql.NullInt64
}

func (b *e2eTagRatingBuf) TargetsFor(cols []Column) []any { return []any{&b.tag, &b.rating} }
func (b *e2eTagRatingBuf) BuildFor(modelCols []Column) *e2eTagRating {
	return &e2eTagRating{Tag: b.tag.String, Rating: b.rating.Int64}
}

// e2eRoleCarriedBuf is a DEDICATED carried-key buffer for
// e2eRoleTagRatingsCountEdge's NewCarriedBuf, distinct from e2eRoleBuf
// (multihop_e2e_test.go). e2eRoleBuf.TargetsFor is deliberately
// cols-ignoring (always 2 targets: id, label — it is only ever used for a
// full-row scan of mh_roles elsewhere in this package), so reusing it
// here as a 1-column carried-key buffer would silently mismatch the
// single-target carriedScanCols() call, exactly the failure mode
// e2eUserBuf's own doc comment (multihop_e2e_test.go) already warns about and
// was in fact hit once while writing this file ("expected 9 destination
// arguments in Scan, not 10").
type e2eRoleCarriedBuf struct {
	id sql.NullInt64
}

func (b *e2eRoleCarriedBuf) TargetsFor(cols []Column) []any { return []any{&b.id} }
func (b *e2eRoleCarriedBuf) BuildFor(modelCols []Column) *e2eRole {
	return &e2eRole{ID: b.id.Int64}
}

func e2eRoleTagRatingsCountEdge(setCount func(p *e2eRole, n int64) error) CountEdge[*e2eRole, *e2eTagRating] {
	return CountEdge[*e2eRole, *e2eTagRating]{
		RelName: "TagRatingsCount",
		Sides: []Side{
			{FromTable: "mh_roles", FromCols: []string{"id"}, ToTable: "mh_role_tags", ToCols: []string{"role_id"}},
			{FromTable: "mh_role_tags", FromCols: []string{"tag"}, ToTable: "mh_tag_ratings", ToCols: []string{"tag"}},
		},
		Child: TableParts[*e2eTagRating]{
			Table:   "mh_tag_ratings",
			Columns: []Column{{Name: "tag", DBType: "text"}, {Name: "rating", DBType: "int8"}},
			NewBuf:  func() Buf[*e2eTagRating] { return &e2eTagRatingBuf{} },
		},
		KeyCols:       []Column{{Name: "role_id", DBType: "int8"}},
		NewCarriedBuf: func() Buf[*e2eRole] { return &e2eRoleCarriedBuf{} },
		ParentKey:     func(p *e2eRole) []any { return []any{p.ID} },
		ChildKey:      func(c *e2eTagRating) []any { return []any{c.Rating} },
		SetCount:      setCount,
	}
}

// TestMultiHopCountE2E_NestedMultiHopCount_MatchesManualGroupBy: a multi-hop
// count nested under a row section (e2eRolesEdge's role children) — the
// exec.go section branch (parentKeyFromBufAny reading the SECTION's own
// scan buffer, not the root's) — must match a manual GROUP BY of the same
// data. Every role ROW alice's row section produces (admin, editor,
// editor — the duplicate join-table pair yields two separate editor
// section rows) gets its own TagRatingsCount invocation,
// once per row, not deduplicated by role id.
func TestMultiHopCountE2E_NestedMultiHopCount_MatchesManualGroupBy(t *testing.T) {
	tx, cleanup := e2eFixture(t)
	defer cleanup()
	ctx := context.Background()

	extra := []string{
		`CREATE TABLE mh_role_tags (role_id bigint, tag text NOT NULL)`,
		`CREATE TABLE mh_tag_ratings (tag text NOT NULL, rating bigint NOT NULL)`,
		`INSERT INTO mh_role_tags VALUES (10, 'built-in'), (11, 'built-in')`,
		// Two ratings for "built-in" -> every role tagged "built-in" gets
		// TagRatingsCount == 2.
		`INSERT INTO mh_tag_ratings VALUES ('built-in', 5), ('built-in', 4)`,
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
	countEdge := e2eRoleTagRatingsCountEdge(func(p *e2eRole, n int64) error {
		calls = append(calls, call{roleID: p.ID, count: n})
		return nil
	})

	rowEdge := e2eRolesEdge(func(p *e2eUser, cs []*e2eRole) error { return nil })

	_, err := All(ctx, tx, e2eUsersQuery(), e2eUserParts(), NewNode(rowEdge, NewCountNode(countEdge)))
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
