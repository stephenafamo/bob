package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/stephenafamo/bob/gen"
	helpers "github.com/stephenafamo/bob/gen/bobgen-helpers"
	"github.com/stephenafamo/bob/gen/bobgen-psql/driver"
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
		Name:      "bobgen-psql",
		Usage:     "Generate models and factories from your PostgreSQL database",
		UsageText: "bobgen-psql [-c FILE]",
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
	config, driverConfig, pluginsConfig, err := helpers.GetConfigFromFile[any, driver.Config](c.String("config"), "psql")
	if err != nil {
		return err
	}

	outputPlugins := helpers.OutputPlugins[any, any, driver.IndexExtra](
		pluginsConfig,
		helpers.TemplatesFromWellKnownTree(gen.PSQLTemplates),
	)

	state := &gen.State[any]{Config: config}
	return gen.Run(c.Context, state, driver.New(driverConfig), outputPlugins...)
}
