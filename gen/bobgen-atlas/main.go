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
	configFile := c.String("config")

	config, driverConfig, err := helpers.GetConfig[driver.Config](configFile, "atlas", map[string]any{
		"dir":        ".",
		"output":     "models",
		"pkgname":    "models",
		"no_factory": false,
	})
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

	cmdState := &gen.State[any]{
		Config:            &config,
		Dialect:           driverConfig.Dialect,
		DestinationFolder: driverConfig.Output,
		ModelsPkgName:     driverConfig.Pkgname,
		Templates: gen.Templates{
			Models: modelTemplates,
		},
	}

	return cmdState.Run(c.Context, d)
}
