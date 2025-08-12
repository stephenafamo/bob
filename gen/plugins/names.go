package plugins

import (
	"io/fs"

	"github.com/stephenafamo/bob/gen"
	"github.com/stephenafamo/bob/internal"
)

func Names[C any](config OnOffConfig, templates ...fs.FS) gen.StatePlugin[C] {
	return namesPlugin[C]{
		disabled:  config.Disabled,
		templates: templates,
	}
}

type namesPlugin[C any] struct {
	disabled  *bool
	templates []fs.FS
}

// Name implements gen.StatePlugin.
func (namesPlugin[C]) Name() string {
	return "Names Output Plugin"
}

// PlugState implements gen.StatePlugin.
func (n namesPlugin[C]) PlugState(state *gen.State[C]) error {
	if err := dependsOn(n.disabled, state, "models"); err != nil {
		return err
	}

	if internal.ValOrZero(n.disabled) {
		return nil
	}

	for _, output := range state.Outputs {
		if output.Key == "models" {
			output.Templates = append(output.Templates, gen.BaseTemplates.Names)
			output.Templates = append(output.Templates, n.templates...)
			break
		}
	}

	return nil
}
