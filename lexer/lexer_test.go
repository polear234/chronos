package lexer

import (
	"testing"

	"chronos/token"
)

func TestNextToken(t *testing.T) {
	input := `let five = 5;
let ten = 10;
let add = fn(x, y) {
    x + y;
};
let result = add(five, ten);
!-5 * 5 / 5;
5 < 10 > 5;
if (5 < 10) {
    return true;
} else {
    return false;
}
10 == 10;
10 != 9;
"foobar"
"foo bar"
[1, 2];
{"foo": "bar"};
class Foo { bar() {} }
new Foo();
this.name;
super.init();
x += 1;
x--;
while (true) { break; }
for x in [1,2] { continue; }
try {} catch(e) {} finally {}
match(x) { case 1: {} default: {} }
import "mod";
export { foo };
throw "err";
`

	tests := []struct {
		expectedType    token.TokenType
		expectedLiteral string
	}{
		{token.LET, "let"},
		{token.IDENT, "five"},
		{token.ASSIGN, "="},
		{token.INT, "5"},
		{token.SEMICOLON, ";"},
		{token.LET, "let"},
		{token.IDENT, "ten"},
		{token.ASSIGN, "="},
		{token.INT, "10"},
		{token.SEMICOLON, ";"},
		{token.LET, "let"},
		{token.IDENT, "add"},
		{token.ASSIGN, "="},
		{token.FN, "fn"},
		{token.LPAREN, "("},
		{token.IDENT, "x"},
		{token.COMMA, ","},
		{token.IDENT, "y"},
		{token.RPAREN, ")"},
		{token.LBRACE, "{"},
		{token.IDENT, "x"},
		{token.PLUS, "+"},
		{token.IDENT, "y"},
		{token.SEMICOLON, ";"},
		{token.RBRACE, "}"},
		{token.SEMICOLON, ";"},
		{token.LET, "let"},
		{token.IDENT, "result"},
		{token.ASSIGN, "="},
		{token.IDENT, "add"},
		{token.LPAREN, "("},
		{token.IDENT, "five"},
		{token.COMMA, ","},
		{token.IDENT, "ten"},
		{token.RPAREN, ")"},
		{token.SEMICOLON, ";"},
		{token.NOT, "!"},
		{token.MINUS, "-"},
		{token.INT, "5"},
		{token.ASTERISK, "*"},
		{token.INT, "5"},
		{token.SLASH, "/"},
		{token.INT, "5"},
		{token.SEMICOLON, ";"},
		{token.INT, "5"},
		{token.LT, "<"},
		{token.INT, "10"},
		{token.GT, ">"},
		{token.INT, "5"},
		{token.SEMICOLON, ";"},
		{token.IF, "if"},
		{token.LPAREN, "("},
		{token.INT, "5"},
		{token.LT, "<"},
		{token.INT, "10"},
		{token.RPAREN, ")"},
		{token.LBRACE, "{"},
		{token.RETURN, "return"},
		{token.TRUE, "true"},
		{token.SEMICOLON, ";"},
		{token.RBRACE, "}"},
		{token.ELSE, "else"},
		{token.LBRACE, "{"},
		{token.RETURN, "return"},
		{token.FALSE, "false"},
		{token.SEMICOLON, ";"},
		{token.RBRACE, "}"},
		{token.INT, "10"},
		{token.EQ, "=="},
		{token.INT, "10"},
		{token.SEMICOLON, ";"},
		{token.INT, "10"},
		{token.NOT_EQ, "!="},
		{token.INT, "9"},
		{token.SEMICOLON, ";"},
		{token.STRING, "foobar"},
		{token.STRING, "foo bar"},
		{token.LBRACKET, "["},
		{token.INT, "1"},
		{token.COMMA, ","},
		{token.INT, "2"},
		{token.RBRACKET, "]"},
		{token.SEMICOLON, ";"},
		{token.LBRACE, "{"},
		{token.STRING, "foo"},
		{token.COLON, ":"},
		{token.STRING, "bar"},
		{token.RBRACE, "}"},
		{token.SEMICOLON, ";"},
		{token.CLASS, "class"},
		{token.IDENT, "Foo"},
		{token.LBRACE, "{"},
		{token.IDENT, "bar"},
		{token.LPAREN, "("},
		{token.RPAREN, ")"},
		{token.LBRACE, "{"},
		{token.RBRACE, "}"},
		{token.RBRACE, "}"},
		{token.NEW, "new"},
		{token.IDENT, "Foo"},
		{token.LPAREN, "("},
		{token.RPAREN, ")"},
		{token.SEMICOLON, ";"},
		{token.THIS, "this"},
		{token.DOT, "."},
		{token.IDENT, "name"},
		{token.SEMICOLON, ";"},
		{token.SUPER, "super"},
		{token.DOT, "."},
		{token.IDENT, "init"},
		{token.LPAREN, "("},
		{token.RPAREN, ")"},
		{token.SEMICOLON, ";"},
		{token.IDENT, "x"},
		{token.PLUS_ASSIGN, "+="},
		{token.INT, "1"},
		{token.SEMICOLON, ";"},
		{token.IDENT, "x"},
		{token.DECREMENT, "--"},
		{token.SEMICOLON, ";"},
		{token.WHILE, "while"},
		{token.LPAREN, "("},
		{token.TRUE, "true"},
		{token.RPAREN, ")"},
		{token.LBRACE, "{"},
		{token.BREAK, "break"},
		{token.SEMICOLON, ";"},
		{token.RBRACE, "}"},
		{token.FOR, "for"},
		{token.IDENT, "x"},
		{token.IN, "in"},
		{token.LBRACKET, "["},
		{token.INT, "1"},
		{token.COMMA, ","},
		{token.INT, "2"},
		{token.RBRACKET, "]"},
		{token.LBRACE, "{"},
		{token.CONTINUE, "continue"},
		{token.SEMICOLON, ";"},
		{token.RBRACE, "}"},
		{token.TRY, "try"},
		{token.LBRACE, "{"},
		{token.RBRACE, "}"},
		{token.CATCH, "catch"},
		{token.LPAREN, "("},
		{token.IDENT, "e"},
		{token.RPAREN, ")"},
		{token.LBRACE, "{"},
		{token.RBRACE, "}"},
		{token.FINALLY, "finally"},
		{token.LBRACE, "{"},
		{token.RBRACE, "}"},
		{token.MATCH, "match"},
		{token.LPAREN, "("},
		{token.IDENT, "x"},
		{token.RPAREN, ")"},
		{token.LBRACE, "{"},
		{token.CASE, "case"},
		{token.INT, "1"},
		{token.COLON, ":"},
		{token.LBRACE, "{"},
		{token.RBRACE, "}"},
		{token.DEFAULT, "default"},
		{token.COLON, ":"},
		{token.LBRACE, "{"},
		{token.RBRACE, "}"},
		{token.RBRACE, "}"},
		{token.IMPORT, "import"},
		{token.STRING, "mod"},
		{token.SEMICOLON, ";"},
		{token.EXPORT, "export"},
		{token.LBRACE, "{"},
		{token.IDENT, "foo"},
		{token.RBRACE, "}"},
		{token.SEMICOLON, ";"},
		{token.THROW, "throw"},
		{token.STRING, "err"},
		{token.SEMICOLON, ";"},
		{token.EOF, ""},
	}

	l := New(input)
	for i, tt := range tests {
		tok := l.NextToken()
		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - tokentype wrong. expected=%q, got=%q", i, tt.expectedType, tok.Type)
		}
		if tok.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d] - literal wrong. expected=%q, got=%q", i, tt.expectedLiteral, tok.Literal)
		}
	}
}

