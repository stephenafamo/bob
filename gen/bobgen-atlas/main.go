package main

import (
	"context"
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/signal"
	"path"
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

	modelTemplates := []fs.FS{gen.ModelTemplates}
	if driverConfig.Dialect == "mysql" {
		modelTemplates = append(modelTemplates, gen.MySQLModelTemplates)
	}

	outputs := []*gen.Output{
		{
			OutFolder: driverConfig.Output,
			PkgName:   driverConfig.Pkgname,
			Templates: modelTemplates,
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

	d := driver.New(driverConfig, os.DirFS(driverConfig.Dir))

	cmdState := &gen.State[any]{
		Config:    &config,
		Dialect:   driverConfig.Dialect,
		Outputs:   outputs,
		ModelsPkg: modPkg,
	}

	return cmdState.Run(c.Context, d)
}
