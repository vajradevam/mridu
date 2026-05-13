package lang

import "fmt"

type TokenType int

const (
	TOKEN_LEFT_PAREN TokenType = iota
	TOKEN_RIGHT_PAREN
	TOKEN_LEFT_BRACE
	TOKEN_RIGHT_BRACE
	TOKEN_COMMA
	TOKEN_DOT
	TOKEN_MINUS
	TOKEN_PLUS
	TOKEN_SEMICOLON
	TOKEN_SLASH
	TOKEN_STAR
	TOKEN_BANG
	TOKEN_BANG_EQUAL
	TOKEN_EQUAL
	TOKEN_EQUAL_EQUAL
	TOKEN_GREATER
	TOKEN_GREATER_EQUAL
	TOKEN_LESS
	TOKEN_LESS_EQUAL
	TOKEN_IDENTIFIER
	TOKEN_STRING
	TOKEN_NUMBER
	TOKEN_AND
	TOKEN_CLASS
	TOKEN_ELSE
	TOKEN_FALSE
	TOKEN_FUN
	TOKEN_FOR
	TOKEN_IF
	TOKEN_NIL
	TOKEN_OR
	TOKEN_PRINT
	TOKEN_RETURN
	TOKEN_SUPER
	TOKEN_THIS
	TOKEN_TRUE
	TOKEN_VAR
	TOKEN_WHILE
	TOKEN_ERROR
	TOKEN_EOF
)

type Token struct {
	Type    TokenType
	Lexeme  string
	Literal interface{}
	Line    int
}

type Scanner struct {
	source  string
	start   int
	current int
	line    int
}

func NewScanner(source string) *Scanner {
	return &Scanner{source: source, line: 1}
}

var keywords = map[string]TokenType{
	"and":    TOKEN_AND,
	"class":  TOKEN_CLASS,
	"else":   TOKEN_ELSE,
	"false":  TOKEN_FALSE,
	"for":    TOKEN_FOR,
	"fun":    TOKEN_FUN,
	"if":     TOKEN_IF,
	"nil":    TOKEN_NIL,
	"or":     TOKEN_OR,
	"print":  TOKEN_PRINT,
	"return": TOKEN_RETURN,
	"super":  TOKEN_SUPER,
	"this":   TOKEN_THIS,
	"true":   TOKEN_TRUE,
	"var":    TOKEN_VAR,
	"while":  TOKEN_WHILE,
}

func (s *Scanner) ScanToken() Token {
	s.skipWhitespace()
	s.start = s.current

	if s.isAtEnd() {
		return s.makeToken(TOKEN_EOF)
	}

	c := s.advance()

	if isAlpha(c) {
		return s.identifier()
	}
	if isDigit(c) {
		return s.number()
	}

	switch c {
	case '(':
		return s.makeToken(TOKEN_LEFT_PAREN)
	case ')':
		return s.makeToken(TOKEN_RIGHT_PAREN)
	case '{':
		return s.makeToken(TOKEN_LEFT_BRACE)
	case '}':
		return s.makeToken(TOKEN_RIGHT_BRACE)
	case ',':
		return s.makeToken(TOKEN_COMMA)
	case '.':
		return s.makeToken(TOKEN_DOT)
	case '-':
		return s.makeToken(TOKEN_MINUS)
	case '+':
		return s.makeToken(TOKEN_PLUS)
	case ';':
		return s.makeToken(TOKEN_SEMICOLON)
	case '*':
		return s.makeToken(TOKEN_STAR)
	case '!':
		if s.match('=') {
			return s.makeToken(TOKEN_BANG_EQUAL)
		}
		return s.makeToken(TOKEN_BANG)
	case '=':
		if s.match('=') {
			return s.makeToken(TOKEN_EQUAL_EQUAL)
		}
		return s.makeToken(TOKEN_EQUAL)
	case '<':
		if s.match('=') {
			return s.makeToken(TOKEN_LESS_EQUAL)
		}
		return s.makeToken(TOKEN_LESS)
	case '>':
		if s.match('=') {
			return s.makeToken(TOKEN_GREATER_EQUAL)
		}
		return s.makeToken(TOKEN_GREATER)
	case '/':
		if s.match('/') {
			for s.peek() != '\n' && !s.isAtEnd() {
				s.advance()
			}
		} else if s.match('*') {
			s.blockComment()
		} else {
			return s.makeToken(TOKEN_SLASH)
		}
		return s.ScanToken()
	case '"':
		return s.string()
	case '\n':
		s.line++
		return s.ScanToken()
	}

	return s.errorToken(fmt.Sprintf("Unexpected character: %c", c))
}

