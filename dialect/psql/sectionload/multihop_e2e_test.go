package sectionload

// DB-backed execution test for a 2-hop (through) relationship, driven
// through the public All() API — a hand-written Edge with
// Sides/Carried/NewCarriedBuf, run against real PostgreSQL inside a
// transaction that is always rolled back (same template as
// schema_e2e_test.go: sql.Open -> PingContext -> t.Skipf if unreachable ->
// BeginTx -> CREATE TABLE fixture -> defer Rollback). The connection string
// is read from SECTIONLOAD_TEST_DSN; the tests are skipped when it is unset.
//
// Schema (mh_ prefix, created fresh inside the transaction):
//
//	mh_users(id, name)
//	mh_roles(id, label)
//	mh_user_roles(user_id, role_id)   -- join table, both columns NULLable
//	mh_profiles(id, bio)
//	mh_user_profile(user_id UNIQUE, profile_id) -- to-one join table
//
// Data deliberately covers these shapes:
//
//	(a) 1 parent, multiple children:      alice -> {admin, editor, editor}
//	(b) duplicate pair in the join table: alice/editor appears twice
//	(c) a join-table row matching no parent: user_id 999 (orphan)
//	(d) a child-less parent:               bob has no mh_user_roles row
//	(e) NULL intermediate FK:              carol's row has role_id NULL
//
// All five are exercised in ONE shared fixture, read by several
// subtests, to avoid repeating the CREATE TABLE/INSERT boilerplate.

import (
	"context"
	"database/sql"
	"os"
	"sort"
	"testing"

	_ "github.com/lib/pq"
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql"
)

// multiHopE2EDSN returns the connection string for the DB-backed tests from the
// SECTIONLOAD_TEST_DSN environment variable, skipping the test when unset.
func multiHopE2EDSN(t *testing.T) string {
	t.Helper()
	dsn := os.Getenv("SECTIONLOAD_TEST_DSN")
	if dsn == "" {
		t.Skip("SECTIONLOAD_TEST_DSN is not set")
	}
	return dsn
}

// --- model/buf fixtures ------------------------------------------------

type e2eUser struct {
	ID   int64
	Name string
}
type e2eUserBuf struct {
	id   sql.NullInt64
	name sql.NullString
}

// TargetsFor is column-aware (switches on cols[i].Name), NOT the simpler
// "ignore cols, always return every field" shape several other fixtures
// in this package use: this buffer type is shared between TWO different
// roles — the root's own full-row scan (root.scanCols() == [id, name])
// AND the carried-key scan for a multi-hop edge (carriedScanCols() ==
// [id] only, one column). A cols-ignoring TargetsFor would always return
// 2 targets regardless of which caller asked, silently mismatching the
// 1-target carried scan (this was caught, not hypothetical: the first
// run of this file's DB-backed tests failed with "expected N destination
// arguments in Scan, not N+1" until this was fixed).
func (b *e2eUserBuf) TargetsFor(cols []Column) []any {
	if len(cols) == 0 {
		return []any{&b.id, &b.name}
	}
	out := make([]any, len(cols))
	for i, c := range cols {
		switch c.Name {
		case "id":
			out[i] = &b.id
		case "name":
			out[i] = &b.name
		default:
			panic("e2eUserBuf: unknown column " + c.Name)
		}
	}
	return out
}

func (b *e2eUserBuf) BuildFor(modelCols []Column) *e2eUser {
	return &e2eUser{ID: b.id.Int64, Name: b.name.String}
}

type e2eRole struct {
	ID    int64
	Label string
}
type e2eRoleBuf struct {
	id    sql.NullInt64
	label sql.NullString
}

func (b *e2eRoleBuf) TargetsFor(cols []Column) []any { return []any{&b.id, &b.label} }
func (b *e2eRoleBuf) BuildFor(modelCols []Column) *e2eRole {
	return &e2eRole{ID: b.id.Int64, Label: b.label.String}
}

type e2eProfile struct {
	ID  int64
	Bio string
}
type e2eProfileBuf struct {
	id  sql.NullInt64
	bio sql.NullString
}

func (b *e2eProfileBuf) TargetsFor(cols []Column) []any { return []any{&b.id, &b.bio} }
func (b *e2eProfileBuf) BuildFor(modelCols []Column) *e2eProfile {
	return &e2eProfile{ID: b.id.Int64, Bio: b.bio.String}
}

