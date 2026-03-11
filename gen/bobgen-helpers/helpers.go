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
	}, ""), nil)
	if err != nil {
		return config, driverConfig, pluginsConfig, fmt.Errorf("failed to load defaults: %w", err)
	}

	if provider != nil {
		// Load YAML config and merge into the previously loaded config (because we can).
		err := k.Load(provider, yaml.Parser())
		if err != nil {
			return config, driverConfig, pluginsConfig, fmt.Errorf("failed to load config from %s: %w", provider, err)
		}
	}

	// Load env variables for ONLY driver config
	envKey := strings.ToUpper(driverConfigKey) + "_"
	err = k.Load(env.Provider(envKey, ".", func(s string) string {
		// replace only the first underscore to make it a flat map[string]any
		return strings.Replace(strings.ToLower(s), "_", ".", 1)
	}), nil)
	if err != nil {
		return config, driverConfig, pluginsConfig, fmt.Errorf("failed to load env variables with prefix %s: %w", envKey, err)
	}

	err = k.UnmarshalWithConf("", &config, koanf.UnmarshalConf{Tag: "yaml"})
	if err != nil {
		return config, driverConfig, pluginsConfig, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	err = k.UnmarshalWithConf(driverConfigKey, &driverConfig, koanf.UnmarshalConf{Tag: "yaml"})
	if err != nil {
		return config, driverConfig, pluginsConfig, fmt.Errorf("failed to unmarshal driver config: %w", err)
	}

	err = k.UnmarshalWithConf("plugins", &pluginsConfig, koanf.UnmarshalConf{Tag: "yaml"})
	if err != nil {
		return config, driverConfig, pluginsConfig, fmt.Errorf("failed to unmarshal plugins config: %w", err)
	}

	switch k.String("plugins_preset") {
	case "all":
		pluginsConfig = plugins.PresetAll.Merge(pluginsConfig)
	case "none":
		pluginsConfig = plugins.PresetNone.Merge(pluginsConfig)
	default:
		pluginsConfig = plugins.PresetDefault.Merge(pluginsConfig)
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

type MigrateOption func(*migrateConfig)

type migrateConfig struct {
	noTxPattern string
}

func WithNoTransactionPattern(pattern string) MigrateOption {
	return func(c *migrateConfig) {
		c.noTxPattern = pattern
	}
}

func Migrate(ctx context.Context, db *sql.DB, dir fs.FS, pattern string, opts ...MigrateOption) error {
	var cfg migrateConfig
	for _, o := range opts {
		o(&cfg)
	}

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

		if cfg.noTxPattern != "" && strings.Contains(string(content), cfg.noTxPattern) {
			for _, stmt := range splitStatements(string(content)) {
				if _, err = db.ExecContext(ctx, stmt); err != nil {
					return fmt.Errorf("migrating %s: %w", filePath, err)
				}
			}
			continue
		}

		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("beginning transaction for %s: %w", filePath, err)
		}

		if _, err = tx.ExecContext(ctx, string(content)); err != nil {
			tx.Rollback() //nolint:errcheck
			return fmt.Errorf("migrating %s: %w", filePath, err)
		}

		if err = tx.Commit(); err != nil {
			return fmt.Errorf("committing transaction for %s: %w", filePath, err)
		}
	}

	fmt.Printf("migrations finished\n")
	return nil
}

// splitStatements splits SQL content into individual statements by splitting
// on top-level semicolons. It correctly handles dollar-quoted strings
// (e.g. $$ ... $$ or $tag$ ... $tag$), single-quoted strings, line comments
// (--), and block comments (/* */).
func splitStatements(sql string) []string {
	var stmts []string
	start := 0
	i := 0

	for i < len(sql) {
		switch {
		// Line comment: skip to end of line
		case sql[i] == '-' && i+1 < len(sql) && sql[i+1] == '-':
			i += 2
			for i < len(sql) && sql[i] != '\n' {
				i++
			}

		// Block comment: skip to closing */
		case sql[i] == '/' && i+1 < len(sql) && sql[i+1] == '*':
			i += 2
			for i+1 < len(sql) && !(sql[i] == '*' && sql[i+1] == '/') {
				i++
			}
			if i+1 < len(sql) {
				i += 2 // skip */
			}

		// Single-quoted string: skip to closing quote, handling '' escapes
		case sql[i] == '\'':
			i++
			for i < len(sql) {
				if sql[i] == '\'' {
					if i+1 < len(sql) && sql[i+1] == '\'' {
						i += 2 // escaped quote
						continue
					}
					break
				}
				i++
			}
			if i < len(sql) {
				i++ // skip closing quote
			}

		// Dollar-quoted string: find the tag and skip to matching close tag
		case sql[i] == '$':
			tagStart := i
			i++
			// Read the tag name (alphanumeric/underscore between the two $)
			for i < len(sql) && sql[i] != '$' && sql[i] != ' ' && sql[i] != '\n' && sql[i] != ';' {
				i++
			}
			if i < len(sql) && sql[i] == '$' {
				tag := sql[tagStart : i+1] // e.g. "$$" or "$tag$"
				i++                         // move past closing $ of open tag
				// Find the matching close tag
				closeIdx := strings.Index(sql[i:], tag)
				if closeIdx >= 0 {
					i += closeIdx + len(tag)
				}
			}

		// Semicolon: end of statement
		case sql[i] == ';':
			stmt := strings.TrimSpace(sql[start : i+1])
			if stmt != "" && stmt != ";" {
				stmts = append(stmts, stmt)
			}
			i++
			start = i

		default:
			i++
		}
	}

	// Handle any trailing content without a final semicolon
	if remaining := strings.TrimSpace(sql[start:]); remaining != "" {
		stmts = append(stmts, remaining)
	}

	return stmts
}
