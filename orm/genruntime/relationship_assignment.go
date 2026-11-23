package genruntime

// OneRelAssignment sets a to-one relationship on a generated model.
type OneRelAssignment[P, C any] struct {
	slot     func(*P) (**C, *bool)
	backlink func(*C, *P)
}

// ToOne binds a relationship field and its loaded flag. The optional backlink
// updates only the inverse field; it must not recursively link the parent.
func ToOne[P, C any](slot func(*P) (**C, *bool), backlink func(*C, *P)) OneRelAssignment[P, C] {
	return OneRelAssignment[P, C]{slot: slot, backlink: backlink}
}

// Reset discards a previous result. A successful batch load resets all
// parents to loaded, including parents with no matching child.
func (r OneRelAssignment[P, C]) Reset(p *P, loaded bool) {
	slot, flag := r.slot(p)
	*slot, *flag = nil, loaded
}

// SetLocal records the known child without changing the inverse.
func (r OneRelAssignment[P, C]) SetLocal(p *P, c *C) {
	slot, loaded := r.slot(p)
	*slot, *loaded = c, true
}

// Replace records a complete result, including a known absent (nil) child.
func (r OneRelAssignment[P, C]) Replace(p *P, c *C) {
	r.SetLocal(p, c)
	if c != nil && r.backlink != nil {
		r.backlink(c, p)
	}
}

// Link keeps the first matching child while stitching a batch.
func (r OneRelAssignment[P, C]) Link(p *P, c *C) {
	slot, _ := r.slot(p)
	if *slot == nil {
		r.Replace(p, c)
	}
}

// ManyRelAssignment sets a to-many relationship on a generated model.
type ManyRelAssignment[P, C any, S ~[]*C] struct {
	slot     func(*P) (*S, *bool)
	backlink func(*C, *P)
}

// ToMany binds a relationship slice and its loaded flag to the optional backlink.
func ToMany[P, C any, S ~[]*C](slot func(*P) (*S, *bool), backlink func(*C, *P)) ManyRelAssignment[P, C, S] {
	return ManyRelAssignment[P, C, S]{slot: slot, backlink: backlink}
}

func (r ManyRelAssignment[P, C, S]) Reset(p *P, loaded bool) {
	slot, flag := r.slot(p)
	*slot, *flag = nil, loaded
}

func (r ManyRelAssignment[P, C, S]) backfill(p *P, children S) {
	if r.backlink != nil {
		for _, c := range children {
			if c != nil {
				r.backlink(c, p)
			}
		}
	}
}

// Replace installs the complete slice as returned, preserving nil vs empty.
func (r ManyRelAssignment[P, C, S]) Replace(p *P, children S) {
	slot, loaded := r.slot(p)
	*slot, *loaded = children, true
	r.backfill(p, children)
}

// AppendLocal adds known children without establishing a complete load or
// modifying the inverse. This also serves as the inverse of a to-one link.
func (r ManyRelAssignment[P, C, S]) AppendLocal(p *P, children ...*C) {
	slot, _ := r.slot(p)
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