// e2eUserRole is the intermediate table's own model — used ONLY by the
// parity check (single-hop-nested tree), never by the 2-hop edge itself.
type e2eUserRole struct {
	UserID int64
	RoleID sql.NullInt64
	Role   *e2eRole // populated by the nested single-hop Role edge below
}
type e2eUserRoleBuf struct {
	userID sql.NullInt64
	roleID sql.NullInt64
}

func (b *e2eUserRoleBuf) TargetsFor(cols []Column) []any { return []any{&b.userID, &b.roleID} }
func (b *e2eUserRoleBuf) BuildFor(modelCols []Column) *e2eUserRole {
	return &e2eUserRole{UserID: b.userID.Int64, RoleID: b.roleID}
}

func e2eUserParts() TableParts[*e2eUser] {
	return TableParts[*e2eUser]{
		Table:   "mh_users",
		Columns: []Column{{Name: "id", DBType: "int8"}, {Name: "name", DBType: "text"}},
		NewBuf:  func() Buf[*e2eUser] { return &e2eUserBuf{} },
	}
}

func e2eRoleParts() TableParts[*e2eRole] {
	return TableParts[*e2eRole]{
		Table:   "mh_roles",
		Columns: []Column{{Name: "id", DBType: "int8"}, {Name: "label", DBType: "text"}},
		NewBuf:  func() Buf[*e2eRole] { return &e2eRoleBuf{} },
	}
}

// e2eRolesEdge is the 2-hop edge under test: users -> user_roles -> roles.
func e2eRolesEdge(stitch func(p *e2eUser, cs []*e2eRole) error) Edge[*e2eUser, *e2eRole] {
	return Edge[*e2eUser, *e2eRole]{
		RelName: "Roles",
		Sides: []Side{
			{FromTable: "mh_users", FromCols: []string{"id"}, ToTable: "mh_user_roles", ToCols: []string{"user_id"}},
			{FromTable: "mh_user_roles", FromCols: []string{"role_id"}, ToTable: "mh_roles", ToCols: []string{"id"}},
		},
		ToMany:        true,
		Child:         e2eRoleParts(),
		Carried:       []Column{{Name: "_sl_k0", DBType: "int8"}},
		NewCarriedBuf: func() Buf[*e2eUser] { return &e2eUserBuf{} },
		ParentKey:     func(p *e2eUser) []any { return []any{p.ID} },
		ChildKey:      func(c *e2eRole) []any { return []any{c.ID} },
		Stitch:        stitch,
	}
}

// e2eProfileEdge is the to-one 2-hop edge: users -> user_profile ->
// profiles, with a UNIQUE(user_id) on the join table.
func e2eProfileEdge(stitch func(p *e2eUser, cs []*e2eProfile) error) Edge[*e2eUser, *e2eProfile] {
	return Edge[*e2eUser, *e2eProfile]{
		RelName: "Profile",
		Sides: []Side{
			{FromTable: "mh_users", FromCols: []string{"id"}, ToTable: "mh_user_profile", ToCols: []string{"user_id"}},
			{FromTable: "mh_user_profile", FromCols: []string{"profile_id"}, ToTable: "mh_profiles", ToCols: []string{"id"}},
		},
		ToMany: false,
		Child: TableParts[*e2eProfile]{
			Table:   "mh_profiles",
			Columns: []Column{{Name: "id", DBType: "int8"}, {Name: "bio", DBType: "text"}},
			NewBuf:  func() Buf[*e2eProfile] { return &e2eProfileBuf{} },
		},
		Carried:       []Column{{Name: "_sl_k0", DBType: "int8"}},
		NewCarriedBuf: func() Buf[*e2eUser] { return &e2eUserBuf{} },
		ParentKey:     func(p *e2eUser) []any { return []any{p.ID} },
		ChildKey:      func(c *e2eProfile) []any { return []any{c.ID} },
		Stitch:        stitch,
	}
}

