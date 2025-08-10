package plugins

import (
	"io/fs"

	"github.com/stephenafamo/bob/gen"
)

func Loaders[C any](templates ...fs.FS) gen.StatePlugin[C] {
	return loadersPlugin[C]{
		templates: templates,
	}
}

type loadersPlugin[C any] struct {
	templates []fs.FS
}

// Name implements gen.StatePlugin.
func (loadersPlugin[C]) Name() string {
	return "Loaders Output Plugin"
}

// PlugState implements gen.StatePlugin.
func (l loadersPlugin[C]) PlugState(state *gen.State[C]) error {
	if err := dependsOn(state, "models"); err != nil {
		return err
	}

	for _, output := range state.Outputs {
		if output.Key == "models" {
			output.Templates = append(output.Templates, gen.BaseTemplates.Loaders)
			output.Templates = append(output.Templates, l.templates...)
		}
	}

	return nil
}
