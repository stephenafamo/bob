package plugins

import (
	"io/fs"

	"github.com/stephenafamo/bob/gen"
)

func DBErrors[C any](config OutputConfig, templates ...fs.FS) gen.StatePlugin[C] {
	return dbErrorsPlugin[C]{
		config:    config,
		templates: templates,
	}
}

type dbErrorsPlugin[C any] struct {
	config    OutputConfig
	templates []fs.FS
}

// Name implements gen.StatePlugin.
func (dbErrorsPlugin[C]) Name() string {
	return "DB Errors Output Plugin"
}

// PlugState implements gen.StatePlugin.
func (d dbErrorsPlugin[C]) PlugState(state *gen.State[C]) error {
	if err := dependsOn(state, "models", "factory"); err != nil {
		return err
	}

	if err := d.config.Validate(); err != nil {
		return err
	}

	state.Outputs = append(state.Outputs, &gen.Output{
		Disabled:  d.config.Disabled,
		Key:       "dberrors",
		OutFolder: d.config.Destination,
		PkgName:   d.config.Pkgname,
		Templates: append(d.templates, gen.BaseTemplates.DBErrors),
	})

	return nil
}
