package internal

import (
	"fmt"
	"slices"
	"strings"
)

func EditString(s string, rules ...EditRule) (string, error) {
	return EditStringSegment(s, 0, len(s)-1, rules...)
}

func EditStringSegment(s string, from, to int, rules ...EditRule) (string, error) {
	// sort rules by priority
	slices.SortFunc(rules, func(i, j EditRule) int {
		ip1, ip2 := i.rulePriority()
		jp1, jp2 := j.rulePriority()

		if ip1 != jp1 {
			return ip1 - jp1
		}

		return ip2 - jp2
	})

	cursor := from // current position in the original string
	s = s[:to+1]   // limit the string to the range

	var err error
	var buf strings.Builder

	for i, r := range rules {
		if cursor, err = r.edit(s, &buf, cursor); err != nil {
			return "", fmt.Errorf("rule %d: cursor %d, %w", i, cursor, err)
		}
	}

	return buf.String() + s[cursor:], nil
}

type OutOfBoundsError int

func (e OutOfBoundsError) Error() string {
	return fmt.Sprintf("out of bounds: %d", int(e))
}

func (e OutOfBoundsError) Is(target error) bool {
	t, ok := target.(OutOfBoundsError)
	return ok && t == e
}

type EditRule interface {
	isEditRule()
	rulePriority() (int, int)
	edit(source string, buf *strings.Builder, cursor int) (int, error)
}

type deleteRule struct{ from, to int }

func (deleteRule) isEditRule() {}

func (d deleteRule) rulePriority() (int, int) {
	return d.from, 2
}

func (d deleteRule) edit(source string, buf *strings.Builder, cursor int) (int, error) {
	if d.from > d.to {
		return cursor, fmt.Errorf("delete: from(%d) > to(%d)", d.from, d.to)
	}
	from := d.from - cursor
	if from < 0 {
		return cursor, fmt.Errorf("delete: %w", OutOfBoundsError(d.from))
	}
	buf.WriteString(source[cursor:(cursor + from)])
	cursor = d.to + 1
	if len(source) < cursor {
		return cursor, fmt.Errorf("delete: %w", OutOfBoundsError(d.to))
	}

	return cursor, nil
}

func Delete(from, to int) deleteRule {
	return deleteRule{from, to}
}

type insertRule struct {
	pos     int
	content string
}

func (insertRule) isEditRule() {}

func (i insertRule) rulePriority() (int, int) {
	return i.pos, 0
}

func (i insertRule) edit(source string, buf *strings.Builder, cursor int) (int, error) {
	from := i.pos - cursor
	if from < 0 {
		return cursor, fmt.Errorf("insert: %w", OutOfBoundsError(i.pos))
	}
	buf.WriteString(source[cursor:(cursor + from)])
	buf.WriteString(i.content)

	return i.pos, nil
}

func Insert(pos int, content string) insertRule {
	return insertRule{pos, content}
}

type replaceRule struct {
	from, to int
	content  string
}

func (replaceRule) isEditRule() {}

func (r replaceRule) rulePriority() (int, int) {
	return r.from, 1
}

func (r replaceRule) edit(source string, buf *strings.Builder, cursor int) (int, error) {
	var err error
	if cursor, err = Insert(r.from, r.content).edit(source, buf, cursor); err != nil {
		return cursor, fmt.Errorf("replace: %w", err)
	}

	if cursor, err = Delete(r.from, r.to).edit(source, buf, cursor); err != nil {
		return cursor, fmt.Errorf("replace: %w", err)
	}

	return cursor, nil
}

func Replace(from, to int, content string) replaceRule {
	return replaceRule{from, to, content}
}
