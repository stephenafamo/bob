package plugins

import (
	"cmp"
	"fmt"

	"github.com/stephenafamo/bob/gen"
	"github.com/stephenafamo/bob/internal"
)

func Setup[T, C, I any](config Config, templates gen.Templates) []gen.Plugin {
	return []gen.Plugin{
		Enums[T, C, I](config.Enums, templates.Enums),
		Models[C](config.Models, templates.Models),
		Factory[C](config.Factory, templates.Factory),
		DBErrors[C](config.DBErrors, templates.DBErrors),
		Where[C](config.Where, templates.Where),
		Loaders[C](config.Loaders, templates.Loaders),
		Joins[C](config.Joins, templates.Joins),
		Names[C](config.Names, templates.Names),
		Queries[T, C, I](templates.Queries),
	}
}

type Config struct {
	Enums    OutputConfig `yaml:"enums"`
	Models   OutputConfig `yaml:"models"`
	Factory  OutputConfig `yaml:"factory"`
	DBErrors OutputConfig `yaml:"dberrors"`
	Where    OnOffConfig  `yaml:"where"`
	Loaders  OnOffConfig  `yaml:"loaders"`
	Joins    OnOffConfig  `yaml:"joins"`
	Names    OnOffConfig  `yaml:"constants,omitempty"`
}

func (c Config) Merge(c2 Config) Config {
	return Config{
		Enums:    mergeOutputConfig(c.Enums, c2.Enums),
		Models:   mergeOutputConfig(c.Models, c2.Models),
		Factory:  mergeOutputConfig(c.Factory, c2.Factory),
		DBErrors: mergeOutputConfig(c.DBErrors, c2.DBErrors),
		Where:    mergeOnOffConfig(c.Where, c2.Where),
		Loaders:  mergeOnOffConfig(c.Loaders, c2.Loaders),
		Joins:    mergeOnOffConfig(c.Joins, c2.Joins),
		Names:    mergeOnOffConfig(c.Names, c2.Names),
	}
}

func mergeOnOffConfig(c1, c2 OnOffConfig) OnOffConfig {
	return OnOffConfig{
		Disabled: cmp.Or(c2.Disabled, c1.Disabled), // The new disabled state takes precedence
	}
}

type OnOffConfig struct {
	Disabled *bool `yaml:"disabled"`
}

func mergeOutputConfig(c1, c2 OutputConfig) OutputConfig {
	return OutputConfig{
		Disabled:    cmp.Or(c2.Disabled, c1.Disabled),
		Destination: cmp.Or(c2.Destination, c1.Destination),
		Pkgname:     cmp.Or(c2.Pkgname, c1.Pkgname),
	}
}

type OutputConfig struct {
	Disabled    *bool  `yaml:"disabled"`
	Destination string `yaml:"destination"`
	Pkgname     string `yaml:"pkgname"`
}

func (o OutputConfig) WithDefaults(name string) OutputConfig {
	if o.Destination == "" {
		o.Destination = name
	}
	if o.Pkgname == "" {
		o.Pkgname = name
	}
	return o
}

func dependsOn[C any](disabled *bool, state *gen.State[C], keys ...string) error {
	if internal.ValOrZero(disabled) {
		return nil
	}

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
