package pgtypes

import (
	"database/sql/driver"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var tsVectorRgx = regexp.MustCompile(`'((?:[^']|'')+)'(?::((?:,?\d+(?:A|B|C|D)?)+))?`)

type rankAndPos struct {
	pos  int
	rank rune // A, B, C or D
}

type lexeme struct {
	word      string
	positions []rankAndPos
}

type TSVector struct {
	lexemes []lexeme
}

// String returns the string representation of the TSVector.
func (t TSVector) String() string {
	if len(t.lexemes) == 0 {
		return ""
	}

	var s strings.Builder
	for i, lex := range t.lexemes {
		if i > 0 {
			s.WriteString(" ")
		}

		fmt.Fprintf(&s, "'%s'", lex.word)

		if len(lex.positions) == 0 {
			continue
		}

		s.WriteString(":")
		for j, pos := range lex.positions {
			if j > 0 {
				s.WriteString(",")
			}
			s.WriteString(fmt.Sprintf("%d", pos.pos))
			if pos.rank != 0 {
				s.WriteString(string(pos.rank))
			}
		}
	}

	return s.String()
}

// Value implements the driver.Valuer interface.
func (t TSVector) Value() (driver.Value, error) {
	return t.String(), nil
}

// Scan implements the sql.Scanner interface.
func (t *TSVector) Scan(src any) error {
	if src == nil {
		return nil
	}

	var val string
	var err error

	switch src := src.(type) {
	case string:
		val = src
	case []byte:
		val = string(src)
	default:
		return fmt.Errorf("cannot scan %T into TSVector", src)
	}

	s := tsVectorRgx.FindAllStringSubmatch(val, -1)

	lexemes := make([]lexeme, len(s))
	for i, match := range s {
		lexemes[i].word = match[1]
		if len(match[2]) == 0 {
			continue
		}

		positions := strings.Split(match[2], ",")
		if len(positions) == 0 {
			continue
		}

		lexemes[i].positions = make([]rankAndPos, len(positions))
		for j, position := range positions {
			var rank rune
			var pos int

			if len(position) == 0 {
				return fmt.Errorf("empty position")
			}

			switch position[len(position)-1] {
			case 'A', 'B', 'C', 'D':
				rank = rune(position[len(position)-1])
				pos, err = strconv.Atoi(position[:len(position)-1])

			default:
				pos, err = strconv.Atoi(position)
			}

			if err != nil {
				return fmt.Errorf("converting position %q: %w", position, err)
			}

			lexemes[i].positions[j] = rankAndPos{
				rank: rank,
				pos:  pos,
			}
		}
	}

	t.lexemes = lexemes

	return nil
}
