package lm

import (
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql/dialect"
)

func Channel(name string) bob.Mod[*dialect.ListenQuery] {
	return bob.ModFunc[*dialect.ListenQuery](func(q *dialect.ListenQuery) {
		q.Channel = name
	})
}
