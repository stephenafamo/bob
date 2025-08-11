package plugins

import (
	"fmt"

	"github.com/stephenafamo/bob/gen"
)

func All[T, C, I any](config Config, templates gen.Templates) []gen.Plugin {
	return []gen.Plugin{
		Enums[T, C, I](config.Enums, templates.Enums),
		Models[C](config.Models, templates.Models),
		Factory[C](config.Factory, templates.Factory),
		DBErrors[C](config.DBErrors, templates.DBErrors),
		Joins[C](config.Joins, templates.Joins),
		Loaders[C](config.Loaders, templates.Loaders),
		Queries[C](templates.Queries),
	}
}

type Config struct {
	Enums    OutputConfig `yaml:"enums"`
	Models   OutputConfig `yaml:"models"`
	Factory  OutputConfig `yaml:"factory"`
	DBErrors OutputConfig `yaml:"dberrors"`
	Joins    OnOffConfig  `yaml:"joins"`
	Loaders  OnOffConfig  `yaml:"loaders"`
}

type OnOffConfig struct {
	Disabled bool `yaml:"disabled"`
}

type OutputConfig struct {
	Disabled    bool   `yaml:"disabled"`
	Destination string `yaml:"destination"`
	Pkgname     string `yaml:"pkgname"`
}

func (o OutputConfig) Validate() error {
	if o.Destination != "" && o.Pkgname != "" {
		return nil
	}

	return fmt.Errorf("output config must have both destination and pkgname set, got: destination=%s, pkgname=%s", o.Destination, o.Pkgname)
}

func dependsOn[C any](state *gen.State[C], keys ...string) error {
Outer:
	for _, key := range keys {
		for _, output := range state.Outputs {
			if output.Key == key && !output.Disabled {
				continue Outer
			}
		}
		return fmt.Errorf("the %s output needs to be present and enabled", key)
	}

	return nil
}
