package main

import (
	"context"
	"fmt"
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
		fmt.Println(err)
		os.Exit(1)
	}
}

func run(c *cli.Context) error {
	config, driverConfig, pluginsConfig, err := helpers.GetConfigFromFile[any, driver.Config](c.String("config"), "sql")
	if err != nil {
		return err
	}

	var templates helpers.Templates
	switch driverConfig.Dialect {
	case "psql", "postgres":
		templates = helpers.TemplatesFromWellKnownTree(gen.PSQLTemplates)
	case "mysql":
		templates = helpers.TemplatesFromWellKnownTree(gen.MySQLTemplates)
	case "sqlite":
		templates = helpers.TemplatesFromWellKnownTree(gen.SQLiteTemplates)
	}

	outputPlugins := helpers.OutputPlugins[any](pluginsConfig, templates)

	if driverConfig.Pattern == "" {
		driverConfig.Pattern = "*.sql"
	}

	state := &gen.State[any]{Config: config}

	switch driverConfig.Dialect {
	case "psql", "postgres":
		return driver.RunPostgres(c.Context, state, driverConfig, outputPlugins...)
	case "mysql":
		return driver.RunMySQL(c.Context, state, driverConfig, outputPlugins...)
	case "sqlite":
		return driver.RunSQLite(c.Context, state, driverConfig, outputPlugins...)
	default:
		return fmt.Errorf("unsupported dialect %s", driverConfig.Dialect)
	}
}
