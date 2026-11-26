package caller

import "github.com/stephenafamo/bob/analysis/bobcheck/testdata/models"

type jobs = models.Query[int]

func check(q jobs, w models.Jobs, mutable models.Mutable, mods []int) jobs {
	q.WithMods(1)             // want "result of WithMods is discarded"
	w.WithMods(mods...)       // want "result of WithMods is discarded"
	jobs.WithMods(q)          // want "result of WithMods is discarded"
	go q.WithMods()           // want "result of WithMods is discarded"
	defer q.WithMods()        // want "result of WithMods is discarded"
	q.WithMods(1).WithMods(2) // want "result of WithMods is discarded"
	q = q.WithMods(1)
	q.WithMods(2).Run()
	_ = q.WithMods(3) // Explicit discard.
	mutable.WithMods()
	models.WithMods(q) // Package function.
	struct{ WithMods func() jobs }{func() jobs { return q }}.WithMods()
	return q.WithMods(4)
}

func viaInterface(q interface{ WithMods() jobs }) {
	q.WithMods() // want "result of WithMods is discarded"
}

func base(q models.Base) {
	q.WithMods() // want "result of WithMods is discarded"
}

type other struct{}

func (other) WithMods() other { return other{} }

func unrelated() { other{}.WithMods() }

type builder[T any] struct{ next T }

func (b builder[T]) WithMods() T { return b.next }

func generic(query builder[jobs], nonQuery builder[int]) {
	query.WithMods() // want "result of WithMods is discarded"
	nonQuery.WithMods()
}
