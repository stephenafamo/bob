package plugins

import (
	"io/fs"

	"github.com/stephenafamo/bob/gen"
	"github.com/stephenafamo/bob/internal"
)

func Factory[C any](config OutputConfig, templates ...fs.FS) gen.StatePlugin[C] {
	return factoryPlugin[C]{
		config:    config,
		templates: templates,
	}
}

type factoryPlugin[C any] struct {
	config    OutputConfig
	templates []fs.FS
}

// Name implements gen.StatePlugin.
func (factoryPlugin[C]) Name() string {
	return "Factory Output Plugin"
}

// PlugState implements gen.StatePlugin.
func (f factoryPlugin[C]) PlugState(state *gen.State[C]) error {
	if err := dependsOn(f.config.Disabled, state, "enums", "models"); err != nil {
		return err
	}

	if err := f.config.Validate(); err != nil {
		return err
	}

	state.Outputs = append(state.Outputs, &gen.Output{
		Disabled:  internal.ValOrZero(f.config.Disabled),
		Key:       "factory",
		OutFolder: f.config.Destination,
		PkgName:   f.config.Pkgname,
		Templates: append(f.templates, gen.BaseTemplates.Factory),
	})

	return nil
}
