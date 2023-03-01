package helpers

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime/debug"
	"strings"

	"github.com/knadh/koanf"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/stephenafamo/bob/gen"
	"github.com/stephenafamo/bob/orm"
	"golang.org/x/mod/modfile"
)

const DefaultConfigPath = "./bobgen.yaml"

func GetConfig[T any](configPath, driverConfigKey string, driverDefaults map[string]any) (gen.Config, T, error) {
	var config gen.Config
	var driverConfig T

	k := koanf.New(".")

	// Add some defaults
	if err := k.Load(confmap.Provider(map[string]any{
		"wipe":              true,
		"struct_tag_casing": "snake",
		"relation_tag":      "-",
		"generator":         fmt.Sprintf("BobGen %s %s", driverConfigKey, Version()),
		driverConfigKey:     (any)(driverDefaults),
	}, "."), nil); err != nil {
		return config, driverConfig, err
	}

	if configPath != "" {
		// Load YAML config and merge into the previously loaded config (because we can).
		err := k.Load(file.Provider(configPath), yaml.Parser())
		if err != nil {
			if !(configPath == DefaultConfigPath && errors.Is(err, os.ErrNotExist)) {
				return config, driverConfig, err
			}
		}
	}

	// Load env variables for ONLY driver config
	envKey := strings.ToUpper(driverConfigKey) + "_"
	if err := k.Load(env.Provider(envKey, ".", func(s string) string {
		// replace only the first underscore to make it a flat map[string]any
		return strings.Replace(strings.ToLower(s), "_", ".", 1)
	}), nil); err != nil {
		return config, driverConfig, err
	}

	if err := k.UnmarshalWithConf("", &config, koanf.UnmarshalConf{Tag: "yaml"}); err != nil {
		return config, driverConfig, err
	}

	if err := k.UnmarshalWithConf(driverConfigKey, &driverConfig, koanf.UnmarshalConf{Tag: "yaml"}); err != nil {
		return config, driverConfig, err
	}

	setColumns(&config)
	flipRelationships(&config)
	return config, driverConfig, nil
}

func setColumns(c *gen.Config) {
	for table, rels := range c.Relationships {
		for relIdx, rel := range rels {
			for sideIdx, side := range rel.Sides {
				c.Relationships[table][relIdx].Sides[sideIdx].FromColumns = make([]string, len(side.Columns))
				c.Relationships[table][relIdx].Sides[sideIdx].ToColumns = make([]string, len(side.Columns))
				for colIndex, colpairs := range side.Columns {
					c.Relationships[table][relIdx].Sides[sideIdx].FromColumns[colIndex] = colpairs[0]
					c.Relationships[table][relIdx].Sides[sideIdx].ToColumns[colIndex] = colpairs[1]
				}
			}
		}
	}
}

func flipRelationships(config *gen.Config) {
	for _, rels := range config.Relationships {
		for _, rel := range rels {
			if len(rel.Sides) < 1 {
				continue
			}
			ftable := rel.Sides[len(rel.Sides)-1].To
			config.Relationships[ftable] = append(
				config.Relationships[ftable], flipRelationship(rel),
			)
		}
	}
}

func flipRelationship(r orm.Relationship) orm.Relationship {
	sideLen := len(r.Sides)
	flipped := orm.Relationship{
		Name:        r.Name,
		ByJoinTable: r.ByJoinTable,
		Ignored:     r.Ignored,
		Sides:       make([]orm.RelSide, sideLen),
	}

	for i, side := range r.Sides {
		flippedSide := orm.RelSide{
			To:   side.From,
			From: side.To,

			ToColumns:   side.FromColumns,
			FromColumns: side.ToColumns,
			ToWhere:     side.FromWhere,
			FromWhere:   side.ToWhere,

			ToKey:       !side.ToKey,
			ToUnique:    !side.ToUnique, // Assumption. Overwrite if necessary
			KeyNullable: side.KeyNullable,
		}
		flipped.Sides[sideLen-(1+i)] = flippedSide
	}

	return flipped
}

func Version() string {
	if info, ok := debug.ReadBuildInfo(); ok {
		return info.Main.Version
	}

	return ""
}

func ModelsPackage(modelsFolder string) (string, error) {
	modRoot, modFile, err := goModInfo()
	if err != nil {
		return "", fmt.Errorf("getting mod details: %w", err)
	}

	fullPath := modelsFolder
	if !filepath.IsAbs(modelsFolder) {
		wd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("could not get working directory: %w", err)
		}

		fullPath = filepath.Join(wd, modelsFolder)
	}

	relPath := strings.TrimPrefix(fullPath, modRoot)
	if fullPath == relPath {
		return "", fmt.Errorf("output folder is not in same module: %w", err)
	}

	return path.Join(modFile.Module.Mod.Path, relPath), nil
}

// goModInfo returns the main module's root directory
// and the parsed contents of the go.mod file.
func goModInfo() (string, *modfile.File, error) {
	goModPath, err := findGoMod()
	if err != nil {
		return "", nil, fmt.Errorf("cannot find main module: %w", err)
	}

	data, err := os.ReadFile(goModPath)
	if err != nil {
		return "", nil, fmt.Errorf("cannot read main go.mod file: %w", err)
	}

	modf, err := modfile.Parse(goModPath, data, nil)
	if err != nil {
		return "", nil, fmt.Errorf("could not parse go.mod: %w", err)
	}

	return filepath.Dir(goModPath), modf, nil
}

func findGoMod() (string, error) {
	var outData, errData bytes.Buffer

	c := exec.Command("go", "env", "GOMOD")
	c.Stdout = &outData
	c.Stderr = &errData
	c.Dir = "."
	err := c.Run()
	if err != nil {
		var exitError *exec.ExitError
		if errors.As(err, &exitError) && errData.Len() > 0 {
			return "", errors.New(strings.TrimSpace(errData.String()))
		}

		return "", fmt.Errorf("cannot run go env GOMOD: %w", err)
	}

	out := strings.TrimSpace(outData.String())
	if out == "" {
		return "", errors.New("no go.mod file found in any parent directory")
	}

	return out, nil
}
