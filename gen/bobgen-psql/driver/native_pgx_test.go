package driver

import (
	"testing"

	helpers "github.com/stephenafamo/bob/gen/bobgen-helpers"
	"github.com/stephenafamo/bob/gen/bobgen-psql/driver/parser"
)

func TestNewSetsTranslatorDriver(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		driver string
		want   string
	}{
		{
			name:   "pgx v5",
			driver: pgxDriver,
			want:   parser.DriverPgx,
		},
		{
			name:   "pgx stdlib",
			driver: pgxStdlibDriver,
			want:   pgxStdlibDriver,
		},
		{
			name:   "default lib/pq",
			driver: "",
			want:   pqDriver,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			cfg := Config{Config: helpers.Config{}}
			if tc.driver != "" {
				cfg.Driver = tc.driver
			}

			d := New(cfg).(*driver)
			if d.translator.Driver != tc.want {
				t.Fatalf("Driver = %q, want %q", d.translator.Driver, tc.want)
			}
		})
	}
}