func (s *Scanner) advance() byte {
	c := s.source[s.current]
	s.current++
	return c
}

func (s *Scanner) match(expected byte) bool {
	if s.isAtEnd() {
		return false
	}
	if s.source[s.current] != expected {
		return false
	}
	s.current++
	return true
}

func (s *Scanner) peek() byte {
	if s.isAtEnd() {
		return 0
	}
	return s.source[s.current]
}

func (s *Scanner) peekNext() byte {
	if s.current+1 >= len(s.source) {
		return 0
	}
	return s.source[s.current+1]
}

func (s *Scanner) isAtEnd() bool {
	return s.current >= len(s.source)
}

func (s *Scanner) skipWhitespace() {
	for {
		c := s.peek()
		switch c {
		case ' ', '\r', '\t':
			s.advance()
		case '\n':
			s.line++
			s.advance()
		case '/':
			if s.peekNext() == '/' {
				for s.peek() != '\n' && !s.isAtEnd() {
					s.advance()
				}
			} else if s.peekNext() == '*' {
				s.advance()
				s.advance()
				s.blockComment()
			} else {
				return
			}
		default:
			return
		}
	}
}

func (s *Scanner) blockComment() {
	nesting := 1
	for nesting > 0 {
		if s.isAtEnd() {
			return
		}
		if s.peek() == '/' && s.peekNext() == '*' {
			s.advance()
			s.advance()
			nesting++
		} else if s.peek() == '*' && s.peekNext() == '/' {
			s.advance()
			s.advance()
			nesting--
		} else {
			if s.peek() == '\n' {
				s.line++
			}
			s.advance()
		}
	}
}

func (s *Scanner) string() Token {
	for s.peek() != '"' && !s.isAtEnd() {
		if s.peek() == '\n' {
			s.line++
		}
		s.advance()
	}
	if s.isAtEnd() {
		return s.errorToken("Unterminated string.")
	}
	s.advance()
	return s.makeToken(TOKEN_STRING)
}

func (s *Scanner) number() Token {
	for isDigit(s.peek()) {
		s.advance()
	}
	if s.peek() == '.' && isDigit(s.peekNext()) {
		s.advance()
		for isDigit(s.peek()) {
			s.advance()
		}
	}
	return s.makeToken(TOKEN_NUMBER)
}

func (s *Scanner) identifier() Token {
	for isAlphaNumeric(s.peek()) {
		s.advance()
	}
	text := s.source[s.start:s.current]
	if typ, ok := keywords[text]; ok {
		return Token{Type: typ, Lexeme: text, Line: s.line}
	}
	return s.makeToken(TOKEN_IDENTIFIER)
}

func (s *Scanner) makeToken(typ TokenType) Token {
	lexeme := s.source[s.start:s.current]
	var lit interface{}
	if typ == TOKEN_STRING {
		lit = s.source[s.start+1 : s.current-1]
	} else if typ == TOKEN_NUMBER {
		lit = parseFloat(lexeme)
	}
	return Token{Type: typ, Lexeme: lexeme, Literal: lit, Line: s.line}
}

func (s *Scanner) errorToken(msg string) Token {
	return Token{Type: TOKEN_ERROR, Lexeme: msg, Line: s.line}
}

func isAlpha(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_'
}

func isDigit(c byte) bool {
	return c >= '0' && c <= '9'
}

func isAlphaNumeric(c byte) bool {
	return isAlpha(c) || isDigit(c)
}

func parseFloat(s string) float64 {
	var result float64
	var frac float64 = 1
	var dec bool
	for i := 0; i < len(s); i++ {
		if s[i] == '.' {
			dec = true
			continue
		}
		d := float64(s[i] - '0')
		if dec {
			frac *= 10
			result += d / frac
		} else {
			result = result*10 + d
		}
	}
	return result
}
