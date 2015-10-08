package parser

import (
	"bytes"
	"container/list"
	"fmt"
	"regexp"
	"unicode/utf8"
)

const (
	pattern = `(?m:(?P<whitespace>\s+)|` +
		`(?P<terminator>[;])|` +
		`["](?P<string>.+?)["]|` +
		`//(?P<comment>.*)$|` +
		`(?P<identifier>-?[a-zA-Z_0-9][a-zA-Z0-9_.]*)|` +
		`(?P<namespace>::)|` +
		`[#](?P<preprocess>[a-zA-Z]*)|` +
		`(?P<operator>[{}<>\]=|])|` +
		`(?P<invalid>[^\s]+))`
)

const (
	OpWhitespace OpCode = iota
	OpTerminator
	OpString
	OpComment
	OpIdentifier
	OpNamespace
	OpPreprocess
	OpOperator
	OpInvalid
)

var (
	patternRegexp        = regexp.MustCompile(pattern)
	patternGroups        = patternRegexp.SubexpNames()
	patternGroupsOpCodes = map[string]OpCode{
		OpWhitespace.String(): OpWhitespace,
		OpTerminator.String(): OpTerminator,
		OpString.String():     OpString,
		OpComment.String():    OpComment,
		OpIdentifier.String(): OpIdentifier,
		OpNamespace.String():  OpNamespace,
		OpPreprocess.String(): OpPreprocess,
		OpOperator.String():   OpOperator,
		OpInvalid.String():    OpInvalid,
	}
)

type OpCode int

func (op OpCode) String() string {
	switch op {
	case OpWhitespace:
		return "whitespace"
	case OpTerminator:
		return "terminator"
	case OpString:
		return "string"
	case OpComment:
		return "comment"
	case OpIdentifier:
		return "identifier"
	case OpNamespace:
		return "namespace"
	case OpPreprocess:
		return "preprocess"
	case OpOperator:
		return "operator"
	case OpInvalid:
		return "invalid"
	default:
		panic(fmt.Errorf("Unknown OpCode %d", op))
	}
}

type Token struct {
	Op    OpCode
	Name  string
	Value []byte
	Raw   []byte
	Row   int
	Col   int
	Error error
}

func (t *Token) Equal(other *Token) bool {
	return t.Op == other.Op && t.ValueEqual(other.Value)
}

func (t *Token) ValueString() string {
	return string(t.Value)
}

func (t *Token) ValueEqual(val []byte) bool {
	return bytes.EqualFold(t.Value, val)
}

func (t *Token) ValueEqualString(val string) bool {
	return t.ValueEqual([]byte(val))
}

func tokenStringValues(tokens []*Token) []string {
	var values []string

	for _, t := range tokens {
		values = append(values, t.ValueString())
	}

	return values
}

type TokenQueue struct {
	list *list.List
}

func NewTokenQueue() *TokenQueue {
	return &TokenQueue{list: list.New()}
}

func (q *TokenQueue) enqueue(t *Token) {
	q.list.PushBack(t)
}

func (q *TokenQueue) Len() int {
	return q.list.Len()
}

func (q *TokenQueue) Peek() *Token {
	if e := q.list.Front(); e != nil {
		return e.Value.(*Token)
	} else {
		return nil
	}
}

func (q *TokenQueue) Dequeue() *Token {
	if e := q.list.Front(); e != nil {
		q.list.Remove(e)
		return e.Value.(*Token)
	} else {
		return nil
	}
}

type Tokenizer struct {
	data []byte
	pos  int
}

func NewTokenizer(data []byte) *Tokenizer {
	return &Tokenizer{data: data}
}

func (t *Tokenizer) Tokenize() (*TokenQueue, error) {
	q := NewTokenQueue()
	err := t.tokenize(q)
	return q, err
}

func (t *Tokenizer) tokenize(q *TokenQueue) error {
	matchIndexes := patternRegexp.FindAllSubmatchIndex(t.data, -1)
	row := 1
	col := 1

	for _, matchIndex := range matchIndexes {
		for i := 2; i < len(matchIndex); i += 2 {
			startIdx := matchIndex[i]

			if startIdx >= 0 {
				gi := i / 2
				group := patternGroups[gi]
				endIdx := matchIndex[i+1]
				matched := t.data[matchIndex[0]:matchIndex[1]]
				captured := t.data[startIdx:endIdx]
				op, ok := patternGroupsOpCodes[group]

				if !ok {
					return fmt.Errorf("Unknown pattern group %q. This is probably a go-steam-language bug, please report it.", group)
				}

				rows, cols, err := countRunes(matched)

				if err != nil {
					return err
				}

				row += rows

				if rows > 0 {
					col = cols
				} else {
					col += cols
				}

				if group == "comment" || group == "whitespace" {
					break
				}

				token := &Token{
					Op:    op,
					Name:  op.String(),
					Value: captured,
					Raw:   matched,
					Row:   row,
					Col:   col,
				}

				q.enqueue(token)

				break
			}
		}
	}

	return nil
}

func countRunes(data []byte) (int, int, error) {
	rows := 0
	cols := 0
	pos := 0
	n := len(data)

	for pos < n {
		r, s := utf8.DecodeRune(data[pos:])

		switch r {
		case '\n':
			rows += 1
			cols = 0
		case utf8.RuneError:
			return -1, -1, fmt.Errorf("Invalid UTF-8 char")
		default:
			cols += 1
		}

		pos += s
	}

	return rows, cols, nil
}
