package orm

import "github.com/stephenafamo/bob"

type Table interface {
	PrimaryKeyVals() bob.Expression
}

type Setter[T any, InsertQ any, UpdateQ any] interface {
	SetColumns() []string
	Overwrite(T)
	Apply(UpdateQ)
	Insert() bob.Mod[InsertQ]
}
