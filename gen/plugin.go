package gen

import "github.com/stephenafamo/bob/gen/drivers"

type Plugin interface {
	Name() string
}

// This is called at the very beginning if there are any changes to be made to the state
type StatePlugin[ConstraintExtra any] interface {
	Plugin
	PlugState(*State[ConstraintExtra]) error
}

// DBInfoPlugin is called immediately after the database information
// is assembled from the driver
type DBInfoPlugin[T, C, I any] interface {
	Plugin
	PlugDBInfo(*drivers.DBInfo[T, C, I]) error
}

// TemplateDataPlugin is called right after assembling the template data, before
// generating them for each output.
// NOTE: The PkgName field is overwritten for each output, so mofifying it in a plugin
// will have no effect. Use a StatePlugin instead
type TemplateDataPlugin[T, C, I any] interface {
	Plugin
	PlugTemplateData(*TemplateData[T, C, I]) error
}
