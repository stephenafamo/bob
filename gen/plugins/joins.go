package plugins

import (
	"io/fs"

	"github.com/stephenafamo/bob/gen"
)

func Joins[C any](templates ...fs.FS) gen.StatePlugin[C] {
	return joinsPlugin[C]{
		templates: templates,
	}
}

type joinsPlugin[C any] struct {
	templates []fs.FS
}

// Name implements gen.StatePlugin.
func (joinsPlugin[C]) Name() string {
	return "Joins Output Plugin"
}

// PlugState implements gen.StatePlugin.
func (j joinsPlugin[C]) PlugState(state *gen.State[C]) error {
	if err := dependsOn(state, "models"); err != nil {
		return err
	}

	for _, output := range state.Outputs {
		if output.Key == "models" {
			output.Templates = append(output.Templates, gen.BaseTemplates.Joins)
			output.Templates = append(output.Templates, j.templates...)
		}
	}

	// For debugging
	state.Outputs = append(state.Outputs, &gen.Output{
		Key:       "joins",
		OutFolder: "debug/joins",
		PkgName:   "joins",
		Templates: append(j.templates, gen.BaseTemplates.Joins),
	})

	return nil
}
