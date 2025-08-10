package plugins

import (
	"io/fs"

	"github.com/stephenafamo/bob/gen"
)

func Queries[C any](templates ...fs.FS) gen.StatePlugin[C] {
	return queriesOutputPlugin[C]{
		templates: templates,
	}
}

type queriesOutputPlugin[C any] struct {
	templates []fs.FS
}

// Name implements gen.StatePlugin.
func (q queriesOutputPlugin[C]) Name() string {
	return "Queries Output Plugin"
}

// PlugState implements gen.StatePlugin.
func (q queriesOutputPlugin[C]) PlugState(state *gen.State[C]) error {
	if err := dependsOn(state, "enums"); err != nil {
		return err
	}

	state.Outputs = append(state.Outputs, &gen.Output{
		Key:       "queries",
		Templates: append(q.templates, gen.BaseTemplates.Queries),
	})

	return nil
}
