package clause

import "slices"

func (w With) Clone() With {
	w.CTEs = slices.Clone(w.CTEs)

	return w
}

func (s SelectList) Clone() SelectList {
	s.Columns = slices.Clone(s.Columns)
	s.PreloadColumns = slices.Clone(s.PreloadColumns)

	return s
}

func (f TableRef) Clone() TableRef {
	f.Columns = slices.Clone(f.Columns)
	f.Partitions = slices.Clone(f.Partitions)
	f.IndexHints = slices.Clone(f.IndexHints)
	f.Joins = cloneJoins(f.Joins)

	if f.IndexedBy != nil {
		indexedBy := *f.IndexedBy
		f.IndexedBy = &indexedBy
	}

	return f
}

func cloneJoins(joins []Join) []Join {
	if joins == nil {
		return nil
	}

	cloned := make([]Join, len(joins))
	for i, join := range joins {
		cloned[i] = join.Clone()
	}

	return cloned
}

func (j Join) Clone() Join {
	j.To = j.To.Clone()
	j.On = slices.Clone(j.On)
	j.Using = slices.Clone(j.Using)

	return j
}

func (w Where) Clone() Where {
	w.Conditions = slices.Clone(w.Conditions)

	return w
}

func (g GroupBy) Clone() GroupBy {
	g.Groups = slices.Clone(g.Groups)

	return g
}

func (h Having) Clone() Having {
	h.Conditions = slices.Clone(h.Conditions)

	return h
}

func (w Windows) Clone() Windows {
	w.Windows = slices.Clone(w.Windows)

	return w
}

func (c Combines) Clone() Combines {
	c.Queries = slices.Clone(c.Queries)

	return c
}

func (o OrderBy) Clone() OrderBy {
	o.Expressions = slices.Clone(o.Expressions)

	return o
}

func (l Locks) Clone() Locks {
	l.Locks = slices.Clone(l.Locks)

	return l
}
