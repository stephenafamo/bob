package caller

import "github.com/stephenafamo/bob/analysis/bobcheck/testdata/models"

type jobs = models.Query[int]

func check(q jobs, w models.Jobs, mutable models.Mutable, mods []int) jobs {
	q.Apply(1)          // want "result of Apply is discarded"
	w.Apply(mods...)    // want "result of Apply is discarded"
	jobs.Apply(q)       // want "result of Apply is discarded"
	go q.Apply()        // want "result of Apply is discarded"
	defer q.Apply()     // want "result of Apply is discarded"
	q.Apply(1).Apply(2) // want "result of Apply is discarded"
	q = q.Apply(1)
	q.Apply(2).Run()
	_ = q.Apply(3) // Explicit discard.
	mutable.Apply()
	models.Apply(q) // Package function.
	struct{ Apply func() jobs }{func() jobs { return q }}.Apply()
	return q.Apply(4)
}

func viaInterface(q interface{ Apply() jobs }) {
	q.Apply() // want "result of Apply is discarded"
}

type other struct{}

func (other) Apply() other { return other{} }

func unrelated() { other{}.Apply() }

type builder[T any] struct{ next T }

func (b builder[T]) Apply() T { return b.next }

func generic(query builder[jobs], nonQuery builder[int]) {
	query.Apply() // want "result of Apply is discarded"
	nonQuery.Apply()
}
