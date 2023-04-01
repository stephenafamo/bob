package main

import (
	"context"
	"io/fs"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/stephenafamo/bob/gen"
	"github.com/stephenafamo/bob/gen/bobgen-atlas/driver"
	helpers "github.com/stephenafamo/bob/gen/bobgen-helpers"
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
		Name:      "bobgen-atlas",
		Usage:     "Generate models and factories from your Atlas schema files",
		UsageText: "bobgen-atlas [-c FILE]",
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
	config, driverConfig, err := helpers.GetConfigFromFile[driver.Config](c.String("config"), "atlas")
	if err != nil {
		return err
	}

	var modelTemplates []fs.FS
	switch driverConfig.Dialect {
	case "mysql":
		modelTemplates = append(modelTemplates, gen.MySQLModelTemplates)
	case "sqlite":
		modelTemplates = append(modelTemplates, gen.SQLiteModelTemplates)
	}

	d := driver.New(driverConfig, os.DirFS(driverConfig.Dir))
	outputs := helpers.DefaultOutputs(
		driverConfig.Output, driverConfig.Pkgname, config.NoFactory,
		&helpers.Templates{Models: modelTemplates},
	)

	state := &gen.State{
		Config:  config,
		Outputs: outputs,
	}

	return gen.Run(c.Context, state, d)
}
