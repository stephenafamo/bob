package plugins

import (
	"io/fs"

	"github.com/stephenafamo/bob/gen"
	"github.com/stephenafamo/bob/internal"
)

func DBInfo[C any](config OutputConfig, templates ...fs.FS) gen.StatePlugin[C] {
	return dbInfoPlugin[C]{
		config:    config.WithDefaults("dbinfo"),
		templates: templates,
	}
}

type dbInfoPlugin[C any] struct {
	config    OutputConfig
	templates []fs.FS
}

// Name implements gen.StatePlugin.
func (dbInfoPlugin[C]) Name() string {
	return "DBInfo Output Plugin"
}

// PlugState implements gen.StatePlugin.
func (d dbInfoPlugin[C]) PlugState(state *gen.State[C]) error {
	state.Outputs = append(state.Outputs, &gen.Output{
		Disabled:  internal.ValOrZero(d.config.Disabled),
		Key:       "dbinfo",
		OutFolder: d.config.Destination,
		PkgName:   d.config.Pkgname,
		Templates: append(d.templates, gen.BaseTemplates.DBInfo),
	})

	return nil
}
