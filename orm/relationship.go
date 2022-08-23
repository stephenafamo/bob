package orm

type Relationship struct {
	Name string

	LocalTable   string
	ForeignTable string

	// pairs of local to foreign columns
	ColumnPairs map[string]string
}
