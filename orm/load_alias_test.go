package orm_test

import (
	"testing"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/expr"
	testutils "github.com/stephenafamo/bob/test/utils"
)

type aliasPilot struct {
	ID       int64 `db:"id,pk"`
	MentorID int64 `db:"mentor_id"`
}

func (*aliasPilot) Preload(string, any) error { return nil }

var aliasPilots = psql.NewView[*aliasPilot, bob.Expression]("", "pilots", expr.ColsForStruct[aliasPilot]("pilots"))

// preloads the mentor of a pilot, who is a pilot too
func preloadMentor(opts ...psql.PreloadOption) psql.Preloader {
	return psql.Preload[*aliasPilot, []*aliasPilot](psql.PreloadRel{
		Name: "Mentor",
		Sides: []psql.PreloadSide{{
			From:        aliasPilots,
			To:          aliasPilots,
			FromColumns: []string{"mentor_id"},
			ToColumns:   []string{"id"},
		}},
	}, []string{"id"}, nil, opts...)
}

// The aliases of preloaded tables are numbered per query, so building the same
// query again gives the same SQL. Otherwise a driver that caches prepared
// statements by their SQL (such as pgx) can never reuse the statement.
func TestPreloadAliases(t *testing.T) {
	// what the queries below have in common, up to the condition of the second join
	common := `SELECT "pilots"."id" AS "id", "pilots"."mentor_id" AS "mentor_id", "pilots_10001"."id" AS "pilots_10001.id", "pilots_10002"."id" AS "pilots_10002.id" FROM "pilots" LEFT JOIN "pilots" AS "pilots_10001" ON "pilots"."mentor_id" = "pilots_10001"."id" LEFT JOIN "pilots" AS "pilots_10002" ON `
	nested := common + `"pilots_10001"."mentor_id" = "pilots_10002"."id"`

	clone := aliasPilots.Query(preloadMentor()).Clone()
	clone.Apply(preloadMentor())

	examples := testutils.Testcases{
		"nested preload": {
			Doc:         "The mentor and the mentor's mentor get different aliases",
			ExpectedSQL: nested,
			Query:       aliasPilots.Query(preloadMentor(preloadMentor())),
		},
		"same query built again": {
			Doc:         "Building the same query again gives the same aliases",
			ExpectedSQL: nested,
			Query:       aliasPilots.Query(preloadMentor(preloadMentor())),
		},
		"preload added to a clone": {
			Doc:         "A clone continues the numbering of the query it was cloned from",
			ExpectedSQL: common + `"pilots"."mentor_id" = "pilots_10002"."id"`,
			Query:       clone,
		},
	}

	testutils.RunTests(t, examples, nil)
}

type aliasMentorship struct {
	PilotID  int64 `db:"pilot_id"`
	MentorID int64 `db:"mentor_id"`
}

var aliasMentorships = psql.NewView[*aliasMentorship, bob.Expression]("", "mentorships", expr.ColsForStruct[aliasMentorship]("mentorships"))

// preloads the mentor of a pilot through the mentorships table
func preloadMentorThrough(opts ...psql.PreloadOption) psql.Preloader {
	return psql.Preload[*aliasPilot, []*aliasPilot](psql.PreloadRel{
		Name: "Mentor",
		Sides: []psql.PreloadSide{
			{
				From:        aliasPilots,
				To:          aliasMentorships,
				FromColumns: []string{"id"},
				ToColumns:   []string{"pilot_id"},
			},
			{
				From:        aliasMentorships,
				To:          aliasPilots,
				FromColumns: []string{"mentor_id"},
				ToColumns:   []string{"id"},
			},
		},
	}, []string{"id"}, nil, opts...)
}

// PreloadAs names the preloaded table, so a relationship through another table
// only uses it for the last join and keeps a generated alias for the others.
func TestPreloadAsThroughTable(t *testing.T) {
	examples := testutils.Testcases{
		"explicit alias": {
			Doc:         "Only the preloaded table gets the explicit alias",
			ExpectedSQL: `SELECT "pilots"."id" AS "id", "pilots"."mentor_id" AS "mentor_id", "mentor"."id" AS "mentor.id" FROM "pilots" LEFT JOIN "mentorships" AS "mentorships_10001" ON "pilots"."id" = "mentorships_10001"."pilot_id" LEFT JOIN "pilots" AS "mentor" ON "mentorships_10001"."mentor_id" = "mentor"."id"`,
			Query:       aliasPilots.Query(preloadMentorThrough(psql.PreloadAs("mentor"))),
		},
		"two explicit aliases": {
			Doc:         "Preloading the same relationship twice gives each through table its own alias",
			ExpectedSQL: `SELECT "pilots"."id" AS "id", "pilots"."mentor_id" AS "mentor_id", "a"."id" AS "a.id", "b"."id" AS "b.id" FROM "pilots" LEFT JOIN "mentorships" AS "mentorships_10001" ON "pilots"."id" = "mentorships_10001"."pilot_id" LEFT JOIN "pilots" AS "a" ON "mentorships_10001"."mentor_id" = "a"."id" LEFT JOIN "mentorships" AS "mentorships_10002" ON "pilots"."id" = "mentorships_10002"."pilot_id" LEFT JOIN "pilots" AS "b" ON "mentorships_10002"."mentor_id" = "b"."id"`,
			Query:       aliasPilots.Query(preloadMentorThrough(psql.PreloadAs("a")), preloadMentorThrough(psql.PreloadAs("b"))),
		},
	}

	testutils.RunTests(t, examples, nil)
}
