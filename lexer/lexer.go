package lexer

import (
	"picox/token"
)

type Lexer struct {
	input        string
	position     int  // 当前字符位置
	readPosition int  // 下一个字符位置
	ch           byte // 当前字符
	line         int
	column       int
}

func New(input string) *Lexer {
	l := &Lexer{input: input, line: 1, column: 0}
	l.readChar()
	return l
}

func (l *Lexer) readChar() {
	if l.readPosition >= len(l.input) {
		l.ch = 0
	} else {
		l.ch = l.input[l.readPosition]
	}
	l.position = l.readPosition
	l.readPosition++
	if l.ch == '\n' {
		l.line++
		l.column = 0
	} else {
		l.column++
	}
}

func (l *Lexer) peekChar() byte {
	if l.readPosition >= len(l.input) {
		return 0
	}
	return l.input[l.readPosition]
}

func (l *Lexer) NextToken() token.Token {
	l.skipWhitespace()

	// 记录 token 起始位置
	line, col := l.line, l.column

	makeTok := func(t token.TokenType, lit string) token.Token {
		return token.Token{Type: t, Literal: lit, Line: line, Column: col}
	}

	switch l.ch {
	case '=':
		switch l.peekChar() {
		case '=':
			ch := l.ch
			l.readChar()
			tok := makeTok(token.EQ, string(ch)+string(l.ch))
			l.readChar()
			return tok
		case '>':
			ch := l.ch
			l.readChar()
			tok := makeTok(token.ARROW, string(ch)+string(l.ch))
			l.readChar()
			return tok
		default:
			tok := makeTok(token.ASSIGN, string(l.ch))
			l.readChar()
			return tok
		}
	case '+':
		if l.peekChar() == '+' {
			l.readChar()
			tok := makeTok(token.INCREMENT, "++")
			l.readChar()
			return tok
		}
		if l.peekChar() == '=' {
			l.readChar()
			tok := makeTok(token.PLUS_ASSIGN, "+=")
			l.readChar()
			return tok
		}
		tok := makeTok(token.PLUS, string(l.ch))
		l.readChar()
		return tok
	case '-':
		if l.peekChar() == '-' {
			l.readChar()
			tok := makeTok(token.DECREMENT, "--")
			l.readChar()
			return tok
		}
		if l.peekChar() == '=' {
			l.readChar()
			tok := makeTok(token.MINUS_ASSIGN, "-=")
			l.readChar()
			return tok
		}
		tok := makeTok(token.MINUS, string(l.ch))
		l.readChar()
		return tok
	case '*':
		if l.peekChar() == '*' {
			l.readChar()
			tok := makeTok(token.POW, "**")
			l.readChar()
			return tok
		}
		if l.peekChar() == '=' {
			l.readChar()
			tok := makeTok(token.MUL_ASSIGN, "*=")
			l.readChar()
			return tok
		}
		tok := makeTok(token.ASTERISK, string(l.ch))
		l.readChar()
		return tok
	case '/':
		if l.peekChar() == '/' {
			l.readChar()
			l.skipComment()
			return l.NextToken()
		} else if l.peekChar() == '*' {
			l.readChar()
			l.skipMultiLineComment()
			return l.NextToken()
		}
		if l.peekChar() == '=' {
			l.readChar()
			tok := makeTok(token.DIV_ASSIGN, "/=")
			l.readChar()
			return tok
		}
		tok := makeTok(token.SLASH, string(l.ch))
		l.readChar()
		return tok
	case '%':
		tok := makeTok(token.MOD, string(l.ch))
		l.readChar()
		return tok
	case '!':
		if l.peekChar() == '=' {
			l.readChar()
			tok := makeTok(token.NOT_EQ, "!=")
			l.readChar()
			return tok
		}
		tok := makeTok(token.NOT, string(l.ch))
		l.readChar()
		return tok
	case '<':
		if l.peekChar() == '=' {
			l.readChar()
			tok := makeTok(token.LTE, "<=")
			l.readChar()
			return tok
		}
		tok := makeTok(token.LT, string(l.ch))
		l.readChar()
		return tok
	case '>':
		if l.peekChar() == '=' {
			l.readChar()
			tok := makeTok(token.GTE, ">=")
			l.readChar()
			return tok
		}
		tok := makeTok(token.GT, string(l.ch))
		l.readChar()
		return tok
	case '&':
		if l.peekChar() == '&' {
			l.readChar()
			tok := makeTok(token.AND, "&&")
			l.readChar()
			return tok
		}
		tok := makeTok(token.ILLEGAL, string(l.ch))
		l.readChar()
		return tok
	case '|':
		if l.peekChar() == '|' {
			l.readChar()
			tok := makeTok(token.OR, "||")
			l.readChar()
			return tok
		}
		if l.peekChar() == '>' {
			l.readChar()
			tok := makeTok(token.PIPE, "|>")
			l.readChar()
			return tok
		}
		tok := makeTok(token.ILLEGAL, string(l.ch))
		l.readChar()
		return tok
	case ',':
		tok := makeTok(token.COMMA, string(l.ch))
		l.readChar()
		return tok
	case ';':
		tok := makeTok(token.SEMICOLON, string(l.ch))
		l.readChar()
		return tok
	case ':':
		tok := makeTok(token.COLON, string(l.ch))
		l.readChar()
		return tok
	case '.':
		tok := makeTok(token.DOT, string(l.ch))
		l.readChar()
		return tok
	case '(':
		tok := makeTok(token.LPAREN, string(l.ch))
		l.readChar()
		return tok
	case ')':
		tok := makeTok(token.RPAREN, string(l.ch))
		l.readChar()
		return tok
	case '{':
		tok := makeTok(token.LBRACE, string(l.ch))
		l.readChar()
		return tok
	case '}':
		tok := makeTok(token.RBRACE, string(l.ch))
		l.readChar()
		return tok
	case '[':
		tok := makeTok(token.LBRACKET, string(l.ch))
		l.readChar()
		return tok
	case ']':
		tok := makeTok(token.RBRACKET, string(l.ch))
		l.readChar()
		return tok
	case '"':
		lit := l.readString('"')
		return makeTok(token.STRING, lit)
	case '\'':
		lit := l.readString('\'')
		return makeTok(token.STRING, lit)
	case 0:
		return makeTok(token.EOF, "")
	default:
		if isLetter(l.ch) {
			lit := l.readIdentifier()
			return makeTok(token.LookupIdent(lit), lit)
		}
		if isDigit(l.ch) {
			tt, lit := l.readNumber()
			return makeTok(tt, lit)
		}
		tok := makeTok(token.ILLEGAL, string(l.ch))
		l.readChar()
		return tok
	}
}

