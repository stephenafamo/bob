package gen

import "github.com/stephenafamo/bob/gen/drivers"

type Plugin interface {
	Name() string
}

// This is called at the very beginning if there are any changes to be made to the state
type StatePlugin interface {
	Plugin
	PlugState(*State) error
}

// DBInfoPlugin is called immediately after the database information
// is assembled from the driver
type DBInfoPlugin[T any] interface {
	Plugin
	PlugDBInfo(*drivers.DBInfo[T]) error
}

// TemplateDataPlugin is called right after assembling the template data, before
// generating them for each output.
// NOTE: The PkgName field is overwritten for each output, so mofifying it in a plugin
// will have no effect. Use a StatePlugin instead
type TemplateDataPlugin[T any] interface {
	Plugin
	PlugTemplateData(*TemplateData[T]) error
}
