package dialect

import (
	"context"
	"slices"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/clause"
	"github.com/stephenafamo/scan"
)

type selectShared uint32

const (
	sharedWith selectShared = 1 << iota
	sharedSelect
	sharedPreloadSelect
	sharedJoins
	sharedWhere
	sharedGroups
	sharedHaving
	sharedWindows
	sharedCombines
	sharedOrderBy
	sharedLoaders
	sharedMapperMods
	sharedHooks
	sharedContextualMods
)

const sharedAll = sharedWith |
	sharedSelect |
	sharedPreloadSelect |
	sharedJoins |
	sharedWhere |
	sharedGroups |
	sharedHaving |
	sharedWindows |
	sharedCombines |
	sharedOrderBy |
	sharedLoaders |
	sharedMapperMods |
	sharedHooks |
	sharedContextualMods

func (s *SelectQuery) Clone() *SelectQuery {
	if s == nil {
		return nil
	}

	clone := *s
	s.shared |= sharedAll
	clone.shared |= sharedAll

	return &clone
}

func (s *SelectQuery) cloneOnWrite(flag selectShared) {
	if s.shared&flag == 0 {
		return
	}

	switch flag {
	case sharedWith:
		s.With.CTEs = slices.Clone(s.With.CTEs)
	case sharedSelect:
		s.SelectList.Columns = slices.Clone(s.SelectList.Columns)
	case sharedPreloadSelect:
		s.SelectList.PreloadColumns = slices.Clone(s.SelectList.PreloadColumns)
	case sharedJoins:
		s.TableRef.Joins = slices.Clone(s.TableRef.Joins)
	case sharedWhere:
		s.Where.Conditions = slices.Clone(s.Where.Conditions)
	case sharedGroups:
		s.GroupBy.Groups = slices.Clone(s.GroupBy.Groups)
	case sharedHaving:
		s.Having.Conditions = slices.Clone(s.Having.Conditions)
	case sharedWindows:
		s.Windows.Windows = slices.Clone(s.Windows.Windows)
	case sharedCombines:
		s.Combines.Queries = slices.Clone(s.Combines.Queries)
	case sharedOrderBy:
		s.OrderBy.Expressions = slices.Clone(s.OrderBy.Expressions)
	case sharedLoaders:
		s.Load.SetLoaders(slices.Clone(s.Load.GetLoaders())...)
	case sharedMapperMods:
		s.Load.SetMapperMods(slices.Clone(s.Load.GetMapperMods())...)
	case sharedHooks:
		s.EmbeddedHook.SetHooks(slices.Clone(s.EmbeddedHook.Hooks)...)
	case sharedContextualMods:
		s.ContextualModdable.Mods = slices.Clone(s.ContextualModdable.Mods)
	}

	s.shared &^= flag
}

func (s *SelectQuery) AppendCTE(cte bob.Expression) {
	s.cloneOnWrite(sharedWith)
	s.With.AppendCTE(cte)
}

func (s *SelectQuery) AppendSelect(columns ...any) {
	s.cloneOnWrite(sharedSelect)
	s.SelectList.AppendSelect(columns...)
}

func (s *SelectQuery) SetSelect(columns ...any) {
	s.shared &^= sharedSelect
	s.SelectList.SetSelect(columns...)
}

func (s *SelectQuery) AppendPreloadSelect(columns ...any) {
	s.cloneOnWrite(sharedPreloadSelect)
	s.SelectList.AppendPreloadSelect(columns...)
}

func (s *SelectQuery) SetPreloadSelect(columns ...any) {
	s.shared &^= sharedPreloadSelect
	s.SelectList.SetPreloadSelect(columns...)
}

func (s *SelectQuery) AppendJoin(join clause.Join) {
	s.cloneOnWrite(sharedJoins)
	s.TableRef.AppendJoin(join)
}

func (s *SelectQuery) AppendWhere(e ...any) {
	s.cloneOnWrite(sharedWhere)
	s.Where.AppendWhere(e...)
}

func (s *SelectQuery) AppendGroup(e any) {
	s.cloneOnWrite(sharedGroups)
	s.GroupBy.AppendGroup(e)
}

func (s *SelectQuery) SetGroups(groups ...any) {
	s.shared &^= sharedGroups
	s.GroupBy.SetGroups(groups...)
}

func (s *SelectQuery) AppendHaving(e ...any) {
	s.cloneOnWrite(sharedHaving)
	s.Having.AppendHaving(e...)
}

func (s *SelectQuery) AppendWindow(w bob.Expression) {
	s.cloneOnWrite(sharedWindows)
	s.Windows.AppendWindow(w)
}

func (s *SelectQuery) AppendCombine(combine clause.Combine) {
	s.cloneOnWrite(sharedCombines)
	s.Combines.AppendCombine(combine)
}

func (s *SelectQuery) AppendOrder(order bob.Expression) {
	s.cloneOnWrite(sharedOrderBy)
	s.OrderBy.AppendOrder(order)
}

func (s *SelectQuery) ClearOrderBy() {
	s.shared &^= sharedOrderBy
	s.OrderBy.ClearOrderBy()
}

func (s *SelectQuery) AppendLoader(loaders ...bob.Loader) {
	s.cloneOnWrite(sharedLoaders)
	s.Load.AppendLoader(loaders...)
}

func (s *SelectQuery) SetLoaders(loaders ...bob.Loader) {
	s.shared &^= sharedLoaders
	s.Load.SetLoaders(loaders...)
}

func (s *SelectQuery) AppendMapperMod(mod scan.MapperMod) {
	s.cloneOnWrite(sharedMapperMods)
	s.Load.AppendMapperMod(mod)
}

func (s *SelectQuery) SetMapperMods(mods ...scan.MapperMod) {
	s.shared &^= sharedMapperMods
	s.Load.SetMapperMods(mods...)
}

func (s *SelectQuery) AppendHooks(hooks ...func(context.Context, bob.Executor) (context.Context, error)) {
	s.cloneOnWrite(sharedHooks)
	s.EmbeddedHook.AppendHooks(hooks...)
}

func (s *SelectQuery) SetHooks(hooks ...func(context.Context, bob.Executor) (context.Context, error)) {
	s.shared &^= sharedHooks
	s.EmbeddedHook.SetHooks(hooks...)
}

func (s *SelectQuery) AppendContextualMod(mods ...bob.ContextualMod[*SelectQuery]) {
	s.cloneOnWrite(sharedContextualMods)
	s.ContextualModdable.AppendContextualMod(mods...)
}

func (s *SelectQuery) AppendContextualModFunc(f func(context.Context, *SelectQuery) (context.Context, error)) {
	s.cloneOnWrite(sharedContextualMods)
	s.ContextualModdable.AppendContextualModFunc(f)
}
