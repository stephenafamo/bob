package models

import "github.com/stephenafamo/bob"

type Query[T any] struct {
	bob.ExpressionQuery[bob.Expression]
}

func (q Query[T]) WithMods(...int) Query[T] { return q }
func (q Query[T]) Run()                     {}

type Jobs struct{ Query[int] }

type Base = bob.QueryBuilder[int]

type Mutable struct{}

func (Mutable) WithMods() {}

func WithMods(q Query[int]) Query[int] { return q }
