package helpers

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"os"
	"path"
	"runtime/debug"
	"strings"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
	"github.com/stephenafamo/bob/gen"
	"github.com/stephenafamo/bob/gen/drivers"
)

const DefaultConfigPath = "./bobgen.yaml"

func Version() string {
	if info, ok := debug.ReadBuildInfo(); ok {
		return info.Main.Version
	}

	return ""
}

type Config struct {
	// Which `database` driver to use (the full module name)
	Driver string `yaml:"driver"`
	// The database connection string
	Dsn string
	// List of tables that will be included. Others are ignored
	Only map[string][]string
	// Folders containing query files
	Queries []string `yaml:"queries"`
	// List of tables that will be should be ignored. Others are included
	Except map[string][]string

	//-------

	// The name of the folder to output the models package to
	Output string
	// The name you wish to assign to your generated models package
	Pkgname   string
	NoFactory bool `yaml:"no_factory"`
}

type Templates struct {
	Models  []fs.FS
	Factory []fs.FS
	Queries []fs.FS
}

func DefaultOutputs(destination, pkgname string, noFactory bool, templates *Templates) []*gen.Output {
	if templates == nil {
		templates = &Templates{}
	}

	if destination == "" {
		destination = "models"
	}

	if pkgname == "" {
		pkgname = "models"
	}

	outputs := []*gen.Output{
		{
			Key:                     "models",
			OutFolder:               destination,
			PkgName:                 pkgname,
			SeparatePackageForTests: true,
			Templates:               append(templates.Models, gen.ModelTemplates),
		},
		{
			Key:       "queries",
			Templates: append(templates.Queries, gen.QueriesTemplates),
		},
	}

	if !noFactory {
		outputs = append(outputs, &gen.Output{
			Key:       "factory",
			OutFolder: path.Join(destination, "factory"),
			PkgName:   "factory",
			Templates: append(templates.Factory, gen.FactoryTemplates),
		})
	}

	return outputs
}

func GetConfigFromFile[ConstraintExtra, DriverConfig any](configPath, driverConfigKey string) (gen.Config[ConstraintExtra], DriverConfig, error) {
	var provider koanf.Provider
	var config gen.Config[ConstraintExtra]
	var driverConfig DriverConfig

	_, err := os.Stat(configPath)
	if err == nil {
		// set the provider if provided
		provider = file.Provider(configPath)
	}
	if err != nil && (configPath != DefaultConfigPath || !errors.Is(err, os.ErrNotExist)) {
		return config, driverConfig, err
	}

	return GetConfigFromProvider[ConstraintExtra, DriverConfig](provider, driverConfigKey)
}

func GetConfigFromProvider[ConstraintExtra, DriverConfig any](provider koanf.Provider, driverConfigKey string) (gen.Config[ConstraintExtra], DriverConfig, error) {
	var config gen.Config[ConstraintExtra]
	var driverConfig DriverConfig

	k := koanf.New(".")

	// Add some defaults
	err := k.Load(confmap.Provider(map[string]any{
		"wipe":              true,
		"struct_tag_casing": "snake",
		"relation_tag":      "-",
		"generator":         fmt.Sprintf("BobGen %s %s", driverConfigKey, Version()),
	}, ""), nil)
	if err != nil {
		return config, driverConfig, err
	}

	if provider != nil {
		// Load YAML config and merge into the previously loaded config (because we can).
		err := k.Load(provider, yaml.Parser())
		if err != nil {
			return config, driverConfig, err
		}
	}

	// Load env variables for ONLY driver config
	envKey := strings.ToUpper(driverConfigKey) + "_"
	err = k.Load(env.Provider(envKey, ".", func(s string) string {
		// replace only the first underscore to make it a flat map[string]any
		return strings.Replace(strings.ToLower(s), "_", ".", 1)
	}), nil)
	if err != nil {
		return config, driverConfig, err
	}

	err = k.UnmarshalWithConf("", &config, koanf.UnmarshalConf{Tag: "yaml"})
	if err != nil {
		return config, driverConfig, err
	}

	err = k.UnmarshalWithConf(driverConfigKey, &driverConfig, koanf.UnmarshalConf{Tag: "yaml"})
	if err != nil {
		return config, driverConfig, err
	}

	return config, driverConfig, nil
}

func EnumType(types drivers.Types, enum string) string {
	types.Register(enum, drivers.Type{
		NoRandomizationTest: true, // enums are often not random enough
		RandomExpr: fmt.Sprintf(`all := all%s()
            return all[f.IntBetween(0, len(all)-1)]`, enum),
	})

	return enum
}

func GetFreePort() (int, error) {
	a, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return 0, fmt.Errorf("resolve localhost:0: %w", err)
	}

	l, err := net.ListenTCP("tcp", a)
	if err != nil {
		return 0, fmt.Errorf("listen on localhost:0: %w", err)
	}
	defer l.Close()

	return l.Addr().(*net.TCPAddr).Port, nil
}

func Migrate(ctx context.Context, db *sql.DB, dir fs.FS, pattern string) error {
	if dir == nil {
		dir = os.DirFS(".")
	}

	matchedFiles, err := fs.Glob(dir, pattern)
	if err != nil {
		return fmt.Errorf("globbing %s: %w", pattern, err)
	}

	for _, filePath := range matchedFiles {
		content, err := fs.ReadFile(dir, filePath)
		if err != nil {
			return fmt.Errorf("reading %s: %w", filePath, err)
		}

		fmt.Printf("migrating %s...\n", filePath)
		if _, err = db.ExecContext(ctx, string(content)); err != nil {
			return fmt.Errorf("migrating %s: %w", filePath, err)
		}
	}

	fmt.Printf("migrations finished\n")
	return nil
}
