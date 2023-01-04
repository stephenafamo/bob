package driver

import (
	"fmt"
	"strings"

	"github.com/stephenafamo/bob/gen/drivers"
)

func (d *Driver) Constraints(colFilter drivers.ColumnFilter) (drivers.DBConstraints, error) {
	ret := drivers.DBConstraints{
		PKs:     map[string]*drivers.Constraint{},
		FKs:     map[string][]drivers.ForeignKey{},
		Uniques: map[string][]drivers.Constraint{},
	}

	allfilter := colFilter["*"]

	for _, model := range d.config.Datamodel.Models {
		filter := colFilter[model.Name]
		include := append(allfilter.Include, filter.Include...)
		exclude := append(allfilter.Exclude, filter.Exclude...)

		// If it is a composite primary key defined on the model
		if len(model.PrimaryKey.Fields) > 0 {
			shouldSkip := false
			cols := make([]string, len(model.PrimaryKey.Fields))

			for i, f := range model.PrimaryKey.Fields {
				if skip(f, include, exclude) {
					shouldSkip = true
				}
				cols[i] = f
			}

			if !shouldSkip {
				pkName := model.PrimaryKey.Name
				if pkName == "" {
					pkName = "pk_" + model.Name
				}
				ret.PKs[model.Name] = &drivers.Constraint{
					Name:    pkName,
					Columns: cols,
				}
			}
		}

		for _, unique := range model.UniqueIndexes {
			shouldSkip := false
			cols := make([]string, len(unique.Fields))

			for i, f := range unique.Fields {
				if skip(f, include, exclude) {
					shouldSkip = true
				}
				cols[i] = f
			}

			if !shouldSkip {
				keyName := unique.InternalName
				if keyName == "" {
					keyName = fmt.Sprintf("unique_%s_%s", model.Name, strings.Join(cols, "_"))
				}

				ret.Uniques[model.Name] = append(ret.Uniques[model.Name], drivers.Constraint{
					Name:    keyName,
					Columns: cols,
				})
			}
		}

		// If one of the fields has an @id attribute
		for _, field := range model.Fields {
			if skip(field.Name, include, exclude) {
				continue
			}

			if field.IsID {
				ret.PKs[model.Name] = &drivers.Constraint{
					Name:    "pk_" + model.Name,
					Columns: []string{field.Name},
				}
			}

			if field.IsUnique {
				ret.Uniques[model.Name] = append(ret.Uniques[model.Name], drivers.Constraint{
					Name:    fmt.Sprintf("unique_%s_%s", model.Name, field.Name),
					Columns: []string{field.Name},
				})
			}

			if field.Kind == FieldKindObject && len(field.RelationFromFields) > 0 {
				ret.FKs[model.Name] = append(ret.FKs[model.Name], drivers.ForeignKey{
					Constraint: drivers.Constraint{
						Name:    field.RelationName,
						Columns: field.RelationFromFields,
					},
					ForeignTable:   field.Type,
					ForeignColumns: field.RelationToFields,
				})
			}
		}
	}

	return ret, nil
}
