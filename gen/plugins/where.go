package plugins

import (
	"io/fs"

	"github.com/stephenafamo/bob/gen"
)

func Where[C any](config OnOffConfig, templates ...fs.FS) gen.StatePlugin[C] {
	return wherePlugin[C]{
		disabled:  config.Disabled,
		templates: templates,
	}
}

type wherePlugin[C any] struct {
	disabled  *bool
	templates []fs.FS
}

// Name implements gen.StatePlugin.
func (wherePlugin[C]) Name() string {
	return "Where Output Plugin"
}

// PlugState implements gen.StatePlugin.
func (l wherePlugin[C]) PlugState(state *gen.State[C]) error {
	if err := dependsOn(l.disabled, state, "models"); err != nil {
		return err
	}

	for _, output := range state.Outputs {
		if output.Key == "models" {
			output.Templates = append(output.Templates, gen.BaseTemplates.Where)
			output.Templates = append(output.Templates, l.templates...)
			break
		}
	}

	return nil
}
