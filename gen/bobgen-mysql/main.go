package main

import (
	"context"
	"io/fs"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/stephenafamo/bob/gen"
	helpers "github.com/stephenafamo/bob/gen/bobgen-helpers"
	"github.com/stephenafamo/bob/gen/bobgen-mysql/driver"
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
		Name:      "bobgen-mysql",
		Usage:     "Generate models and factories from your PostgreSQL database",
		UsageText: "bobgen-mysql [-c FILE]",
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
	config, driverConfig, err := helpers.GetConfigFromFile[driver.Config](c.String("config"), "mysql")
	if err != nil {
		return err
	}

	d := driver.New(driverConfig)
	outputs := helpers.DefaultOutputs(
		driverConfig.Output, driverConfig.Pkgname, config.NoFactory,
		&helpers.Templates{Models: []fs.FS{gen.MySQLModelTemplates}},
	)

	state := &gen.State{
		Config:  config,
		Outputs: outputs,
	}

	return gen.Run(c.Context, state, d)
}
