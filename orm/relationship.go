package orm

type RelSide struct {
	From  string
	To    string
	Pairs map[string]string
}

type Relationship struct {
	Name  string
	Sides []RelSide
}
