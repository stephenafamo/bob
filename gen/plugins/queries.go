package plugins

import (
	"fmt"
	"io/fs"
	"slices"

	"github.com/stephenafamo/bob/gen"
	"github.com/stephenafamo/bob/gen/drivers"
)

func Queries[T, C, I any](templates ...fs.FS) gen.Plugin {
	return &queriesOutputPlugin[T, C, I]{
		templates: templates,
	}
}

type queriesOutputPlugin[T, C, I any] struct {
	hasEnumsOutput bool
	templates      []fs.FS
}

// Name implements gen.StatePlugin.
func (*queriesOutputPlugin[T, C, I]) Name() string {
	return "Queries Output Plugin"
}

// PlugState implements gen.StatePlugin.
func (q *queriesOutputPlugin[T, C, I]) PlugState(state *gen.State[C]) error {
	for _, output := range state.Outputs {
		if output.Key == "enums" && !output.Disabled {
			q.hasEnumsOutput = true
			break
		}
	}

	state.Outputs = append(state.Outputs, &gen.Output{
		Key:       "queries",
		Templates: append(q.templates, gen.BaseTemplates.Queries),
	})

	return nil
}

// PlugDBInfo implements gen.DBInfoPlugin.
func (q *queriesOutputPlugin[T, C, I]) PlugTemplateData(data *gen.TemplateData[T, C, I]) error {
	var usesEnums bool

MainLoop:
	for _, folder := range data.QueryFolders {
		for _, file := range folder.Files {
			for _, query := range file.Queries {
				for _, col := range query.Columns {
					if hasEnumImports(data.Types.Index(col.TypeName)) {
						usesEnums = true
						break MainLoop
					}
				}
				for _, arg := range query.Args {
					if hasEnumImports(data.Types.Index(arg.Col.TypeName)) {
						usesEnums = true
						break MainLoop
					}
				}
			}
		}
	}
	// Disable the output if there are no enums
	if usesEnums && !q.hasEnumsOutput {
		return fmt.Errorf("your queries uses enums and so requires the \"enum\" output to be enabled")
	}

	return nil
}

func hasEnumImports(typ drivers.Type) bool {
	if slices.Contains(typ.Imports, "output(enums)") {
		return true
	}

	if slices.Contains(typ.RandomExprImports, "output(enums)") {
		return true
	}

	return slices.Contains(typ.CompareExprImports, "output(enums)")
}
