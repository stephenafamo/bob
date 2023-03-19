package helpers

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"runtime/debug"
	"strings"

	"github.com/knadh/koanf"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/stephenafamo/bob/gen"
	"github.com/stephenafamo/bob/orm"
)

const DefaultConfigPath = "./bobgen.yaml"

func Version() string {
	if info, ok := debug.ReadBuildInfo(); ok {
		return info.Main.Version
	}

	return ""
}

type Templates struct {
	Models  []fs.FS
	Factory []fs.FS
}

func DefaultOutputs(destination, pkgname string, noFactory bool, templates *Templates) []*gen.Output {
	if templates == nil {
		templates = &Templates{}
	}

	outputs := []*gen.Output{
		{
			Key:       "models",
			OutFolder: destination,
			PkgName:   pkgname,
			Templates: append(templates.Models, gen.ModelTemplates),
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

func GetConfigFromFile[DriverConfig any](configPath, driverConfigKey string) (gen.Config, DriverConfig, error) {
	var provider koanf.Provider
	var config gen.Config
	var driverConfig DriverConfig

	_, err := os.Stat(configPath)
	if err == nil {
		// set the provider if provided
		provider = file.Provider(configPath)
	}
	if err != nil && !(configPath == DefaultConfigPath && errors.Is(err, os.ErrNotExist)) {
		return config, driverConfig, err
	}

	return GetConfigFromProvider[DriverConfig](provider, driverConfigKey)
}

func GetConfigFromProvider[DriverConfig any](provider koanf.Provider, driverConfigKey string) (gen.Config, DriverConfig, error) {
	var config gen.Config
	var driverConfig DriverConfig

	k := koanf.New(".")

	// Add some defaults
	err := k.Load(confmap.Provider(map[string]any{
		"wipe":              true,
		"struct_tag_casing": "snake",
		"relation_tag":      "-",
		"generator":         fmt.Sprintf("BobGen %s %s", driverConfigKey, Version()),
	}, "."), nil)
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
			if rel.NoReverse || len(rel.Sides) < 1 {
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
