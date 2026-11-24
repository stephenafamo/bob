package models

import "github.com/stephenafamo/bob"

type Query[T any] struct{ bob.BaseQuery[bob.Expression] }

func (q Query[T]) Apply(...int) Query[T] { return q }
func (q Query[T]) Run()                  {}

type Jobs struct{ Query[int] }

type Mutable = bob.BaseQuery[bob.Expression]

func Apply(q Query[int]) Query[int] { return q }