func TestPipeToken(t *testing.T) {
	input := `a |> f(b)`

	tests := []struct {
		expectedType    token.TokenType
		expectedLiteral string
	}{
		{token.IDENT, "a"},
		{token.PIPE, "|>"},
		{token.IDENT, "f"},
		{token.LPAREN, "("},
		{token.IDENT, "b"},
		{token.RPAREN, ")"},
		{token.EOF, ""},
	}

	l := New(input)
	for i, tt := range tests {
		tok := l.NextToken()
		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - tokentype wrong. expected=%q, got=%q", i, tt.expectedType, tok.Type)
		}
		if tok.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d] - literal wrong. expected=%q, got=%q", i, tt.expectedLiteral, tok.Literal)
		}
	}
}

func TestPipeVsOr(t *testing.T) {
	// 确保 |> 不会和 || 混淆
	input := `a || b`
	l := New(input)

	tok := l.NextToken()
	if tok.Type != token.IDENT || tok.Literal != "a" {
		t.Fatalf("expected IDENT 'a', got %q %q", tok.Type, tok.Literal)
	}
	tok = l.NextToken()
	if tok.Type != token.OR || tok.Literal != "||" {
		t.Fatalf("expected OR '||', got %q %q", tok.Type, tok.Literal)
	}
}

func TestUnicodeIdentifiers(t *testing.T) {
	// 中文标识符应被识别为 IDENT，而不是逐字节拆成 ILLEGAL
	input := `let 问候语 = "你好"`

	tests := []struct {
		expectedType    token.TokenType
		expectedLiteral string
	}{
		{token.LET, "let"},
		{token.IDENT, "问候语"},
		{token.ASSIGN, "="},
		{token.STRING, "你好"},
		{token.EOF, ""},
	}

	l := New(input)
	for i, tt := range tests {
		tok := l.NextToken()
		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - tokentype wrong. expected=%q, got=%q", i, tt.expectedType, tok.Type)
		}
		if tok.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d] - literal wrong. expected=%q, got=%q", i, tt.expectedLiteral, tok.Literal)
		}
	}
}

func TestUnicodeMixedIdentifier(t *testing.T) {
	// 中英文混排的标识符也应完整识别
	input := `let 水果list2 = 1`
	l := New(input)
	l.NextToken() // let
	tok := l.NextToken()
	if tok.Type != token.IDENT || tok.Literal != "水果list2" {
		t.Fatalf("expected IDENT '水果list2', got %q %q", tok.Type, tok.Literal)
	}
}
