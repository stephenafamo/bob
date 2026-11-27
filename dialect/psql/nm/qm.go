package nm

import (
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql/dialect"
)

func Channel(name string) bob.Mod[*dialect.NotifyQuery] {
	return bob.ModFunc[*dialect.NotifyQuery](func(q *dialect.NotifyQuery) {
		q.Channel = name
	})
}

func Payload(p string) bob.Mod[*dialect.NotifyQuery] {
	return bob.ModFunc[*dialect.NotifyQuery](func(q *dialect.NotifyQuery) {
		q.Payload = p
	})
}
