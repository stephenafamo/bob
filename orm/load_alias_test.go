package orm_test

import (
	"context"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/dialect"
	"github.com/stephenafamo/bob/dialect/psql/sm"
	"github.com/stephenafamo/bob/orm"
)

type aliasNameable struct{ name, alias string }

func (n aliasNameable) NameExpr() psql.Expression { return psql.Quote(n.name) }
func (n aliasNameable) Alias() string             { return n.alias }

type aliasChild struct {
	ID int64 `db:"id"`
}

func (c *aliasChild) Preload(name string, rel any) error { return nil }

type aliasGrandchild struct {
	ID int64 `db:"id"`
}

func (g *aliasGrandchild) Preload(name string, rel any) error { return nil }

var joinAliasRE = regexp.MustCompile(`(?i)(?:LEFT|INNER|RIGHT)?\s*JOIN\s+[^\s]+\s+AS\s+"([^"]+)"`)

func buildPreloadSQL(t *testing.T, mods ...bob.Mod[*dialect.SelectQuery]) string {
	t.Helper()
	q := psql.Select(append([]bob.Mod[*dialect.SelectQuery]{
		sm.From(psql.Quote("parents")),
		sm.Columns(psql.Quote("parents", "id")),
	}, mods...)...)
	sql, _, err := bob.Build(context.Background(), q)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	return sql
}

func joinAliases(sql string) []string {
	matches := joinAliasRE.FindAllStringSubmatch(sql, -1)
	out := make([]string, 0, len(matches))
	for _, m := range matches {
		out = append(out, m[1])
	}
	return out
}

func assertDistinct(t *testing.T, aliases []string) {
	t.Helper()
	seen := make(map[string]struct{}, len(aliases))
	for _, a := range aliases {
		if _, ok := seen[a]; ok {
			t.Fatalf("duplicate alias %q in %v", a, aliases)
		}
		seen[a] = struct{}{}
	}
}

func assertCompact(t *testing.T, aliases ...string) {
	t.Helper()
	for _, a := range aliases {
		if !strings.HasPrefix(a, "bob_q") {
			t.Fatalf("alias %q is not a compact bob_q alias", a)
		}
	}
}

func oneSideRel(fromName, fromAlias, toName, toAlias, fromCol, relName string) psql.PreloadRel {
	return psql.PreloadRel{
		Name: relName,
		Sides: []psql.PreloadSide{{
			From:        aliasNameable{name: fromName, alias: fromAlias},
			To:          aliasNameable{name: toName, alias: toAlias},
			FromColumns: []string{fromCol},
			ToColumns:   []string{"id"},
		}},
	}
}

func multiSideRel() psql.PreloadRel {
	return psql.PreloadRel{
		Name: "Through",
		Sides: []psql.PreloadSide{
			{
				From:        aliasNameable{name: "parents", alias: "parents"},
				To:          aliasNameable{name: "junction", alias: "junction"},
				FromColumns: []string{"id"},
				ToColumns:   []string{"parent_id"},
			},
			{
				From:        aliasNameable{name: "junction", alias: "junction"},
				To:          aliasNameable{name: "children", alias: "children"},
				FromColumns: []string{"child_id"},
				ToColumns:   []string{"id"},
			},
		},
	}
}

func TestPreloadAliasSQLStable(t *testing.T) {
	t.Parallel()

	rel := oneSideRel("parents", "parents", "children", "children", "child_id", "Child")
	mk := func() bob.Mod[*dialect.SelectQuery] {
		return psql.Preload[*aliasChild, []*aliasChild](rel, []string{"id"}, nil)
	}

	sql1 := buildPreloadSQL(t, mk())
	sql2 := buildPreloadSQL(t, mk())
	if sql1 != sql2 {
		t.Fatalf("SQL not stable across builds:\n%s\n---\n%s", sql1, sql2)
	}
	aliases := joinAliases(sql1)
	if len(aliases) != 1 || aliases[0] != "bob_q0" {
		t.Fatalf("aliases = %v, want [bob_q0]", aliases)
	}
}

func TestPreloadAliasSiblingAndDuplicate(t *testing.T) {
	t.Parallel()

	pilot := oneSideRel("parents", "parents", "children", "children", "pilot_id", "Pilot")
	copilot := oneSideRel("parents", "parents", "children", "children", "copilot_id", "CoPilot")

	sql := buildPreloadSQL(t,
		psql.Preload[*aliasChild, []*aliasChild](pilot, []string{"id"}, nil),
		psql.Preload[*aliasChild, []*aliasChild](copilot, []string{"id"}, nil),
		psql.Preload[*aliasChild, []*aliasChild](pilot, []string{"id"}, nil), // duplicate implicit
	)
	aliases := joinAliases(sql)
	if len(aliases) != 3 {
		t.Fatalf("aliases = %v, want 3", aliases)
	}
	assertDistinct(t, aliases)
	assertCompact(t, aliases...)
	if aliases[0] != "bob_q0" || aliases[1] != "bob_q1" || aliases[2] != "bob_q2" {
		t.Fatalf("aliases = %v, want bob_q0..bob_q2", aliases)
	}
}

