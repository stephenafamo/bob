package psql

import (
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql/dialect"
	"github.com/stephenafamo/bob/orm"
)

type (
	// Preloader builds a query mod that modifies the original query to retrieve related fields
	// while it can be used as a queryMod, it does not have any direct effect.
	// if using manually, the ApplyPreload method should be called
	// with the query's context AFTER other mods have been applied
	Preloader   = orm.Preloader[*dialect.SelectQuery]
	PreloadRel  = orm.PreloadRel[Expression]
	PreloadSide = orm.PreloadSide[Expression]
	// Settings for preloading relationships
	PreloadSettings = orm.PreloadSettings[*dialect.SelectQuery]
	// Modifies preloading relationships
	PreloadOption = orm.PreloadOption[*dialect.SelectQuery]
)

func PreloadOnly(cols ...string) PreloadOption {
	return orm.PreloadOnly[*dialect.SelectQuery](cols)
}

func PreloadExcept(cols ...string) PreloadOption {
	return orm.PreloadExcept[*dialect.SelectQuery](cols)
}

func PreloadWhere(f ...func(from, to string) []bob.Expression) PreloadOption {
	return orm.PreloadWhere[*dialect.SelectQuery](f)
}

func PreloadAs(alias string) PreloadOption {
	return orm.PreloadAs[*dialect.SelectQuery](alias)
}

func Preload[T orm.Preloadable, Ts ~[]T](rel orm.PreloadRel[Expression], cols []string, opts ...PreloadOption) Preloader {
	return orm.Preload[T, Ts](rel, cols, opts...)
}
