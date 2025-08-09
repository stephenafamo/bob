package plugins

import (
	"io/fs"

	"github.com/stephenafamo/bob/gen"
)

func Enums[C any](config OutputConfig, templates ...fs.FS) gen.StatePlugin[C] {
	return enumsPlugin[C]{
		config:    config,
		templates: templates,
	}
}

type enumsPlugin[C any] struct {
	config    OutputConfig
	templates []fs.FS
}

// Name implements gen.StatePlugin.
func (e enumsPlugin[C]) Name() string {
	return "Enums Output Plugin"
}

// PlugState implements gen.StatePlugin.
func (e enumsPlugin[C]) PlugState(state *gen.State[C]) error {
	if err := e.config.Validate(); err != nil {
		return err
	}

	state.Outputs = append(state.Outputs, &gen.Output{
		Disabled:  e.config.Disabled,
		Key:       "enums",
		OutFolder: e.config.Destination,
		PkgName:   e.config.Pkgname,
		Templates: append(e.templates, gen.EnumTemplates),
	})

	return nil
}
