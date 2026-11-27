package psql_test

import (
	"testing"

	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/lm"
	testutils "github.com/stephenafamo/bob/test/utils"
)

func TestListen(t *testing.T) {
	examples := testutils.Testcases{
		"simple": {
			Query:        psql.Listen(lm.Channel("my_channel")),
			ExpectedSQL:  `LISTEN "my_channel"`,
			ExpectedArgs: nil,
		},
	}

	testutils.RunTests(t, examples, formatter)
}
