package expr

// ColumnName supplies column metadata to generated dialect expressions.
type ColumnName string

// Name returns the unqualified SQL column name.
func (c ColumnName) Name() string { return string(c) }

// ShouldOmitParens prevents expression builders from grouping a column reference.
func (ColumnName) ShouldOmitParens() bool { return true }
