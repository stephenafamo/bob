package plugins

import (
	"io/fs"

	"github.com/stephenafamo/bob/gen"
	"github.com/stephenafamo/bob/internal"
)

func Loaders[C any](config OnOffConfig, templates ...fs.FS) gen.StatePlugin[C] {
	return loadersPlugin[C]{
		disabled:  config.Disabled,
		templates: templates,
	}
}

type loadersPlugin[C any] struct {
	disabled  *bool
	templates []fs.FS
}

// Name implements gen.StatePlugin.
func (loadersPlugin[C]) Name() string {
	return "Loaders Output Plugin"
}

// PlugState implements gen.StatePlugin.
func (l loadersPlugin[C]) PlugState(state *gen.State[C]) error {
	if err := dependsOn(l.disabled, state, "models"); err != nil {
		return err
	}

	if internal.ValOrZero(l.disabled) {
		return nil
	}

	for _, output := range state.Outputs {
		if output.Key == "models" {
			output.Templates = append(output.Templates, gen.BaseTemplates.Loaders)
			output.Templates = append(output.Templates, l.templates...)
			break
		}
	}

	return nil
}
