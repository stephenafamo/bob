package parser

import (
	"strings"

	"github.com/stephenafamo/bob/gen/drivers"
)

// ParseQueryConfig parses a user configuration string into a QueryCoonfig.
// The configuration string should be in the format:
// "row_name:row_slice_name:generate_row"
func ParseQueryConfig(options string) drivers.QueryConfig {
	var i int
	var part string
	var found bool

	col := drivers.QueryConfig{
		GenerateRow: true,
	}
	for {
		part, options, found = strings.Cut(options, ":")
		switch i {
		case 0:
			col.RowName = part
		case 1:
			col.RowSliceName = part
		case 2:
			switch part {
			case "true", "yes":
				col.GenerateRow = true
			case "false", "no", "skip":
				col.GenerateRow = false
			}
		}
		if !found {
			break
		}
		i++
	}

	return col
}

// ParseQueryColumnConfig parses a user configuration string into a QueryCol.
// The configuration string should be in the format:
// "name:type:notnull"
func ParseQueryColumnConfig(options string) drivers.QueryCol {
	var i int
	var part string
	var found bool

	col := drivers.QueryCol{}
	for {
		part, options, found = strings.Cut(options, ":")
		switch i {
		case 0:
			col.Name = part
		case 1:
			col.TypeName = part
		case 2:
			switch part {
			case "null", "true", "yes":
				col.Nullable.Set(true)
			case "notnull", "nnull", "false", "no":
				col.Nullable.Set(false)
			}
		}
		if !found {
			break
		}
		i++
	}

	return col
}
