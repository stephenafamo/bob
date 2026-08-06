package psql_test

import (
	"testing"

	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/nm"
	testutils "github.com/stephenafamo/bob/test/utils"
)

func TestNotify(t *testing.T) {
	examples := testutils.Testcases{
		"simple": {
			Query:        psql.Notify(nm.Channel("my_channel")),
			ExpectedSQL:  `NOTIFY "my_channel"`,
			ExpectedArgs: nil,
		},
		"with payload": {
			Query:        psql.Notify(nm.Channel("my_channel"), nm.Payload("hello world")),
			ExpectedSQL:  `NOTIFY "my_channel", 'hello world'`,
			ExpectedArgs: nil,
		},
	}

	testutils.RunTests(t, examples, formatter)
}
