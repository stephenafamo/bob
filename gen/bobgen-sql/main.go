package main

import (
	"context"
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/stephenafamo/bob/gen"
	helpers "github.com/stephenafamo/bob/gen/bobgen-helpers"
	"github.com/stephenafamo/bob/gen/bobgen-sql/driver"
	"github.com/urfave/cli/v2"
)

func main() {
	ctx, cancel := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT,
		syscall.SIGTERM,
	)
	defer cancel()

	app := &cli.App{
		Name:      "bobgen-sql",
		Usage:     "Generate models and factories from your SQL schema files",
		UsageText: "bobgen-sql [-c FILE]",
		Version:   helpers.Version(),
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "config",
				Aliases: []string{"c"},
				Value:   helpers.DefaultConfigPath,
				Usage:   "Load configuration from `FILE`",
			},
		},
		Action: run,
	}

	if err := app.RunContext(ctx, os.Args); err != nil {
		log.Fatal(err)
	}
}

func run(c *cli.Context) error {
	config, driverConfig, err := helpers.GetConfigFromFile[driver.Config](c.String("config"), "sql")
	if err != nil {
		return err
	}

	var modelTemplates []fs.FS
	switch driverConfig.Dialect {
	case "psql", "postgres":
		modelTemplates = append(modelTemplates, gen.PSQLModelTemplates)
	case "mysql":
		modelTemplates = append(modelTemplates, gen.MySQLModelTemplates)
	case "sqlite":
		modelTemplates = append(modelTemplates, gen.SQLiteModelTemplates)
	}

	outputs := helpers.DefaultOutputs(
		driverConfig.Output, driverConfig.Pkgname, config.NoFactory,
		&helpers.Templates{Models: modelTemplates},
	)

	state := &gen.State{
		Config:  config,
		Outputs: outputs,
	}

	switch driverConfig.Dialect {
	case "psql", "postgres":
		return driver.RunPostgres(c.Context, state, driverConfig)
	case "mysql":
		return driver.RunMySQL(c.Context, state, driverConfig)
	case "sqlite":
		return driver.RunSQLite(c.Context, state, driverConfig)
	default:
		return fmt.Errorf("unsupported dialect %s", driverConfig.Dialect)
	}
}