// e2eUserToMidEdge/e2eMidToRoleEdge: the SAME relationship as
// e2eRolesEdge, expressed as two NESTED single-hop edges instead of one
// 2-hop edge — the parity fixture.
func e2eUserToMidEdge(stitch func(p *e2eUser, cs []*e2eUserRole) error) Edge[*e2eUser, *e2eUserRole] {
	return Edge[*e2eUser, *e2eUserRole]{
		RelName:  "UserRoles",
		FromCols: []string{"id"},
		ToCols:   []string{"user_id"},
		ToMany:   true,
		Child: TableParts[*e2eUserRole]{
			Table:   "mh_user_roles",
			Columns: []Column{{Name: "user_id", DBType: "int8"}, {Name: "role_id", DBType: "int8"}},
			NewBuf:  func() Buf[*e2eUserRole] { return &e2eUserRoleBuf{} },
		},
		ParentKey: func(p *e2eUser) []any { return []any{p.ID} },
		ChildKey:  func(c *e2eUserRole) []any { return []any{c.UserID} },
		Stitch:    stitch,
	}
}

func e2eMidToRoleEdge(stitch func(p *e2eUserRole, cs []*e2eRole) error) Edge[*e2eUserRole, *e2eRole] {
	return Edge[*e2eUserRole, *e2eRole]{
		RelName:  "Role",
		FromCols: []string{"role_id"},
		ToCols:   []string{"id"},
		ToMany:   false,
		Child:    e2eRoleParts(),
		ParentKey: func(p *e2eUserRole) []any {
			if !p.RoleID.Valid {
				return nil // NULL intermediate FK: no link, Stitch still called once (to-one contract)
			}
			return []any{p.RoleID.Int64}
		},
		ChildKey: func(c *e2eRole) []any { return []any{c.ID} },
		Stitch:   stitch,
	}
}

// --- shared fixture setup -----------------------------------------------

// e2eFixture opens testdb, skips if unreachable, begins a transaction,
// creates the schema above with its data, and returns (tx, cleanup). The
// caller must defer cleanup() to always roll back.
func e2eFixture(t *testing.T) (bob.Tx, func()) {
	t.Helper()
	ctx := context.Background()

	sqlDB, err := sql.Open("postgres", multiHopE2EDSN(t))
	if err != nil {
		t.Fatalf("open testdb: %v", err)
	}
	if err := sqlDB.PingContext(ctx); err != nil {
		_ = sqlDB.Close()
		t.Skipf("testdb not reachable: %v", err)
	}

	db := bob.NewDB(sqlDB)
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		_ = sqlDB.Close()
		t.Fatalf("BeginTx: %v", err)
	}

	fixtureSQL := []string{
		`CREATE TABLE mh_users (id bigint PRIMARY KEY, name text NOT NULL)`,
		`CREATE TABLE mh_roles (id bigint PRIMARY KEY, label text NOT NULL)`,
		`CREATE TABLE mh_user_roles (user_id bigint, role_id bigint)`,
		`CREATE TABLE mh_profiles (id bigint PRIMARY KEY, bio text NOT NULL)`,
		`CREATE TABLE mh_user_profile (user_id bigint UNIQUE, profile_id bigint)`,

		`INSERT INTO mh_users VALUES (1, 'alice'), (2, 'bob'), (3, 'carol')`,
		`INSERT INTO mh_roles VALUES (10, 'admin'), (11, 'editor')`,
		// (a)+(b): alice has admin once, editor twice (duplicate pair).
		`INSERT INTO mh_user_roles VALUES (1, 10), (1, 11), (1, 11)`,
		// (c): orphan join-table row, no matching user (dropped by the
		// semi-join, must not error or attach anywhere).
		`INSERT INTO mh_user_roles VALUES (999, 10)`,
		// (e): carol's join-table row has a NULL role_id (INNER JOIN to
		// roles drops it — carol ends up with zero role rows despite
		// having a join-table row at all). (d) bob has NO join-table row
		// at all (child-less parent, the other empty-result shape).
		`INSERT INTO mh_user_roles VALUES (3, NULL)`,

		`INSERT INTO mh_profiles VALUES (100, 'alice bio')`,
		`INSERT INTO mh_user_profile VALUES (1, 100)`,
		// bob and carol have no mh_user_profile row at all.
	}
	for _, stmt := range fixtureSQL {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			_ = tx.Rollback(ctx)
			_ = sqlDB.Close()
			t.Fatalf("fixture statement failed:\n%s\nerror: %v", stmt, err)
		}
	}

	cleanup := func() {
		_ = tx.Rollback(ctx)
		_ = sqlDB.Close()
	}
	return tx, cleanup
}

