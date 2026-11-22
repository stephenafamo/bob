package orm

// OneRelAssignment sets a to-one relationship on a generated model. Slot returns
// the relationship field and its loaded flag. Backlink, when present, updates
// only the inverse field; it must not recursively link the parent again.
type OneRelAssignment[P, C any] struct {
	Slot     func(*P) (**C, *bool)
	Backlink func(*C, *P)
}

// Reset discards a previous result. A successful batch load resets all
// parents to loaded, including parents with no matching child.
func (r OneRelAssignment[P, C]) Reset(p *P, loaded bool) {
	slot, flag := r.Slot(p)
	*slot, *flag = nil, loaded
}

// SetLocal records the known child without changing the inverse.
func (r OneRelAssignment[P, C]) SetLocal(p *P, c *C) {
	slot, loaded := r.Slot(p)
	*slot, *loaded = c, true
}

// Replace records a complete result, including a known absent (nil) child.
func (r OneRelAssignment[P, C]) Replace(p *P, c *C) {
	r.SetLocal(p, c)
	if c != nil && r.Backlink != nil {
		r.Backlink(c, p)
	}
}

// Link keeps the first matching child while stitching a batch.
func (r OneRelAssignment[P, C]) Link(p *P, c *C) {
	slot, _ := r.Slot(p)
	if *slot == nil {
		r.Replace(p, c)
	}
}

// ManyRelAssignment sets a to-many relationship on a generated model.
type ManyRelAssignment[P, C any, S ~[]*C] struct {
	Slot     func(*P) (*S, *bool)
	Backlink func(*C, *P)
}

func (r ManyRelAssignment[P, C, S]) Reset(p *P, loaded bool) {
	slot, flag := r.Slot(p)
	*slot, *flag = nil, loaded
}

func (r ManyRelAssignment[P, C, S]) backfill(p *P, children S) {
	if r.Backlink != nil {
		for _, c := range children {
			if c != nil {
				r.Backlink(c, p)
			}
		}
	}
}

// Replace installs the complete slice as returned, preserving nil vs empty.
func (r ManyRelAssignment[P, C, S]) Replace(p *P, children S) {
	slot, loaded := r.Slot(p)
	*slot, *loaded = children, true
	r.backfill(p, children)
}

// AppendLocal adds known children without establishing a complete load or
// modifying the inverse. This also serves as the inverse of a to-one link.
func (r ManyRelAssignment[P, C, S]) AppendLocal(p *P, children ...*C) {
	slot, _ := r.Slot(p)
	*slot = append(*slot, children...)
}

// Append attaches children and their inverses, preserving the loaded flag.
func (r ManyRelAssignment[P, C, S]) Append(p *P, children ...*C) {
	r.AppendLocal(p, children...)
	r.backfill(p, children)
}

// Link appends one matching child while stitching a batch.
func (r ManyRelAssignment[P, C, S]) Link(p *P, c *C) {
	r.Append(p, c)
}
