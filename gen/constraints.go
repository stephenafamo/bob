package gen

import "github.com/stephenafamo/bob/gen/drivers"

type Constraints map[string]drivers.Constraints

func processConstraintConfig(tables []drivers.Table, extras Constraints) {
	if len(tables) == 0 {
		return
	}

	for i, t := range tables {
		extra, ok := extras[t.Key]
		if !ok {
			continue
		}

		tables[i].Constraints = mergeConstraints(t.Constraints, extra)
	}
}

func mergeConstraints(srcs, extras drivers.Constraints) drivers.Constraints {
	if extras.Primary != nil {
		srcs.Primary = extras.Primary
	}

	srcs.Uniques = append(srcs.Uniques, extras.Uniques...)
	srcs.Foreign = append(srcs.Foreign, extras.Foreign...)

	return srcs
}