func e2eUsersQuery() bob.Query {
	return psql.RawQuery("SELECT id, name FROM mh_users")
}

// --- tests ---------------------------------------------------------

// TestMultiHopE2E_TwoHopToMany covers (a)-(e): basic to-many grouping,
// duplicate join-table pairs, an orphan join-table row, a child-less
// parent, and a NULL intermediate FK — all against the SAME 2-hop edge.
func TestMultiHopE2E_TwoHopToMany(t *testing.T) {
	tx, cleanup := e2eFixture(t)
	defer cleanup()
	ctx := context.Background()

	stitched := map[int64][]string{}
	calls := map[int64]int{}
	edge := e2eRolesEdge(func(p *e2eUser, cs []*e2eRole) error {
		calls[p.ID]++
		labels := make([]string, len(cs))
		for i, c := range cs {
			labels[i] = c.Label
		}
		sort.Strings(labels)
		stitched[p.ID] = labels
		return nil
	})

	users, err := All(ctx, tx, e2eUsersQuery(), e2eUserParts(), NewNode(edge))
	if err != nil {
		t.Fatalf("All: %v", err)
	}
	if len(users) != 3 {
		t.Fatalf("got %d users, want 3", len(users))
	}

	// (a)+(b): alice has admin once and editor TWICE — the duplicate
	// join-table pair must produce a duplicate role row, matching bob's
	// own INNER JOIN semantics (no implicit DISTINCT).
	if got, want := stitched[1], []string{"admin", "editor", "editor"}; !equalStrings(got, want) {
		t.Fatalf("alice's roles = %v, want %v", got, want)
	}
	// (d): bob has no join-table row at all — to-many stitches an empty
	// slice (not skipped: his parent key is not null, there is simply no
	// match — stitch.go's documented to-many contract).
	if got := stitched[2]; len(got) != 0 {
		t.Fatalf("bob's roles = %v, want empty (child-less parent)", got)
	}
	// (e): carol's join-table row has role_id NULL — the INNER JOIN to
	// roles drops it entirely, so carol also ends up with zero role rows,
	// despite having a join-table row. No panic, no error.
	if got := stitched[3]; len(got) != 0 {
		t.Fatalf("carol's roles = %v, want empty (NULL intermediate FK)", got)
	}
	// Every one of the 3 users got Stitch called exactly once — the (a)
	// contract check that a to-many edge's caller sees ALL parents,
	// including the empty ones.
	for _, id := range []int64{1, 2, 3} {
		if calls[id] != 1 {
			t.Fatalf("Stitch called %d times for user %d, want 1", calls[id], id)
		}
	}
	// (c): the orphan join-table row (user_id=999) must not have caused
	// an error and must not appear anywhere — there is no user 999 in
	// `users` or `stitched` to check against, so its ABSENCE from calls
	// (only 1/2/3 present) is itself the assertion.
	if len(calls) != 3 {
		t.Fatalf("Stitch called for %d distinct users, want 3 (orphan join-table row must not create a phantom parent): %v", len(calls), calls)
	}
}

// TestMultiHopE2E_TwoHopToOne_ChildlessParentStillStitchesLoaded pins the
// to-one contract for a 2-hop edge specifically: a parent with NO
// matching row anywhere in the join chain still gets Stitch called
// exactly once, with zero children — this is what "Loaded=true, no
// value" means for a generated model's R/RLoaded fields.
func TestMultiHopE2E_TwoHopToOne_ChildlessParentStillStitchesLoaded(t *testing.T) {
	tx, cleanup := e2eFixture(t)
	defer cleanup()
	ctx := context.Background()

	calls := map[int64]int{}
	profiles := map[int64]*e2eProfile{}
	edge := e2eProfileEdge(func(p *e2eUser, cs []*e2eProfile) error {
		calls[p.ID]++
		if len(cs) > 0 {
			profiles[p.ID] = cs[0]
		}
		return nil
	})

	_, err := All(ctx, tx, e2eUsersQuery(), e2eUserParts(), NewNode(edge))
	if err != nil {
		t.Fatalf("All: %v", err)
	}

	if p := profiles[1]; p == nil || p.Bio != "alice bio" {
		t.Fatalf("alice's profile = %+v, want Bio=\"alice bio\"", p)
	}
	for _, id := range []int64{1, 2, 3} {
		if calls[id] != 1 {
			t.Fatalf("Stitch called %d times for user %d, want 1 (to-one Loaded contract)", calls[id], id)
		}
	}
	if profiles[2] != nil || profiles[3] != nil {
		t.Fatalf("bob/carol must have no profile: got bob=%+v carol=%+v", profiles[2], profiles[3])
	}
}

