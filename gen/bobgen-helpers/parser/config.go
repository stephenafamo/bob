package parser

import (
	"strings"

	"github.com/stephenafamo/bob/gen/drivers"
	"github.com/stephenafamo/bob/internal"
)

// ParseQueryConfig parses a user configuration string into a QueryCoonfig.
// The configuration string should be in the format:
// "result_type_one:result_type_all:result_type_transformer"
func ParseQueryConfig(options string) drivers.QueryConfig {
	var i int
	var part string
	var found bool

	col := drivers.QueryConfig{}
	for {
		part, options, found = strings.Cut(options, ":")
		switch i {
		case 0:
			col.ResultTypeOne = part
		case 1:
			col.ResultTypeAll = part
		case 2:
			col.ResultTransformer = part
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
	if options == "" {
		return drivers.QueryCol{}
	}

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
				col.Nullable = internal.Pointer(true)
			case "notnull", "nnull", "false", "no":
				col.Nullable = internal.Pointer(false)
			}
		}
		if !found {
			break
		}
		i++
	}

	return col
}