func TestPreloadAliasNested(t *testing.T) {
	t.Parallel()

	childRel := oneSideRel("parents", "parents", "children", "children", "child_id", "Child")
	grandRel := oneSideRel("children", "children", "grandchildren", "grandchildren", "grandchild_id", "Grandchild")

	sql := buildPreloadSQL(t,
		psql.Preload[*aliasChild, []*aliasChild](childRel, []string{"id"}, nil,
			psql.Preload[*aliasGrandchild, []*aliasGrandchild](grandRel, []string{"id"}, nil),
		),
	)
	aliases := joinAliases(sql)
	if len(aliases) != 2 {
		t.Fatalf("aliases = %v, want 2", aliases)
	}
	assertDistinct(t, aliases)
	assertCompact(t, aliases...)
	if !strings.Contains(sql, `"bob_q0"."grandchild_id"`) {
		t.Fatalf("nested ON should qualify parent as bob_q0:\n%s", sql)
	}
}

func TestPreloadAsFinalSideOnly(t *testing.T) {
	t.Parallel()

	rel := multiSideRel()
	sql := buildPreloadSQL(t,
		psql.Preload[*aliasChild, []*aliasChild](rel, []string{"id"}, nil, psql.PreloadAs("final")),
	)
	aliases := joinAliases(sql)
	if len(aliases) != 2 {
		t.Fatalf("aliases = %v, want 2", aliases)
	}
	if aliases[0] != "bob_q0" {
		t.Fatalf("intermediate alias = %q, want bob_q0", aliases[0])
	}
	if aliases[1] != "final" {
		t.Fatalf("final alias = %q, want final", aliases[1])
	}
	if !strings.Contains(sql, `"bob_q0"."child_id" = "final"."id"`) {
		t.Fatalf("ON clause should link intermediate to final:\n%s", sql)
	}
	if !strings.Contains(sql, `"parents"."id" = "bob_q0"."parent_id"`) {
		t.Fatalf("first ON should use parent table and intermediate:\n%s", sql)
	}
}

func TestPreloadAsMultiHopDistinctFinals(t *testing.T) {
	t.Parallel()

	rel := multiSideRel()
	sql := buildPreloadSQL(t,
		psql.Preload[*aliasChild, []*aliasChild](rel, []string{"id"}, nil, psql.PreloadAs("a")),
		psql.Preload[*aliasChild, []*aliasChild](rel, []string{"id"}, nil, psql.PreloadAs("b")),
	)
	aliases := joinAliases(sql)
	if len(aliases) != 4 {
		t.Fatalf("aliases = %v, want 4", aliases)
	}
	assertDistinct(t, aliases)
	if aliases[1] != "a" || aliases[3] != "b" {
		t.Fatalf("final aliases = %v, want a and b at positions 1 and 3", aliases)
	}
	assertCompact(t, aliases[0], aliases[2])
}

func TestPreloadAsNestedUsesExplicitParent(t *testing.T) {
	t.Parallel()

	childRel := oneSideRel("parents", "parents", "children", "children", "child_id", "Child")
	grandRel := oneSideRel("children", "children", "grandchildren", "grandchildren", "grandchild_id", "Grandchild")

	sql := buildPreloadSQL(t,
		psql.Preload[*aliasChild, []*aliasChild](childRel, []string{"id"}, nil,
			psql.PreloadAs("p"),
			psql.Preload[*aliasGrandchild, []*aliasGrandchild](grandRel, []string{"id"}, nil),
		),
	)
	aliases := joinAliases(sql)
	if len(aliases) != 2 {
		t.Fatalf("aliases = %v, want 2", aliases)
	}
	if aliases[0] != "p" {
		t.Fatalf("parent alias = %q, want p", aliases[0])
	}
	assertCompact(t, aliases[1])
	if !strings.Contains(sql, `"p"."grandchild_id" = "`+aliases[1]+`"."id"`) {
		t.Fatalf("nested ON should use explicit parent alias:\n%s", sql)
	}
}