// TestMultiHopE2E_ThenBoundary_SameResultAsInIsland re-runs the exact to-many
// case, but with the 2-hop edge placed behind Then(...) — a second round
// trip (route 4) instead of the same island (routes 1/2). The result
// must be identical.
func TestMultiHopE2E_ThenBoundary_SameResultAsInIsland(t *testing.T) {
	tx, cleanup := e2eFixture(t)
	defer cleanup()
	ctx := context.Background()

	stitched := map[int64][]string{}
	edge := e2eRolesEdge(func(p *e2eUser, cs []*e2eRole) error {
		labels := make([]string, len(cs))
		for i, c := range cs {
			labels[i] = c.Label
		}
		sort.Strings(labels)
		stitched[p.ID] = labels
		return nil
	})

	users, err := All(ctx, tx, e2eUsersQuery(), e2eUserParts(), Then(NewNode(edge)))
	if err != nil {
		t.Fatalf("All (Then boundary): %v", err)
	}
	if len(users) != 3 {
		t.Fatalf("got %d users, want 3", len(users))
	}
	if got, want := stitched[1], []string{"admin", "editor", "editor"}; !equalStrings(got, want) {
		t.Fatalf("alice's roles (Then boundary) = %v, want %v", got, want)
	}
	if len(stitched[2]) != 0 || len(stitched[3]) != 0 {
		t.Fatalf("bob/carol must still be empty behind a Then boundary: bob=%v carol=%v", stitched[2], stitched[3])
	}
}

