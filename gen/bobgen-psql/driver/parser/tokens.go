package parser

import (
	"fmt"
	"slices"
	"sort"
	"strings"

	pg "github.com/pganalyze/pg_query_go/v6"
)

func (w walker) getEnd(start int32) int32 {
	if len(w.tokens) == 0 || start == -1 {
		return -1
	}

	index := sort.Search(len(w.tokens), func(i int) bool {
		return w.tokens[i].Start >= start
	})
	if index < len(w.tokens) {
		return w.tokens[index].End
	}

	return -1
}

func (w walker) findIdentOrUnreserved(pos int32) nodeInfo {
	return w.findTokenAfterFunc(pos, func(_ int, t *pg.ScanToken) bool {
		return t.Token == pg.Token_IDENT || t.KeywordKind == pg.KeywordKind_UNRESERVED_KEYWORD
	})
}

func (w walker) findTokenBefore(position int32, tokens ...pg.Token) nodeInfo {
	return w.findTokenBeforeFunc(position, func(t *pg.ScanToken) bool {
		return slices.Contains(tokens, t.Token)
	})
}

func (w walker) findTokenBeforeFunc(position int32, f func(*pg.ScanToken) bool) nodeInfo {
	if len(w.tokens) == 0 || position == -1 {
		return newNodeInfo()
	}

	index := sort.Search(len(w.tokens), func(i int) bool {
		return w.tokens[i].Start >= position
	})

	for i := index; i >= 0; i-- {
		if f(w.tokens[i]) {
			return nodeInfo{
				start: w.tokens[i].Start,
				end:   w.tokens[i].End,
			}
		}
	}

	return newNodeInfo()
}

func (w walker) findTokenAfter(position int32, tokens ...pg.Token) nodeInfo {
	return w.findTokenAfterFunc(position, func(_ int, t *pg.ScanToken) bool {
		return slices.Contains(tokens, t.Token)
	})
}

func (w walker) findTokenAfterFunc(position int32, f func(int, *pg.ScanToken) bool) nodeInfo {
	if len(w.tokens) == 0 || position == -1 {
		return newNodeInfo()
	}

	index := sort.Search(len(w.tokens), func(i int) bool {
		return w.tokens[i].End > position
	})

	for i := index; i < len(w.tokens); i++ {
		if f(i, w.tokens[i]) {
			return nodeInfo{
				start: w.tokens[i].Start,
				end:   w.tokens[i].End,
			}
		}
	}

	return newNodeInfo()
}

func (w walker) getStartOfTokenBefore(start int32, tokens ...pg.Token) int32 {
	if len(w.tokens) == 0 {
		return start
	}

	index := sort.Search(len(w.tokens), func(i int) bool {
		return w.tokens[i].Start >= start
	})

	for i := index; i >= 0; i-- {
		if slices.Contains(tokens, w.tokens[i].Token) {
			return w.tokens[i].Start
		}
	}

	return start
}

func (w walker) getEndOfTokenAfter(end int32, tokens ...pg.Token) int32 {
	if len(w.tokens) == 0 {
		return end
	}

	index := sort.Search(len(w.tokens), func(i int) bool {
		return w.tokens[i].End >= end
	})

	for i := index; i < len(w.tokens); i++ {
		if slices.Contains(tokens, w.tokens[i].Token) {
			return w.tokens[i].End
		}
	}

	return end
}

func (w walker) balanceParenthesis(info nodeInfo) nodeInfo {
	if len(w.tokens) == 0 || info.start == -1 || info.end == -1 {
		return info
	}

	startIndex := sort.Search(len(w.tokens), func(i int) bool {
		return w.tokens[i].Start >= info.start
	})

	if startIndex >= len(w.tokens) {
		return info
	}

	count := 0
	endIndex := startIndex

	for endIndex < len(w.tokens) && w.tokens[endIndex].End <= info.end {
		switch w.tokens[endIndex].Token {
		case openParToken:
			count++
		case closeParToken:
			count--
		}

		if w.tokens[endIndex].End >= info.end {
			break
		}

		endIndex++
	}

	// Increase the endIndex so we start at the next token
	endIndex++
	for endIndex < len(w.tokens) && count > 0 {
		switch w.tokens[endIndex].Token {
		case openParToken:
			count++
		case closeParToken:
			count--
		}

		if count == 0 {
			info.end = w.tokens[endIndex].End
			return info
		}

		endIndex++
	}

	// Decrease the startIndex so we start at the previous token
	startIndex--
	for startIndex >= 0 && count < 0 {
		switch w.tokens[startIndex].Token {
		case openParToken:
			count++
		case closeParToken:
			count--
		}

		if count == 0 {
			info.start = w.tokens[startIndex].Start
			return info
		}

		startIndex--
	}

	return info
}

func (w walker) getQueryComment(pos int32) (string, error) {
	token := w.findTokenBefore(pos, pg.Token_SQL_COMMENT)
	if token.start == -1 {
		return "", fmt.Errorf("no comment before keyword: %s", token.String())
	}

	comment := w.input[token.start+2 : token.end]
	return strings.TrimSpace(comment), nil
}

func (w walker) getConfigComment(pos position) string {
	if pos == (position{}) {
		return ""
	}

	index := sort.Search(len(w.tokens), func(i int) bool {
		return w.tokens[i].End > pos[1]
	})

	if index >= len(w.tokens) {
		return ""
	}

	nextToken := w.tokens[index]
	if nextToken.GetToken() != pg.Token_C_COMMENT {
		return ""
	}

	comment := w.input[nextToken.GetStart()+2 : nextToken.GetEnd()-2]
	return strings.TrimSpace(comment)
}
