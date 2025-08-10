package driver

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"

	"github.com/lib/pq"
	"github.com/stephenafamo/bob/gen"
	helpers "github.com/stephenafamo/bob/gen/bobgen-helpers"
	mysqlDriver "github.com/stephenafamo/bob/gen/bobgen-mysql/driver"
	psqlDriver "github.com/stephenafamo/bob/gen/bobgen-psql/driver"
	sqliteDriver "github.com/stephenafamo/bob/gen/bobgen-sqlite/driver"
	"github.com/stephenafamo/bob/gen/drivers"
	"github.com/stephenafamo/bob/gen/plugins"
	"github.com/testcontainers/testcontainers-go"
	mysqltest "github.com/testcontainers/testcontainers-go/modules/mysql"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

type Config struct {
	helpers.Config `yaml:",squash"`

	// What dialect to generate with
	// psql | mysql | sqlite
	Dialect string
	// Glob pattern to match migration files
	Pattern string
	// The database schemas to generate models for
	Schemas []string
	// The name of this schema will not be included in the generated models
	// a context value can then be used to set the schema at runtime
	// useful for multi-tenant setups
	SharedSchema string `yaml:"shared_schema"`
	// How many tables to fetch in parallel
	Concurrency int
	// Which UUID package to use (gofrs or google)
	UUIDPkg string `yaml:"uuid_pkg"`
	fs      fs.FS
}

func RunPostgres(ctx context.Context, state *gen.State[any], config Config, pluginsConfig plugins.Config) error {
	d, err := getPsqlDriver(ctx, config)
	if err != nil {
		return fmt.Errorf("getting psql driver: %w", err)
	}

	templates := helpers.TemplatesFromWellKnownTree(gen.PSQLTemplates)
	plugins := helpers.OutputPlugins[any, any, psqlDriver.IndexExtra](pluginsConfig, templates)

	return gen.Run(ctx, state, d, plugins...)
}

func getPsqlDriver(ctx context.Context, config Config) (psqlDriver.Interface, error) {
	postgresContainer, err := postgres.Run(
		ctx, "postgres:16",
		postgres.BasicWaitStrategies(),
		testcontainers.WithLogger(log.New(io.Discard, "", log.LstdFlags)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to start container: %w", err)
	}
	defer func() {
		if err := testcontainers.TerminateContainer(postgresContainer); err != nil {
			log.Printf("failed to terminate container: %s", err)
		}
	}()

	dsn, err := postgresContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		return nil, fmt.Errorf("failed to get connection string: %w", err)
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close()

	if err := helpers.Migrate(ctx, db, config.fs, config.Pattern); err != nil {
		return nil, fmt.Errorf("migrating: %w", err)
	}
	db.Close() // close early

	config.Dsn = dsn
	d := wrapDriver(ctx, psqlDriver.New(psqlDriver.Config{
		Config:       config.Config,
		Schemas:      pq.StringArray(config.Schemas),
		SharedSchema: config.SharedSchema,
		Concurrency:  config.Concurrency,
		UUIDPkg:      config.UUIDPkg,
	}))

	return d, nil
}

func RunMySQL(ctx context.Context, state *gen.State[any], config Config, pluginsConfig plugins.Config) error {
	d, err := getMySQLDriver(ctx, config)
	if err != nil {
		return fmt.Errorf("getting mysql driver: %w", err)
	}

	templates := helpers.TemplatesFromWellKnownTree(gen.MySQLTemplates)
	plugins := helpers.OutputPlugins[any, any, any](pluginsConfig, templates)

	return gen.Run(ctx, state, d, plugins...)
}

func getMySQLDriver(ctx context.Context, config Config) (mysqlDriver.Interface, error) {
	mysqlContainer, err := mysqltest.Run(ctx,
		"mysql:8.0.35",
		mysqltest.WithDatabase("bobgen"),
		mysqltest.WithUsername("root"),
		mysqltest.WithPassword("password"),
	)
	defer func() {
		if err := testcontainers.TerminateContainer(mysqlContainer); err != nil {
			fmt.Printf("failed to terminate MySQL container: %v\n", err)
		}
	}()
	if err != nil {
		return nil, fmt.Errorf("failed to start container: %w", err)
	}

	dsn, err := mysqlContainer.ConnectionString(ctx, "tls=skip-verify", "multiStatements=true", "parseTime=true")
	if err != nil {
		return nil, fmt.Errorf("failed to get connection string: %w", err)
	}

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close()

	if err := helpers.Migrate(ctx, db, config.fs, config.Pattern); err != nil {
		return nil, fmt.Errorf("migrating: %w", err)
	}
	db.Close() // close early

	config.Dsn = dsn
	d := wrapDriver(ctx, mysqlDriver.New(mysqlDriver.Config{
		Config:      config.Config,
		Concurrency: config.Concurrency,
	}))

	return d, nil
}

func RunSQLite(ctx context.Context, state *gen.State[any], config Config, pluginsConfig plugins.Config) error {
	d, err := getSQLiteDriver(ctx, config)
	if err != nil {
		return fmt.Errorf("getting sqlite driver: %w", err)
	}

	templates := helpers.TemplatesFromWellKnownTree(gen.SQLiteTemplates)
	plugins := helpers.OutputPlugins[any, any, sqliteDriver.IndexExtra](pluginsConfig, templates)

	return gen.Run(ctx, state, d, plugins...)
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

	if err := helpers.Migrate(ctx, db, config.fs, config.Pattern); err != nil {
		return nil, fmt.Errorf("migrating: %w", err)
	}
	db.Close() // close early

	config.Dsn = "file:" + tmp.Name()
	d := sqliteDriver.New(sqliteDriver.Config{
		Config:       config.Config,
		Attach:       attach,
		SharedSchema: config.SharedSchema,
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