// TestMultiHopE2E_NestedSingleHopUnderMultiHop nests a plain single-hop child
// under the 2-hop edge, forcing it to have children of its own
// (hasKids=true — route 1 CTE + route 3 read-back, not just route 2 as a
// leaf). This is the execution-side counterpart of
// TestMultiHopRoute3_CTEHasJoinReadBackDoesNot (multihop_sql_test.go, SQL-shape
// only).
func TestMultiHopE2E_NestedSingleHopUnderMultiHop(t *testing.T) {
	tx, cleanup := e2eFixture(t)
	defer cleanup()
	ctx := context.Background()

	if _, err := tx.ExecContext(ctx, `CREATE TABLE mh_role_tags (role_id bigint, tag text NOT NULL)`); err != nil {
		t.Fatalf("create mh_role_tags: %v", err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO mh_role_tags VALUES (10, 'built-in'), (11, 'built-in')`); err != nil {
		t.Fatalf("insert mh_role_tags: %v", err)
	}

	tagsByRole := map[int64][]string{}
	roleStitch := func(p *e2eRole, cs []roleTag) error {
		for _, c := range cs {
			tagsByRole[p.ID] = append(tagsByRole[p.ID], c.Tag)
		}
		return nil
	}
	roleTagEdge := Edge[*e2eRole, roleTag]{
		RelName:  "Tags",
		FromCols: []string{"id"},
		ToCols:   []string{"role_id"},
		ToMany:   true,
		Child: TableParts[roleTag]{
			Table:   "mh_role_tags",
			Columns: []Column{{Name: "role_id", DBType: "int8"}, {Name: "tag", DBType: "text"}},
			NewBuf:  func() Buf[roleTag] { return &roleTagBuf{} },
		},
		ParentKey: func(p *e2eRole) []any { return []any{p.ID} },
		ChildKey:  func(c roleTag) []any { return []any{c.RoleID} },
		Stitch:    roleStitch,
	}

	stitchedRoles := map[int64][]string{}
	usersEdge := e2eRolesEdge(func(p *e2eUser, cs []*e2eRole) error {
		labels := make([]string, len(cs))
		for i, c := range cs {
			labels[i] = c.Label
		}
		sort.Strings(labels)
		stitchedRoles[p.ID] = labels
		return nil
	})

	_, err := All(ctx, tx, e2eUsersQuery(), e2eUserParts(), NewNode(usersEdge, NewNode(roleTagEdge)))
	if err != nil {
		t.Fatalf("All: %v", err)
	}
	if got, want := stitchedRoles[1], []string{"admin", "editor", "editor"}; !equalStrings(got, want) {
		t.Fatalf("alice's roles (with nested child) = %v, want %v", got, want)
	}
	// Each role ROW returned for alice must have its own tag stitched via
	// the multi-hop CTE's own join key (route 1) — including the
	// duplicate editor row: alice's join-table has (1,11) TWICE, so
	// editor legitimately surfaces as 2 SEPARATE section rows (sectionload's
	// INNER JOIN chain reproduces bob's own duplicate-row semantics, with
	// no implicit DISTINCT), and the nested Tags
	// edge below is invoked independently for EACH of those 2 rows — so
	// "built-in" is legitimately stitched twice for role 11, once per
	// row, not deduplicated to one.
	if got := tagsByRole[10]; len(got) != 1 || got[0] != "built-in" {
		t.Fatalf("admin's tags = %v, want [\"built-in\"] (admin appears once)", got)
	}
	if got := tagsByRole[11]; len(got) != 2 || got[0] != "built-in" || got[1] != "built-in" {
		t.Fatalf("editor's tags = %v, want [\"built-in\", \"built-in\"] (editor appears twice, once per duplicate join-table row)", got)
	}
}

// roleTag is mh_role_tags' own model — package-level (not declared
// inside the test function) because roleTagBuf.BuildFor's return type
// must be the EXACT type Buf[roleTag]/TableParts[roleTag] expect; a
// function-local type declaration with the same fields would be a
// different, incompatible type even though structurally identical.
type roleTag struct {
	RoleID int64
	Tag    string
}
type roleTagBuf struct {
	roleID sql.NullInt64
	tag    sql.NullString
}

func (b *roleTagBuf) TargetsFor(cols []Column) []any { return []any{&b.roleID, &b.tag} }
func (b *roleTagBuf) BuildFor(modelCols []Column) roleTag {
	return roleTag{RoleID: b.roleID.Int64, Tag: b.tag.String}
}

// TestMultiHopE2E_Parity_MultiHopEdgeVsNestedSingleHopEdges is a parity check:
// "a multi-hop Edge" and "two nested single-hop Edges expressing the same
// relationship" must produce the SAME final role-id multiset per user.
// Instance identity is explicitly NOT compared — only the multiset of role
// IDs.
func TestMultiHopE2E_Parity_MultiHopEdgeVsNestedSingleHopEdges(t *testing.T) {
	tx, cleanup := e2eFixture(t)
	defer cleanup()
	ctx := context.Background()

	// Multi-hop edge, one section.
	multiHop := map[int64][]int64{}
	edgeMulti := e2eRolesEdge(func(p *e2eUser, cs []*e2eRole) error {
		ids := make([]int64, len(cs))
		for i, c := range cs {
			ids[i] = c.ID
		}
		sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
		multiHop[p.ID] = ids
		return nil
	})
	if _, err := All(ctx, tx, e2eUsersQuery(), e2eUserParts(), NewNode(edgeMulti)); err != nil {
		t.Fatalf("All (multi-hop): %v", err)
	}

	// Two nested single-hop edges over the SAME join table.
	nested := map[int64][]int64{}
	midToRole := e2eMidToRoleEdge(func(p *e2eUserRole, cs []*e2eRole) error {
		if len(cs) > 0 {
			p.Role = cs[0]
		}
		return nil
	})
	userToMid := e2eUserToMidEdge(func(p *e2eUser, cs []*e2eUserRole) error {
		var ids []int64
		for _, c := range cs {
			if c.Role != nil {
				ids = append(ids, c.Role.ID)
			}
		}
		sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
		nested[p.ID] = ids
		return nil
	})
	if _, err := All(ctx, tx, e2eUsersQuery(), e2eUserParts(), NewNode(userToMid, NewNode(midToRole))); err != nil {
		t.Fatalf("All (nested single-hop): %v", err)
	}

	for _, id := range []int64{1, 2, 3} {
		if !equalInt64s(multiHop[id], nested[id]) {
			t.Fatalf("user %d: multi-hop roles = %v, nested single-hop roles = %v — must match", id, multiHop[id], nested[id])
		}
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func equalInt64s(a, b []int64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
