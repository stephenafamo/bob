package antlrhelpers

func KnownType(t string, nullable func() bool) NodeType {
	return NodeType{DBType: t, NullableF: nullable}
}

func KnownTypeNull(t string) NodeType {
	return NodeType{DBType: t, NullableF: Nullable}
}

func KnownTypeNotNull(t string) NodeType {
	return NodeType{DBType: t, NullableF: NotNullable}
}

func Nullable() bool {
	return true
}

func NotNullable() bool {
	return false
}

func AnyNullable(fs ...func() bool) func() bool {
	return func() bool {
		for _, f := range fs {
			if f() {
				return true
			}
		}

		return false
	}
}

func AllNullable(fs ...func() bool) func() bool {
	return func() bool {
		for _, f := range fs {
			if !f() {
				return false
			}
		}

		return true
	}
}

func NeverNullable(...func() bool) func() bool {
	return NotNullable
}
