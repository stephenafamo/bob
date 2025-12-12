package plugins

import (
	"io/fs"

	"github.com/stephenafamo/bob/gen"
	"github.com/stephenafamo/bob/internal"
)

func Counts[C any](config OnOffConfig, templates ...fs.FS) gen.StatePlugin[C] {
	return countsPlugin[C]{
		disabled:  config.Disabled,
		templates: templates,
	}
}

type countsPlugin[C any] struct {
	disabled  *bool
	templates []fs.FS
}

// Name implements gen.StatePlugin.
func (countsPlugin[C]) Name() string {
	return "Counts Output Plugin"
}

// PlugState implements gen.StatePlugin.
func (c countsPlugin[C]) PlugState(state *gen.State[C]) error {
	if err := dependsOn(c.disabled, state, "models"); err != nil {
		return err
	}

	if internal.ValOrZero(c.disabled) {
		return nil
	}

	for _, output := range state.Outputs {
		if output.Key == "models" {
			output.Templates = append(output.Templates, gen.BaseTemplates.Counts)
			output.Templates = append(output.Templates, c.templates...)
			break
		}
	}

	return nil
}
