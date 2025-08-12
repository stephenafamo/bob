package plugins

import (
	"io/fs"

	"github.com/stephenafamo/bob/gen"
	"github.com/stephenafamo/bob/internal"
)

func Joins[C any](config OnOffConfig, templates ...fs.FS) gen.StatePlugin[C] {
	return joinsPlugin[C]{
		disabled:  config.Disabled,
		templates: templates,
	}
}

type joinsPlugin[C any] struct {
	disabled  *bool
	templates []fs.FS
}

// Name implements gen.StatePlugin.
func (joinsPlugin[C]) Name() string {
	return "Joins Output Plugin"
}

// PlugState implements gen.StatePlugin.
func (j joinsPlugin[C]) PlugState(state *gen.State[C]) error {
	if err := dependsOn(j.disabled, state, "models"); err != nil {
		return err
	}

	if internal.ValOrZero(j.disabled) {
		return nil
	}

	for _, output := range state.Outputs {
		if output.Key == "models" {
			output.Templates = append(output.Templates, gen.BaseTemplates.Joins)
			output.Templates = append(output.Templates, j.templates...)
			break
		}
	}

	return nil
}
