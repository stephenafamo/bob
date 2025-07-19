package drivers

import (
	"strings"
)

type Filter struct {
	Only   []string
	Except []string
}

//nolint:nonamedreturns
func ClassifyPatterns(patterns []string) (stringPatterns []string, regexPatterns []string) {
	const regexDelimiter = "/"

	for _, pattern := range patterns {
		if isRegexPattern(pattern, regexDelimiter) {
			regexPatterns = append(regexPatterns, strings.Trim(pattern, regexDelimiter))
			continue
		}
		stringPatterns = append(stringPatterns, pattern)
	}

	return stringPatterns, regexPatterns
}

func isRegexPattern(pattern, delimiter string) bool {
	return strings.HasPrefix(pattern, delimiter) && strings.HasSuffix(pattern, delimiter)
}

type ColumnFilter map[string]Filter

func ParseTableFilter(only, except map[string][]string) Filter {
	var filter Filter
	for name := range only {
		filter.Only = append(filter.Only, name)
	}

	for name, cols := range except {
		// If they only want to exclude some columns, then we don't want to exclude the whole table
		if len(cols) == 0 {
			filter.Except = append(filter.Except, name)
		}
	}

	return filter
}

func ParseColumnFilter(tables []string, only, except map[string][]string) ColumnFilter {
	global := Filter{
		Only:   only["*"],
		Except: except["*"],
	}

	colFilter := make(ColumnFilter, len(tables))
	for _, t := range tables {
		colFilter[t] = Filter{
			Only:   append(global.Only, only[t]...),
			Except: append(global.Except, except[t]...),
		}
	}
	return colFilter
}
