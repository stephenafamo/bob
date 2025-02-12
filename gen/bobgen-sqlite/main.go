package main

import (
	"context"
	"io/fs"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/twitter-payments/bob/gen"
	helpers "github.com/twitter-payments/bob/gen/bobgen-helpers"
	"github.com/twitter-payments/bob/gen/bobgen-sqlite/driver"
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
		log.Fatal(err)
	}
}

func run(c *cli.Context) error {
	config, driverConfig, err := helpers.GetConfigFromFile[any, driver.Config](c.String("config"), "sqlite")
	if err != nil {
		return err
	}

	d := driver.New(driverConfig)
	outputs := helpers.DefaultOutputs(
		driverConfig.Output, driverConfig.Pkgname, config.NoFactory,
		&helpers.Templates{Models: []fs.FS{gen.SQLiteModelTemplates}},
	)

	state := &gen.State[any]{
		Config:  config,
		Outputs: outputs,
	}

	return gen.Run(c.Context, state, d)
}
