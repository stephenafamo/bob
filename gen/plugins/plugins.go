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
		Joins[C](config.Joins, templates.Joins),
		Loaders[C](config.Loaders, templates.Loaders),
		Queries[T, C, I](templates.Queries),
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

func (c Config) Merge(c2 Config) Config {
	return Config{
		Enums:    mergeOutputConfig(c.Enums, c2.Enums),
		Models:   mergeOutputConfig(c.Models, c2.Models),
		Factory:  mergeOutputConfig(c.Factory, c2.Factory),
		DBErrors: mergeOutputConfig(c.DBErrors, c2.DBErrors),
		Joins:    mergeOnOffConfig(c.Joins, c2.Joins),
		Loaders:  mergeOnOffConfig(c.Loaders, c2.Loaders),
	}
}

//nolint:gochecknoglobals
var PresetAll = Config{
	Enums:    OutputConfig{Destination: "enums", Pkgname: "enums"},
	Models:   OutputConfig{Destination: "models", Pkgname: "models"},
	Factory:  OutputConfig{Destination: "factory", Pkgname: "factory"},
	DBErrors: OutputConfig{Destination: "dberrors", Pkgname: "dberrors"},
	Joins:    OnOffConfig{}, // Joins are enabled by default
	Loaders:  OnOffConfig{}, // Loaders are enabled by default
}

//nolint:gochecknoglobals
var PresetNone = PresetAll.Merge(Config{
	Enums:    OutputConfig{Disabled: internal.Pointer(true)},
	Models:   OutputConfig{Disabled: internal.Pointer(true)},
	Factory:  OutputConfig{Disabled: internal.Pointer(true)},
	DBErrors: OutputConfig{Disabled: internal.Pointer(true)},
	Joins:    OnOffConfig{Disabled: internal.Pointer(true)}, // Joins are disabled
	Loaders:  OnOffConfig{Disabled: internal.Pointer(true)}, // Loaders are disabled
})

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

func (o OutputConfig) Validate() error {
	if internal.ValOrZero(o.Disabled) {
		return nil
	}

	if o.Destination != "" && o.Pkgname != "" {
		return nil
	}

	return fmt.Errorf("output config must have both destination and pkgname set, got: destination=%s, pkgname=%s", o.Destination, o.Pkgname)
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
