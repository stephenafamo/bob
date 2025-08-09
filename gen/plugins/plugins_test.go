package plugins

import "github.com/stephenafamo/bob/gen"

var (
	_ gen.StatePlugin[any] = enumsPlugin[any]{}
	_ gen.StatePlugin[any] = modelsPlugin[any]{}
	_ gen.StatePlugin[any] = factoryPlugin[any]{}
	_ gen.StatePlugin[any] = queriesOutputPlugin[any]{}
	_ gen.StatePlugin[any] = dbErrorsPlugin[any]{}
)
