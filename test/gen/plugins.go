package testgen

import (
	"embed"
	"fmt"
	"io/fs"
	"path/filepath"
	"slices"
	"strings"

	"github.com/stephenafamo/bob/gen"
	"github.com/stephenafamo/bob/gen/drivers"
)

//go:embed templates
var TestTemplates embed.FS

var (
	_ gen.DBInfoPlugin[any, any, any]       = queryPathPlugin[any, any, any]{}
	_ gen.StatePlugin[any]                  = templatePlugin[any]{}
	_ gen.StatePlugin[any]                  = (*aliasPlugin[any, any, any])(nil)
	_ gen.TemplateDataPlugin[any, any, any] = (*aliasPlugin[any, any, any])(nil)
)

type templatePlugin[C any] struct {
	dialect string // "mysql", "psql", "sqlite"
}

func (q templatePlugin[C]) Name() string {
	return "template"
}

func (t templatePlugin[C]) PlugState(s *gen.State[C]) error {
	templates, err := fs.Sub(TestTemplates, "templates")
	if err != nil {
		return fmt.Errorf("failed to load templates: %w", err)
	}

	// Load shared factory templates (includes database.bob_test.go.tpl)
	sharedFactoryTemplates, err := fs.Sub(TestTemplates, "templates/factory")
	if err != nil {
		return fmt.Errorf("failed to load shared factory templates: %w", err)
	}

	// Load dialect-specific factory templates if they exist
	var dialectFactoryTemplates fs.FS
	if t.dialect != "" {
		dialectFactoryTemplates, err = fs.Sub(TestTemplates, "templates/factory/"+t.dialect)
		if err != nil {
			// Dialect-specific templates are optional
			dialectFactoryTemplates = nil
		}
	}

	for i := range s.Outputs {
		if slices.Contains([]string{"dbinfo", "enums"}, s.Outputs[i].Key) {
			// Skip the enums output, since there is no testDB defined
			continue
		}

		// Factory output gets its own templates, not the root templates
		if s.Outputs[i].Key == "factory" {
			s.Outputs[i].Templates = append(s.Outputs[i].Templates, sharedFactoryTemplates)
			if dialectFactoryTemplates != nil {
				s.Outputs[i].Templates = append(s.Outputs[i].Templates, dialectFactoryTemplates)
			}
		} else {
			s.Outputs[i].Templates = append(s.Outputs[i].Templates, templates)
		}
	}

	return nil
}

type queryPathPlugin[T, C, I any] struct {
	outputPath   string
	trimPrefixes []string
}

func (q queryPathPlugin[T, C, I]) Name() string {
	return "query_path"
}

func (q queryPathPlugin[T, C, I]) PlugDBInfo(info *drivers.DBInfo[T, C, I]) error {
	for i, folder := range info.QueryFolders {
		info.QueryFolders[i].Path = q.addPath(folder.Path)
		for j, file := range folder.Files {
			info.QueryFolders[i].Files[j].Path = q.addPath(file.Path)
		}
	}

	return nil
}

func (q queryPathPlugin[T, C, I]) addPath(old string) string {
	for _, prefix := range q.trimPrefixes {
		old = strings.TrimPrefix(old, prefix)
	}
	return filepath.Join(q.outputPath, old)
}

type aliasPlugin[T, C, I any] struct {
	tables drivers.Tables[C, I]
	rels   gen.Relationships
}

func (a *aliasPlugin[T, C, I]) Name() string {
	return "aliaser"
}

func (a *aliasPlugin[T, C, I]) PlugState(s *gen.State[C]) error {
	if a.rels == nil || len(a.tables) == 0 {
		return nil
	}

	aliases := make(map[string]drivers.TableAlias, len(a.tables))
	for i, table := range a.tables {
		tableAlias := drivers.TableAlias{
			UpPlural:     fmt.Sprintf("Alias%dThings", i),
			UpSingular:   fmt.Sprintf("Alias%dThing", i),
			DownPlural:   fmt.Sprintf("alias%dThings", i),
			DownSingular: fmt.Sprintf("alias%dThing", i),
		}

		tableAlias.Columns = make(map[string]string)
		for j, column := range table.Columns {
			tableAlias.Columns[column.Name] = fmt.Sprintf("Alias%dThingColumn%d", i, j)
		}

		tableAlias.Relationships = make(map[string]string)
		for j, rel := range a.rels[table.Key] {
			tableAlias.Relationships[rel.Name] = fmt.Sprintf("Alias%dThingRel%d", i, j)
		}

		aliases[table.Key] = tableAlias
	}

	s.Config.Aliases = aliases

	return nil
}

func (a *aliasPlugin[T, C, I]) PlugTemplateData(data *gen.TemplateData[T, C, I]) error {
	a.tables = data.Tables
	a.rels = data.Relationships
	return nil
}
