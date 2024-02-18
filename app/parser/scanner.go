package parser

import (
	"log"
	"strconv"
)

type TokenType int

const (
	EOF TokenType = iota
	Plus
	Minus
	Dollar
	Asterisk
	String
	Number
)

type Token struct {
	TokenType TokenType
	Lexeme    string
	Literal   any
	Line      int
}

type Scanner struct {
	source  string
	tokens  []Token
	start   int
	current int
	line    int
}

func NewScanner(source string) *Scanner {
	return &Scanner{
		source:  source,
		start:   0,
		current: 0,
		line:    1,
	}
}

func (s *Scanner) ScanTokens() []Token {
	for !s.isAtEnd() {
		s.start = s.current
		s.scanToken()
	}
	s.tokens = append(s.tokens, Token{EOF, "", nil, s.line})
	return s.tokens
}

func (s *Scanner) scanToken() {
	c := s.source[s.current]
	s.current++

	switch c {
	case '*':
		s.addToken(Asterisk, nil)
	case '$':
		s.addToken(Dollar, nil)
	case '+':
		s.addToken(Plus, nil)
	case '\n':
		s.line++
	case ' ':
	case '\r':
	case '\t':
		break
	default:
		if isDigit(c) {
			s.scanNumber()
		} else if isAlfa(c) {
			s.scanString()
		} else {
			log.Println("Unknown token")
		}
	}
}

func (s *Scanner) addToken(tokenType TokenType, literal any) {
	lexeme := s.source[s.start:s.current]
	s.tokens = append(s.tokens, Token{tokenType, lexeme, literal, s.line})
}

func (s *Scanner) scanString() {
	for isAlfa(s.peek()) && !s.isAtEnd() {
		s.current++
	}

	tokenType := String
	literal := s.source[s.start:s.current]

	s.addToken(tokenType, literal)
}

func (s *Scanner) scanNumber() {
	for isDigit(s.peek()) {
		s.current++
	}

	value, err := strconv.Atoi(s.source[s.start:s.current])

	if err != nil {
		return
	}

	s.addToken(Number, value)
}

func (s *Scanner) peek() byte {
	if s.isAtEnd() {
		return '\000'
	}
	return s.source[s.current]
}

func (s *Scanner) peekNext() byte {
	if s.current+1 > len(s.source) {
		return '\000'
	}
	return s.source[s.current+1]
}

func (s Scanner) isAtEnd() bool {
	return s.current >= len(s.source)
}

func isDigit(c byte) bool {
	return c >= '0' && c <= '9'
}

func isAlfa(c byte) bool {
	return (c >= 'a' && c <= 'z') ||
		(c >= 'A' && c <= 'Z') ||
		c == '_'
}
