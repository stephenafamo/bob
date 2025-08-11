package plugins

import "github.com/stephenafamo/bob/gen"

var (
	_ gen.StatePlugin[any]            = &enumsPlugin[any, any, any]{}
	_ gen.DBInfoPlugin[any, any, any] = &enumsPlugin[any, any, any]{}

	_ gen.StatePlugin[any] = modelsPlugin[any]{}
	_ gen.StatePlugin[any] = factoryPlugin[any]{}
	_ gen.StatePlugin[any] = dbErrorsPlugin[any]{}
	_ gen.StatePlugin[any] = joinsPlugin[any]{}
	_ gen.StatePlugin[any] = loadersPlugin[any]{}

	_ gen.StatePlugin[any]                  = &queriesOutputPlugin[any, any, any]{}
	_ gen.TemplateDataPlugin[any, any, any] = &queriesOutputPlugin[any, any, any]{}
)
