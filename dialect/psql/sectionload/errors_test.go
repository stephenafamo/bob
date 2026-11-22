package sectionload

import (
	"strings"
	"testing"
)

// TestMissingRequiredColumnError_SelfReferentialCaseDoesNotRepeatRelName
// covers the RelName == ChildRelName call site added by validate.go's own
// ToCols check the naive shared format string
// would render "relation \"X\" needs column \"Y\" on \"X\" for its join",
// naming the same relation twice. The message must mention the relation
// name exactly once and say "its own column" instead.
func TestMissingRequiredColumnError_SelfReferentialCaseDoesNotRepeatRelName(t *testing.T) {
	e := &MissingRequiredColumnError{RelName: "Children", ChildRelName: "Children", Column: "parent_id"}
	msg := e.Error()
	want := `sectionload: relation "Children" needs its own column "parent_id" for its join, but its ColumnPlan does not fetch it; build "Children"'s ColumnPlan with NewColumnPlan and pass "parent_id" as one of keyCols`
	if msg != want {
		t.Fatalf("Error() =\n%s\nwant:\n%s", msg, want)
	}
	// The redundant shared-format-string phrasing this replaces would have
	// read `relation "Children" needs column "parent_id" on "Children" for
	// its join` — the same relation named twice in one clause. Guard
	// against that phrasing coming back.
	if strings.Contains(msg, `on "Children" for its join`) {
		t.Fatalf("message repeats the relation name in \"...on X for its join\"; got: %s", msg)
	}
}

// TestMissingRequiredColumnError_DistinctRelationsCaseIsUnchanged pins the
// existing wording for RelName != ChildRelName (the pre-existing "nested
// edge's FromCols missing from the parent's ColumnPlan" call site) so a
// future edit to the self-referential branch cannot accidentally change
// this one too.
func TestMissingRequiredColumnError_DistinctRelationsCaseIsUnchanged(t *testing.T) {
	e := &MissingRequiredColumnError{RelName: "Children", ChildRelName: "Grandchildren", Column: "parent_id"}
	got := e.Error()
	want := `sectionload: relation "Grandchildren" needs column "parent_id" on "Children" for its join, but "Children"'s ColumnPlan does not fetch it; build "Children"'s ColumnPlan with NewColumnPlan and pass "parent_id" as one of keyCols`
	if got != want {
		t.Fatalf("Error() =\n%s\nwant:\n%s", got, want)
	}
}
