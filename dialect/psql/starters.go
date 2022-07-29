package psql

func F(name string, args ...any) *function {
	f := &function{
		name: name,
		args: args,
	}

	// We have embeded the same function as the chain base
	// this is so that chained methods can also be used by functions
	f.Chain.Base = f

	return f
}

func S(s string) chain {
	return bmod.S(s)
}

func X(exp any) chain {
	return bmod.X(exp)
}

func Not(exp any) chain {
	return bmod.Not(exp)
}

func Or(args ...any) chain {
	return bmod.Or(args...)
}

func And(args ...any) chain {
	return bmod.And(args...)
}

func Concat(args ...any) chain {
	return bmod.Concat(args...)
}

func Arg(args ...any) chain {
	return bmod.Arg(args...)
}

func P(exp any) chain {
	return bmod.P(exp)
}

func Placeholder(n uint) chain {
	return bmod.Placeholder(n)
}

func Raw(query string, args ...any) chain {
	return bmod.Raw(query, args...)
}

func Group(exps ...any) chain {
	return bmod.Group(exps...)
}

func Quote(ss ...string) chain {
	return bmod.Quote(ss...)
}
