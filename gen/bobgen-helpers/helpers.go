package helpers

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"runtime/debug"
	"strings"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
	"github.com/stephenafamo/bob/gen"
	"github.com/stephenafamo/bob/gen/drivers"
	"github.com/stephenafamo/bob/gen/plugins"
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
}

func OutputPlugins[T, C, I any](config plugins.Config, templates gen.Templates) []gen.Plugin {
	return []gen.Plugin{
		plugins.Enums[T, C, I](config.Enums, templates.Enums),
		plugins.Models[C](config.Models, templates.Models),
		plugins.Factory[C](config.Factory, templates.Factory),
		plugins.Queries[C](templates.Queries),
		plugins.DBErrors[C](config.DBErrors, templates.DBErrors),
		plugins.Joins[C](templates.Joins),
		plugins.Loaders[C](templates.Loaders),
	}
}

func GetConfigFromFile[ConstraintExtra, DriverConfig any](configPath, driverConfigKey string) (gen.Config[ConstraintExtra], DriverConfig, plugins.Config, error) {
	var provider koanf.Provider
	var config gen.Config[ConstraintExtra]
	var driverConfig DriverConfig

	_, err := os.Stat(configPath)
	if err == nil {
		// set the provider if provided
		provider = file.Provider(configPath)
	}
	if err != nil && (configPath != DefaultConfigPath || !errors.Is(err, os.ErrNotExist)) {
		return config, driverConfig, plugins.Config{}, err
	}

	return GetConfigFromProvider[ConstraintExtra, DriverConfig](provider, driverConfigKey)
}

func GetConfigFromProvider[ConstraintExtra, DriverConfig any](provider koanf.Provider, driverConfigKey string) (gen.Config[ConstraintExtra], DriverConfig, plugins.Config, error) {
	var config gen.Config[ConstraintExtra]
	var driverConfig DriverConfig
	var pluginsConfig plugins.Config

	k := koanf.New(".")

	// Add some defaults
	err := k.Load(confmap.Provider(map[string]any{
		"struct_tag_casing": "snake",
		"relation_tag":      "-",
		"generator":         fmt.Sprintf("BobGen %s %s", driverConfigKey, Version()),
		"plugins": map[string]any{
			"enums": map[string]any{
				"destination": "enums",
				"pkgname":     "enums",
			},
			"models": map[string]any{
				"destination": "models",
				"pkgname":     "models",
			},
			"factory": map[string]any{
				"destination": "factory",
				"pkgname":     "factory",
			},
			"dberrors": map[string]any{
				"destination": "dberrors",
				"pkgname":     "dberrors",
			},
		},
	}, ""), nil)
	if err != nil {
		return config, driverConfig, pluginsConfig, err
	}

	if provider != nil {
		// Load YAML config and merge into the previously loaded config (because we can).
		err := k.Load(provider, yaml.Parser())
		if err != nil {
			return config, driverConfig, pluginsConfig, err
		}
	}

	// Load env variables for ONLY driver config
	envKey := strings.ToUpper(driverConfigKey) + "_"
	err = k.Load(env.Provider(envKey, ".", func(s string) string {
		// replace only the first underscore to make it a flat map[string]any
		return strings.Replace(strings.ToLower(s), "_", ".", 1)
	}), nil)
	if err != nil {
		return config, driverConfig, pluginsConfig, err
	}

	err = k.UnmarshalWithConf("", &config, koanf.UnmarshalConf{Tag: "yaml"})
	if err != nil {
		return config, driverConfig, pluginsConfig, err
	}

	err = k.UnmarshalWithConf(driverConfigKey, &driverConfig, koanf.UnmarshalConf{Tag: "yaml"})
	if err != nil {
		return config, driverConfig, pluginsConfig, err
	}

	err = k.UnmarshalWithConf("plugins", &pluginsConfig, koanf.UnmarshalConf{Tag: "yaml"})
	if err != nil {
		return config, driverConfig, pluginsConfig, err
	}

	return config, driverConfig, pluginsConfig, nil
}

func EnumType(types drivers.Types, enum string) string {
	fullTyp := fmt.Sprintf("enums.%s", enum)
	types.Register(fullTyp, drivers.Type{
		NoRandomizationTest: true, // enums are often not random enough
		Imports:             []string{"output(enums)"},
		RandomExpr: `var e BASETYPE
			all := e.All()
			return all[f.IntBetween(0, len(all)-1)]`,
	})

	return fullTyp
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
