package plugins

import (
	"io/fs"

	"github.com/stephenafamo/bob/gen"
	"github.com/stephenafamo/bob/gen/drivers"
	"github.com/stephenafamo/bob/internal"
)

func Enums[T, C, I any](config OutputConfig, templates ...fs.FS) gen.Plugin {
	config = config.WithDefaults("enums")
	return &enumsPlugin[T, C, I]{
		config: config,
		output: &gen.Output{
			Disabled:  internal.ValOrZero(config.Disabled),
			Key:       "enums",
			OutFolder: config.Destination,
			PkgName:   config.Pkgname,
			Templates: append(templates, gen.BaseTemplates.Enums),
		},
	}
}

type enumsPlugin[T, C, I any] struct {
	config OutputConfig
	output *gen.Output
}

// Name implements gen.StatePlugin.
func (*enumsPlugin[T, C, I]) Name() string {
	return "Enums Output Plugin"
}

// PlugState implements gen.StatePlugin.
func (e *enumsPlugin[T, C, I]) PlugState(state *gen.State[C]) error {
	state.Outputs = append(state.Outputs, e.output)

	return nil
}

// PlugDBInfo implements gen.DBInfoPlugin.
func (e *enumsPlugin[T, C, I]) PlugDBInfo(info *drivers.DBInfo[T, C, I]) error {
	// Disable the output if there are no enums
	if len(info.Enums) == 0 {
		e.output.Disabled = true
	}
	return nil
}
