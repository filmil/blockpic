// SPDX-License-Identifier: Apache-2.0
package scanner

import (
	"unicode"
)

type TokenType int

const (
	EOF TokenType = iota
	ILLEGAL
	IDENT
	STRING
	NUMBER
	LBRACE
	RBRACE
	LBRACKET
	RBRACKET
	LPAREN
	RPAREN
	COMMA
	COLON
	ARROW
	DOT
	ASSIGN
	SEMICOLON

	// Keywords
	DIAGRAM
	BLOCK
	WIRE
	LAYOUT
	GROUP
	KIND
	NAME
	INPUTS
	OUTPUTS
	PARAMS
	SIZE
	HINT
	THICKNESS
	CROSSLINE
	AT
	TYPE
)

var keywords = map[string]TokenType{
	"diagram":   DIAGRAM,
	"block":     BLOCK,
	"wire":      WIRE,
	"layout":    LAYOUT,
	"group":     GROUP,
	"kind":      KIND,
	"name":      NAME,
	"inputs":    INPUTS,
	"outputs":   OUTPUTS,
	"params":    PARAMS,
	"size":      SIZE,
	"hint":      HINT,
	"thickness": THICKNESS,
	"crossline": CROSSLINE,
	"at":        AT,
	"type":      TYPE,
}

type Token struct {
	Type    TokenType
	Literal string
	Line    int
}

type Scanner struct {
	input []rune
	pos   int
	line  int
}

func New(input string) *Scanner {
	return &Scanner{input: []rune(input), line: 1}
}

func (s *Scanner) NextToken() Token {
	s.skipWhitespace()

	if s.pos >= len(s.input) {
		return Token{Type: EOF, Line: s.line}
	}

	ch := s.input[s.pos]

	switch {
	case isLetter(ch):
		ident := s.readIdentifier()
		if tokType, ok := keywords[ident]; ok {
			return Token{Type: tokType, Literal: ident, Line: s.line}
		}
		return Token{Type: IDENT, Literal: ident, Line: s.line}
	case isDigit(ch):
		num := s.readNumber()
		return Token{Type: NUMBER, Literal: num, Line: s.line}
	case ch == '"':
		str := s.readString()
		return Token{Type: STRING, Literal: str, Line: s.line}
	case ch == '{':
		s.pos++
		return Token{Type: LBRACE, Literal: "{", Line: s.line}
	case ch == '}':
		s.pos++
		return Token{Type: RBRACE, Literal: "}", Line: s.line}
	case ch == '[':
		s.pos++
		return Token{Type: LBRACKET, Literal: "[", Line: s.line}
	case ch == ']':
		s.pos++
		return Token{Type: RBRACKET, Literal: "]", Line: s.line}
	case ch == '(':
		s.pos++
		return Token{Type: LPAREN, Literal: "(", Line: s.line}
	case ch == ')':
		s.pos++
		return Token{Type: RPAREN, Literal: ")", Line: s.line}
	case ch == ',':
		s.pos++
		return Token{Type: COMMA, Literal: ",", Line: s.line}
	case ch == ':':
		s.pos++
		return Token{Type: COLON, Literal: ":", Line: s.line}
	case ch == '.':
		s.pos++
		return Token{Type: DOT, Literal: ".", Line: s.line}
	case ch == '=':
		s.pos++
		return Token{Type: ASSIGN, Literal: "=", Line: s.line}
	case ch == ';':
		s.pos++
		return Token{Type: SEMICOLON, Literal: ";", Line: s.line}
	case ch == '-':
		if s.pos+1 < len(s.input) && s.input[s.pos+1] == '>' {
			s.pos += 2
			return Token{Type: ARROW, Literal: "->", Line: s.line}
		}
		s.pos++
		return Token{Type: ILLEGAL, Literal: "-", Line: s.line}
	case ch == '/':
		if s.pos+1 < len(s.input) && s.input[s.pos+1] == '/' {
			s.skipComment()
			return s.NextToken()
		}
		s.pos++
		return Token{Type: ILLEGAL, Literal: "/", Line: s.line}
	default:
		s.pos++
		return Token{Type: ILLEGAL, Literal: string(ch), Line: s.line}
	}
}

func (s *Scanner) skipWhitespace() {
	for s.pos < len(s.input) && unicode.IsSpace(s.input[s.pos]) {
		if s.input[s.pos] == '\n' {
			s.line++
		}
		s.pos++
	}
}

func (s *Scanner) skipComment() {
	for s.pos < len(s.input) && s.input[s.pos] != '\n' {
		s.pos++
	}
}

func (s *Scanner) readIdentifier() string {
	start := s.pos
	for s.pos < len(s.input) && (isLetter(s.input[s.pos]) || isDigit(s.input[s.pos]) || s.input[s.pos] == '_' || s.input[s.pos] == '~') {
		s.pos++
	}
	return string(s.input[start:s.pos])
}

func (s *Scanner) readNumber() string {
	start := s.pos
	for s.pos < len(s.input) && (isDigit(s.input[s.pos]) || s.input[s.pos] == '.') {
		s.pos++
	}
	return string(s.input[start:s.pos])
}

func (s *Scanner) readString() string {
	s.pos++ // skip "
	start := s.pos
	for s.pos < len(s.input) && s.input[s.pos] != '"' {
		s.pos++
	}
	res := string(s.input[start:s.pos])
	if s.pos < len(s.input) {
		s.pos++ // skip "
	}
	return res
}

func isLetter(ch rune) bool {
	return unicode.IsLetter(ch) || ch == '_' || ch == '~'
}

func isDigit(ch rune) bool {
	return unicode.IsDigit(ch)
}
