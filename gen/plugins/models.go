package plugins

import (
	"io/fs"
	"path/filepath"

	"github.com/stephenafamo/bob/gen"
	"github.com/stephenafamo/bob/internal"
)

func Models[C any](config OutputConfig, templates ...fs.FS) gen.StatePlugin[C] {
	config = config.WithDefaults("models")
	return modelsPlugin[C]{
		config:    config,
		templates: templates,
	}
}

type modelsPlugin[C any] struct {
	config    OutputConfig
	templates []fs.FS
}

// Name implements gen.StatePlugin.
func (modelsPlugin[C]) Name() string {
	return "Models Output Plugin"
}

// PlugState implements gen.StatePlugin.
func (m modelsPlugin[C]) PlugState(state *gen.State[C]) error {
	err := dependsOn(m.config.Disabled, state, "enums")
	if internal.ValOrZero(m.config.Disabled) && err != nil {
		return err
	}

	state.Outputs = append(state.Outputs, &gen.Output{
		Disabled:  internal.ValOrZero(m.config.Disabled),
		Key:       "models",
		OutFolder: m.config.Destination,
		PkgName:   m.config.Pkgname,
		Templates: append(m.templates, gen.BaseTemplates.Models),
	})

	// To delete the old factory output if it exists
	state.Outputs = append(state.Outputs, &gen.Output{
		Disabled:  true,
		Key:       "old-factory",
		OutFolder: filepath.Join(m.config.Destination, "factory"),
	})

	return nil
}