func TestPreloadWhereReceivesActualAliases(t *testing.T) {
	t.Parallel()

	rel := oneSideRel("parents", "parents", "children", "children", "child_id", "Child")
	var gotFrom, gotTo string
	sql := buildPreloadSQL(t,
		psql.Preload[*aliasChild, []*aliasChild](rel, []string{"id"}, nil,
			psql.PreloadWhere(func(from, to string) []bob.Expression {
				gotFrom, gotTo = from, to
				return nil
			}),
		),
	)
	aliases := joinAliases(sql)
	if gotFrom != "parents" || gotTo != aliases[0] {
		t.Fatalf("PreloadWhere got (%q, %q), want (parents, %q)", gotFrom, gotTo, aliases[0])
	}
}

func manualPreloadSQL(t *testing.T, loader psql.Preloader, opts ...orm.PreloaderOpt) string {
	t.Helper()
	mod, _, _ := loader("", opts...)
	q := psql.Select(sm.From(psql.Quote("parents")), sm.Columns(psql.Quote("parents", "id")), mod)
	sql, _, err := bob.Build(context.Background(), q)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	return sql
}

func TestPreloadManualInvocationWithNextAlias(t *testing.T) {
	t.Parallel()

	childRel := oneSideRel("parents", "parents", "children", "children", "child_id", "Child")
	grandRel := oneSideRel("children", "children", "grandchildren", "grandchildren", "grandchild_id", "Grandchild")
	loader := psql.Preload[*aliasChild, []*aliasChild](childRel, []string{"id"}, nil,
		psql.Preload[*aliasGrandchild, []*aliasGrandchild](grandRel, []string{"id"}, nil),
	)

	n := 0
	next := func() string {
		n++
		return "custom" + strconv.Itoa(n)
	}
	aliases := joinAliases(manualPreloadSQL(t, loader, orm.WithNextAlias(next)))
	if len(aliases) != 2 || aliases[0] != "custom1" || aliases[1] != "custom2" {
		t.Fatalf("aliases = %v, want [custom1 custom2]", aliases)
	}
}

func TestPreloadManualInvocationFallbackIsUnique(t *testing.T) {
	t.Parallel()

	rel := oneSideRel("parents", "parents", "children", "children", "child_id", "Child")
	loader := psql.Preload[*aliasChild, []*aliasChild](rel, []string{"id"}, nil)

	// Two manual calls on one query must not collide.
	modA, _, _ := loader("")
	modB, _, _ := loader("")
	q := psql.Select(sm.From(psql.Quote("parents")), sm.Columns(psql.Quote("parents", "id")), modA, modB)
	sql, _, err := bob.Build(context.Background(), q)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	aliases := joinAliases(sql)
	if len(aliases) != 2 {
		t.Fatalf("aliases = %v, want 2", aliases)
	}
	assertDistinct(t, aliases)
	for _, a := range aliases {
		if !strings.HasPrefix(a, "bob_g") {
			t.Fatalf("fallback alias %q, want bob_g prefix", a)
		}
	}
}

func TestPreloadAliasCloneContinuesSequence(t *testing.T) {
	t.Parallel()

	rel := oneSideRel("parents", "parents", "children", "children", "child_id", "Child")
	preload := func() bob.Mod[*dialect.SelectQuery] {
		return psql.Preload[*aliasChild, []*aliasChild](rel, []string{"id"}, nil)
	}

	q := psql.Select(sm.From(psql.Quote("parents")), sm.Columns(psql.Quote("parents", "id")), preload())
	c := q.Clone()
	c.Apply(preload())

	sql, _, err := bob.Build(context.Background(), c)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	aliases := joinAliases(sql)
	if len(aliases) != 2 || aliases[0] != "bob_q0" || aliases[1] != "bob_q1" {
		t.Fatalf("aliases = %v, want [bob_q0 bob_q1]", aliases)
	}
}

func TestPreloadWhereQualifiesSides(t *testing.T) {
	t.Parallel()

	rel := oneSideRel("parents", "parents", "children", "children", "child_id", "Child")
	rel.Sides[0].FromWhere = []orm.RelWhere{{Column: "parent_kind", SQLValue: "'p'"}}
	rel.Sides[0].ToWhere = []orm.RelWhere{{Column: "child_kind", SQLValue: "'c'"}}

	sql := buildPreloadSQL(t, psql.Preload[*aliasChild, []*aliasChild](rel, []string{"id"}, nil))
	if !strings.Contains(sql, `"parents"."parent_kind" = 'p'`) {
		t.Fatalf("FromWhere should qualify the parent table:\n%s", sql)
	}
	if !strings.Contains(sql, `"bob_q0"."child_kind" = 'c'`) {
		t.Fatalf("ToWhere should qualify the joined table:\n%s", sql)
	}
}
