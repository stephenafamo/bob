package plugins

import (
	"fmt"

	"github.com/stephenafamo/bob/gen"
)

type Config struct {
	Enums    OutputConfig `yaml:"enums"`
	Models   OutputConfig `yaml:"models"`
	Factory  OutputConfig `yaml:"factory"`
	DBErrors OutputConfig `yaml:"dberrors"`
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
