package main

import (
	"github.com/iancoleman/strcase"
	"github.com/stephenafamo/bob/gen"
	"github.com/stephenafamo/bob/gen/bobgen-prisma/driver"
	"github.com/stephenafamo/bob/gen/drivers"
)

// This plugin sets the default Alias of relationships based on the field name
// we do this after building the relationships based on the keys
type relAliasPlugin struct {
	models  []driver.Model
	aliases gen.Aliases
	config  driver.Config
}

func (p relAliasPlugin) Name() string {
	return "relationshipAliaser"
}

//nolint:unparam
func (p relAliasPlugin) PlugTemplateData(data *gen.TemplateData[driver.Extra]) error {
	tblFilter := drivers.ParseTableFilter(p.config.Only, p.config.Except)

	colFilter := drivers.ParseColumnFilter(
		driver.TableNamesFromFilter(p.models, drivers.Filter{
			Only:   tblFilter.Only,
			Except: tblFilter.Except,
		}), p.config.Only, p.config.Except)

	for _, model := range p.models {
		if drivers.Skip(model.TableName(), tblFilter.Only, tblFilter.Except) {
			continue
		}

		var table drivers.Table
		for _, t := range data.Tables {
			if t.Key == model.TableName() {
				table = t
				break
			}
		}

		if table.Key == "" {
			continue
		}

		tableName := model.TableName()
		allfilter := colFilter["*"]
		filter := colFilter[tableName]
		include := append(allfilter.Only, filter.Only...)
		exclude := append(allfilter.Except, filter.Except...)
		for _, field := range model.Fields {
			if drivers.Skip(field.Name, include, exclude) {
				continue
			}

			if field.Kind != driver.FieldKindObject {
				continue
			}

			for _, rel := range data.Relationships[table.Key] {
				if rel.Name != field.RelationName {
					continue
				}

				if p.aliases[table.Key].Relationships[rel.Name] == "" {
					data.Aliases[table.Key].Relationships[rel.Name] = strcase.ToCamel(field.Name)
				}
			}
		}
	}

	return nil
}
