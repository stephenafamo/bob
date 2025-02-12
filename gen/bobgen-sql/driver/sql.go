package driver

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	"github.com/lib/pq"
	"github.com/twitter-payments/bob/gen"
	helpers "github.com/twitter-payments/bob/gen/bobgen-helpers"
	psqlDriver "github.com/twitter-payments/bob/gen/bobgen-psql/driver"
	sqliteDriver "github.com/twitter-payments/bob/gen/bobgen-sqlite/driver"
	"github.com/twitter-payments/bob/gen/drivers"
)

type Config struct {
	// What dialect to generate with
	// psql | mysql | sqlite
	Dialect string
	// Where the SQL files are
	Dir string
	// The database schemas to generate models for
	Schemas []string
	// The name of this schema will not be included in the generated models
	// a context value can then be used to set the schema at runtime
	// useful for multi-tenant setups
	SharedSchema string `yaml:"shared_schema"`
	// List of tables that will be included. Others are ignored
	Only map[string][]string
	// List of tables that will be should be ignored. Others are included
	Except map[string][]string
	// How many tables to fetch in parallel
	Concurrency int
	// Which UUID package to use (gofrs or google)
	UUIDPkg string `yaml:"uuid_pkg"`
	// Which `database/sql` driver to use (the full module name)
	DriverName string `yaml:"driver_name"`

	Output    string
	Pkgname   string
	NoFactory bool `yaml:"no_factory"`

	fs fs.FS
}

func RunPostgres(ctx context.Context, state *gen.State[any], config Config) error {
	config.fs = os.DirFS(config.Dir)

	d, err := getPsqlDriver(ctx, config)
	if err != nil {
		return fmt.Errorf("getting psql driver: %w", err)
	}

	return gen.Run(ctx, state, d)
}

func getPsqlDriver(ctx context.Context, config Config) (psqlDriver.Interface, error) {
	port, err := helpers.GetFreePort()
	if err != nil {
		return nil, fmt.Errorf("could not get a free port: %w", err)
	}

	dbConfig := embeddedpostgres.
		DefaultConfig().
		RuntimePath(filepath.Join(os.TempDir(), "bobgen_sql")).
		Port(uint32(port))
	dsn := dbConfig.GetConnectionURL() + "?sslmode=disable"

	postgres := embeddedpostgres.NewDatabase(dbConfig)
	if err := postgres.Start(); err != nil {
		return nil, fmt.Errorf("starting embedded postgres: %w", err)
	}
	defer func() {
		if err := postgres.Stop(); err != nil {
			fmt.Println("Error stopping postgres:", err)
		}
	}()

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close()

	if err := helpers.Migrate(ctx, db, config.fs); err != nil {
		return nil, fmt.Errorf("migrating: %w", err)
	}
	db.Close() // close early

	d := wrapDriver(ctx, psqlDriver.New(psqlDriver.Config{
		Dsn: dsn,

		Schemas:      pq.StringArray(config.Schemas),
		SharedSchema: config.SharedSchema,
		Only:         config.Only,
		Except:       config.Except,
		Concurrency:  config.Concurrency,
		UUIDPkg:      config.UUIDPkg,
		DriverName:   config.DriverName,
		Output:       config.Output,
		Pkgname:      config.Pkgname,
		NoFactory:    config.NoFactory,
	}))

	return d, nil
}

func RunSQLite(ctx context.Context, state *gen.State[any], config Config) error {
	config.fs = os.DirFS(config.Dir)

	d, err := getSQLiteDriver(ctx, config)
	if err != nil {
		return fmt.Errorf("getting sqlite driver: %w", err)
	}

	return gen.Run(ctx, state, d)
}

func getSQLiteDriver(ctx context.Context, config Config) (sqliteDriver.Interface, error) {
	tmp, err := os.CreateTemp("", "bobgen_sqlite")
	if err != nil {
		return nil, fmt.Errorf("creating temp file: %w", err)
	}
	defer tmp.Close()

	db, err := sql.Open("sqlite", tmp.Name())
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close()

	attach := make(map[string]string)
	for _, schema := range config.Schemas {
		tmp, err := os.CreateTemp("", "bobgen_sqlite_"+schema)
		if err != nil {
			return nil, fmt.Errorf("creating temp file: %w", err)
		}
		defer tmp.Close()

		attach[schema] = tmp.Name()
		_, err = db.ExecContext(ctx, fmt.Sprintf(
			"attach database '%s' as %s", tmp.Name(), schema,
		))
		if err != nil {
			return nil, fmt.Errorf("could not attach %q: %w", schema, err)
		}
	}

	if err := helpers.Migrate(ctx, db, config.fs); err != nil {
		return nil, fmt.Errorf("migrating: %w", err)
	}
	db.Close() // close early

	d := sqliteDriver.New(sqliteDriver.Config{
		DSN:        tmp.Name(),
		Attach:     attach,
		DriverName: config.DriverName,

		SharedSchema: config.SharedSchema,
		Only:         config.Only,
		Except:       config.Except,
		Output:       config.Output,
		Pkgname:      config.Pkgname,
		NoFactory:    config.NoFactory,
	})

	return d, nil
}

func wrapDriver[T, C, I any](ctx context.Context, d drivers.Interface[T, C, I]) driver[T, C, I] {
	info, err := d.Assemble(ctx)
	return driver[T, C, I]{d, info, err}
}

type driver[T, C, I any] struct {
	drivers.Interface[T, C, I]
	info *drivers.DBInfo[T, C, I]
	err  error
}

func (d driver[T, C, I]) Assemble(context.Context) (*drivers.DBInfo[T, C, I], error) {
	return d.info, d.err
}
