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