func (l *Lexer) skipWhitespace() {
	for l.ch == ' ' || l.ch == '\t' || l.ch == '\n' || l.ch == '\r' {
		l.readChar()
	}
}

func (l *Lexer) skipComment() {
	for l.ch != '\n' && l.ch != 0 {
		l.readChar()
	}
}

func (l *Lexer) skipMultiLineComment() {
	for {
		if l.ch == '*' && l.peekChar() == '/' {
			l.readChar()
			l.readChar()
			return
		}
		if l.ch == 0 {
			return
		}
		l.readChar()
	}
}

func (l *Lexer) readIdentifier() string {
	position := l.position
	for isLetter(l.ch) || isDigit(l.ch) {
		l.readChar()
	}
	return l.input[position:l.position]
}

func (l *Lexer) readNumber() (token.TokenType, string) {
	position := l.position
	var tokType token.TokenType = token.INT

	for isDigit(l.ch) {
		l.readChar()
	}

	if l.ch == '.' && isDigit(l.peekChar()) {
		tokType = token.FLOAT
		l.readChar()
		for isDigit(l.ch) {
			l.readChar()
		}
	}

	return tokType, l.input[position:l.position]
}

// readString 读取字符串字面量（quote 为分隔符，支持转义）
func (l *Lexer) readString(quote byte) string {
	// 起始引号已经消费前的字符是 l.ch == quote
	var result []byte
	// 消费起始引号
	l.readChar()
	for {
		if l.ch == quote || l.ch == 0 {
			l.readChar() // 消费结束引号
			break
		}
		if l.ch == '\\' && l.peekChar() != 0 {
			l.readChar()
			// 简单转义处理
			switch l.ch {
			case 'n':
				result = append(result, '\n')
			case 't':
				result = append(result, '\t')
			case 'r':
				result = append(result, '\r')
			case '\\':
				result = append(result, '\\')
			case '"':
				result = append(result, '"')
			case '\'':
				result = append(result, '\'')
			case '0':
				result = append(result, 0)
			default:
				result = append(result, '\\', l.ch)
			}
			l.readChar()
			continue
		}
		result = append(result, l.ch)
		l.readChar()
	}
	return string(result)
}

func isLetter(ch byte) bool {
	return 'a' <= ch && ch <= 'z' || 'A' <= ch && ch <= 'Z' || ch == '_'
}

func isDigit(ch byte) bool {
	return '0' <= ch && ch <= '9'
}
