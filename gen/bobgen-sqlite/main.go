package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/stephenafamo/bob/gen"
	helpers "github.com/stephenafamo/bob/gen/bobgen-helpers"
	"github.com/stephenafamo/bob/gen/bobgen-sqlite/driver"
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
		Name:      "bobgen-sqlite",
		Usage:     "Generate models and factories from your SQLite database",
		UsageText: "bobgen-sqlite [-c FILE]",
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
	config, driverConfig, pluginsConfig, err := helpers.GetConfigFromFile[any, driver.Config](c.String("config"), "sqlite")
	if err != nil {
		return err
	}

	outputPlugins := helpers.OutputPlugins[any, any, driver.IndexExtra](
		pluginsConfig,
		helpers.TemplatesFromWellKnownTree(gen.SQLiteTemplates),
	)

	state := &gen.State[any]{Config: config}
	return gen.Run(c.Context, state, driver.New(driverConfig), outputPlugins...)
}
