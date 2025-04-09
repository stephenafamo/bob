package internal

import (
	"fmt"
	"slices"
	"strings"
)

type RuleType int

const (
	CopyRuleType RuleType = iota
	InsertRuleType
	ReplaceRuleType
	DeleteRuleType
)

func (r RuleType) String() string {
	switch r {
	case CopyRuleType:
		return "copy"
	case InsertRuleType:
		return "insert"
	case ReplaceRuleType:
		return "replace"
	case DeleteRuleType:
		return "delete"
	default:
		return "unknown"
	}
}

func EditString(s string, rules ...EditRule) (string, error) {
	return EditStringSegment(s, 0, len(s)-1, rules...)
}

func EditStringSegment(s string, from, to int, rules ...EditRule) (string, error) {
	// sort rules by priority
	slices.SortStableFunc(rules, func(i, j EditRule) int {
		iStart, _ := i.position()
		jStart, _ := j.position()

		if iStart != jStart {
			return iStart - jStart
		}

		return int(i.ruleType() - j.ruleType())
	})

	cursor := from // current position in the original string
	s = s[:to+1]   // limit the string to the range

	var err error
	var buf strings.Builder

	for i, r := range rules {
		start, end := r.position()

		if start < cursor {
			// rule starts before cursor, skip it
			// fmt.Printf("Skipping rule %d: %s[%d-%d] starts before cursor(%d), %#v\n", i, r.ruleType(), start, end, cursor, r)
			continue
		}

		if end < cursor {
			// rule is before cursor, skip it
			// fmt.Printf("Skipping rule %d: %s[%d-%d] ends before cursor(%d), %#v\n", i, r.ruleType(), start, end, cursor, r)
			continue
		}

		if start > len(s) {
			// rule is after the string, skip it
			// fmt.Printf("Skipping rule %d: %s[%d-%d] out of bounds(%d), %#v\n", i, r.ruleType(), start, end, len(s), r)
			continue
		}

		// fmt.Printf("Applying rule %d: %s[%d-%d] cursor(%d), %#v\n", i, r.ruleType(), start, end, cursor, r)

		buf.WriteString(s[cursor:start])
		cursor = start

		if err = r.edit(s, &buf); err != nil {
			return "", fmt.Errorf("rule %d: cursor %d, %w", i, cursor, err)
		}

		cursor = end
		if cursor > len(s) {
			return "", fmt.Errorf("out of segment cursor(%d) %s[%d-%d]: %w", len(s), r.ruleType(), start, end, OutOfBoundsError(cursor))
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
	position() (int, int)
	ruleType() RuleType
	edit(source string, buf *strings.Builder) error
}

type deleteRule struct{ from, to int }

func (d deleteRule) position() (int, int) {
	return d.from, d.to + 1
}

func (d deleteRule) ruleType() RuleType {
	return DeleteRuleType
}

func (d deleteRule) edit(source string, buf *strings.Builder) error {
	return nil
}

func Delete(from, to int) deleteRule {
	return deleteRule{from, to}
}

type insertRule struct {
	pos     int
	content func() string
}

func (i insertRule) position() (int, int) {
	return i.pos, i.pos
}

func (i insertRule) ruleType() RuleType {
	return InsertRuleType
}

func (i insertRule) edit(source string, buf *strings.Builder) error {
	if _, err := buf.WriteString(i.content()); err != nil {
		return fmt.Errorf("insert: %w", err)
	}
	return nil
}

func Insert(pos int, content string) insertRule {
	return insertRule{pos, func() string { return content }}
}

func InsertFromFunc(pos int, content func() string) insertRule {
	return insertRule{pos, content}
}

type replaceRule struct {
	from, to int
	content  func() string
}

func (r replaceRule) position() (int, int) {
	return Delete(r.from, r.to).position()
}

func (r replaceRule) ruleType() RuleType {
	return ReplaceRuleType
}

func (r replaceRule) edit(source string, buf *strings.Builder) error {
	var err error
	if err = Insert(r.from, r.content()).edit(source, buf); err != nil {
		return fmt.Errorf("replace: %w", err)
	}

	if err = Delete(r.from, r.to).edit(source, buf); err != nil {
		return fmt.Errorf("replace: %w", err)
	}

	return nil
}

func Replace(from, to int, content string) replaceRule {
	return replaceRule{from, to, func() string { return content }}
}

func ReplaceFromFunc(from, to int, content func() string) replaceRule {
	return replaceRule{from, to, content}
}

type callbackRule struct {
	EditRule
	callbacks []func(start, end int, before, after string) error
}

func (r callbackRule) edit(source string, buf *strings.Builder) error {
	start := buf.Len()

	ruleStart, ruleEnd := r.EditRule.position()

	if err := r.EditRule.edit(source, buf); err != nil {
		return err
	}

	end := buf.Len()

	for _, cb := range r.callbacks {
		if err := cb(start, end, source[ruleStart:ruleEnd], buf.String()[start:]); err != nil {
			return err
		}
	}

	return nil
}

func EditCallback(rule EditRule, callbacks ...func(start, end int, before, after string) error) callbackRule {
	return callbackRule{rule, callbacks}
}

func RecordPoints(oldStart, oldEnd int, callbacks ...func(start, end int) error) []EditRule {
	firstPoint := 0
	return []EditRule{
		EditCallback(
			Insert(oldStart, ""),
			func(start int, _ int, _, _ string) error { firstPoint = start; return nil },
		),
		EditCallback(
			Insert(oldEnd+1, ""),
			func(_, end int, _, content string) error {
				for _, cb := range callbacks {
					if err := cb(firstPoint, end); err != nil {
						return err
					}
				}
				return nil
			},
		),
	}
}
