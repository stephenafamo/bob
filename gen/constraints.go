package gen

import "github.com/stephenafamo/bob/gen/drivers"

type Constraints[C any] map[string]drivers.Constraints[C]

func processConstraintConfig[C, I any](tables []drivers.Table[C, I], extras Constraints[C]) {
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

func mergeConstraints[C any](srcs, extras drivers.Constraints[C]) drivers.Constraints[C] {
	if extras.Primary != nil {
		srcs.Primary = extras.Primary
	}

	srcs.Uniques = append(srcs.Uniques, extras.Uniques...)
	srcs.Foreign = append(srcs.Foreign, extras.Foreign...)
	srcs.Checks = append(srcs.Checks, extras.Checks...)

	return srcs
}
