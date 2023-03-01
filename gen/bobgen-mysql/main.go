package main

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/signal"
	"path"
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
	configFile := c.String("config")

	config, driverConfig, err := helpers.GetConfig[driver.Config](configFile, "mysql", map[string]any{
		"output":      "models",
		"pkgname":     "models",
		"no_factory":  false,
		"concurrency": 10,
	})
	if err != nil {
		return err
	}

	if driverConfig.Dsn == "" {
		return errors.New("database dsn is not set")
	}

	outputs := []*gen.Output{
		{
			OutFolder: driverConfig.Output,
			PkgName:   driverConfig.Pkgname,
			Templates: []fs.FS{gen.ModelTemplates, gen.MySQLModelTemplates},
		},
	}

	if !config.NoFactory {
		outputs = append(outputs, &gen.Output{
			OutFolder: path.Join(driverConfig.Output, "factory"),
			PkgName:   "factory",
			Templates: []fs.FS{gen.FactoryTemplates},
		})
	}

	modPkg, err := helpers.ModelsPackage(driverConfig.Output)
	if err != nil {
		return fmt.Errorf("getting models pkg details: %w", err)
	}

	d := driver.New(driverConfig)

	cmdState := &gen.State[any]{
		Config:    &config,
		Dialect:   "mysql",
		Outputs:   outputs,
		ModelsPkg: modPkg,
	}

	return cmdState.Run(c.Context, d)
}
