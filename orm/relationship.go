package orm

type RelSide struct {
	From     string
	To       string
	Pairs    map[string]string
	ToUnique bool
}

type Relationship struct {
	Name  string
	Sides []RelSide
}

func (r Relationship) Local() string {
	return r.Sides[0].From
}

func (r Relationship) Foreign() string {
	return r.Sides[len(r.Sides)-1].To
}
