package orm

import (
	"io"

	"github.com/stephenafamo/bob"
)

func sliceToMap[T comparable](s []T) map[T]int {
	m := make(map[T]int, len(s))
	for k, v := range s {
		m[v] = k
	}
	return m
}

func NewColumns(names []string) Columns {
	return Columns{
		names: names,
	}
}

type Columns struct {
	parent      []string
	names       []string
	aggFunc     [2]string
	aliasPrefix string
}

func (c Columns) Names() []string {
	names := make([]string, len(c.names))
	copy(names, c.names)
	return names
}

func (c Columns) WithAggFunc(a, b string) Columns {
	c.aggFunc = [2]string{a, b}
	return c
}

func (c Columns) WithParent(p ...string) Columns {
	c.parent = p
	return c
}

func (c Columns) WithPrefix(prefix string) Columns {
	c.aliasPrefix = prefix
	return c
}

func (c Columns) Only(cols ...string) Columns {
	filtered := make([]string, 0, len(cols)) // max capacity is the only list

	filteredMap := sliceToMap(cols)
	for _, basecol := range c.names {
		if len(basecol) == 0 {
			continue
		}
		if _, ok := filteredMap[basecol]; !ok {
			continue
		}

		filtered = append(filtered, basecol)
	}

	c.names = filtered
	return c
}

func (c Columns) Except(cols ...string) Columns {
	filtered := make([]string, 0, len(c.names)) // max capacity is current capacity

	filteredMap := sliceToMap(cols)
	for _, basecol := range c.names {
		if len(basecol) == 0 {
			continue
		}
		if _, ok := filteredMap[basecol]; ok {
			continue
		}

		filtered = append(filtered, basecol)
	}

	c.names = filtered
	return c
}

func (c Columns) WriteSQL(w io.Writer, d bob.Dialect, start int) ([]any, error) {
	if len(c.names) == 0 {
		return nil, nil
	}

	// wrap in parenthesis and join with comma
	for k, col := range c.names {
		if k != 0 {
			w.Write([]byte(", "))
		}

		w.Write([]byte(c.aggFunc[0]))
		for _, part := range c.parent {
			d.WriteQuoted(w, part)
			w.Write([]byte("."))
		}

		d.WriteQuoted(w, col)
		w.Write([]byte(c.aggFunc[1]))

		w.Write([]byte(" AS "))
		d.WriteQuoted(w, c.aliasPrefix+col)
	}

	return nil, nil
}
